package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"orbita/profiles"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var hopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailers":            true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

type Proxy struct {
	server    *http.Server
	ln        net.Listener
	rules     []profiles.RewriteRule
	headers   map[string]string
	mocks     []MockRule
	context   context.Context
	ca        *CA
	onNetwork func(method, url string, status int, body, contentType string)
	mu        sync.RWMutex
}

type MockRule struct {
	Method  string
	Path    string
	Body    string
	Enabled bool
	Status  int
}

type WSFrame struct {
	URL       string
	Direction string // send or recv
	MsgType   int
	Payload   string
}

type LogEntry struct {
	Method      string
	Path        string
	Status      int
	Latency     int64
	Mocked      bool
	ContentType string
}

type connResponseWriter struct {
	conn       *tls.Conn
	header     http.Header
	statusCode int
	body       []byte
}

func New(rules []profiles.RewriteRule) *Proxy {
	return &Proxy{
		rules: rules,
	}
}

func (p *Proxy) Start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	p.server = &http.Server{Handler: http.HandlerFunc(p.handle)}
	p.ln = ln

	go p.server.Serve(ln)
	return ln.Addr().String(), nil
}

func (p *Proxy) Stop() error {
	return p.server.Shutdown(context.Background())
}

func (p *Proxy) Addr() string {
	if p.ln == nil {
		return ""
	}
	return p.ln.Addr().String()
}

func (p *Proxy) SetRules(rules []profiles.RewriteRule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rules = rules
}

func (p *Proxy) GetRules() []profiles.RewriteRule {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.rules
}

func (p *Proxy) SetHeaders(headers map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.headers = headers
}

func (p *Proxy) SetMocks(mocks []MockRule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mocks = mocks
}

func (p *Proxy) GetMock() []MockRule {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mocks
}

func (p *Proxy) SetContext(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.context = ctx
}

func (p *Proxy) SetCA(ca *CA) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ca = ca
}

func (p *Proxy) SetNetworkHook(fn func(method, url string, status int, body, contentType string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onNetwork = fn
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleCONNECT(w, r)
		return
	}
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		p.handleWS(w, r)
		return
	}
	p.handleHTTP(w, r)
}

func (p *Proxy) rewrite(rawURL string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, r := range p.rules {
		if strings.HasPrefix(rawURL, r.From) {
			return r.To + strings.TrimPrefix(rawURL, r.From)
		}
	}
	return rawURL
}

func (p *Proxy) handleCONNECT(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer clientConn.Close()

	p.mu.RLock()
	ca := p.ca
	p.mu.RUnlock()
	if ca == nil {
		return
	}
	leaftCert, err := ca.GenerateLeafCert(r.Host)
	if err != nil {
		return
	}
	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{leaftCert},
	})
	defer tlsConn.Close()
	if err := tlsConn.Handshake(); err != nil {
		return

	}
	reader := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return
		}
		req.URL.Scheme = "https"
		req.URL.Host = r.Host
		rw := newConnResponseWriter(tlsConn)
		p.handleHTTP(rw, req)
		if err := rw.flush(); err != nil {
			return
		}
	}

}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	targetRaw := r.URL.String()
	if !r.URL.IsAbs() {
		targetRaw = "http://" + r.Host + r.RequestURI
	}
	target := p.rewrite(targetRaw)
	targetURL, err := url.Parse(target)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	copyHeaders(outReq.Header, r.Header)

	p.mu.RLock()
	defer p.mu.RUnlock()

	if r.Method == http.MethodOptions {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for _, m := range p.mocks {
		if m.Enabled && strings.EqualFold(m.Method, r.Method) && m.Path == targetURL.Path {
			timeEnd := time.Since(startTime).Milliseconds()
			runtime.EventsEmit(p.context, "request-log", LogEntry{
				Method:      m.Method,
				Path:        targetURL.RequestURI(),
				Status:      m.Status,
				Latency:     timeEnd,
				Mocked:      m.Enabled,
				ContentType: "application/json",
			})
			w.Header().Set("Content-Type", "application/json")
			if origin := r.Header.Get("Origin"); origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			w.WriteHeader(m.Status)
			w.Write([]byte(m.Body))
			return
		}
	}

	for k, v := range p.headers {
		outReq.Header.Set(k, v)
	}

	outReq.Host = targetURL.Host
	res, err := http.DefaultTransport.RoundTrip(outReq)

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	res.Header.Del("Access-Control-Allow-Origin")
	res.Header.Del("Access-Control-Allow-Credentials")
	res.Header.Del("Access-Control-Allow-Methods")
	res.Header.Del("Access-Control-Allow-Headers")
	runtime.EventsEmit(p.context, "request-log", LogEntry{
		Method:      r.Method,
		Path:        targetURL.RequestURI(),
		Status:      res.StatusCode,
		Latency:     time.Since(startTime).Milliseconds(),
		Mocked:      false,
		ContentType: res.Header.Get("Content-Type"),
	})

	copyHeaders(w.Header(), res.Header)

	var bodyStr string
	contentType := res.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body, err := io.ReadAll(res.Body)
		if err == nil {
			res.Body = io.NopCloser(bytes.NewReader(body))
			bodyStr = string(body)
		}
	}

	if p.onNetwork != nil {
		p.onNetwork(r.Method, targetURL.String(), res.StatusCode, bodyStr, contentType)
	}

	// CORS override
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
}

func (p *Proxy) handleWS(w http.ResponseWriter, r *http.Request) {
	targetRaw := "ws://" + r.Host + r.RequestURI
	target := p.rewrite(targetRaw)
	target = strings.ReplaceAll(target, "https://", "wss://")
	target = strings.ReplaceAll(target, "http://", "ws://")
	targetURL, err := url.Parse(target)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()
	fwdHeaders := make(http.Header)
	for k, v := range r.Header {
		switch strings.ToLower(k) {
		case "upgrade", "connection", "sec-websocket-key", "sec-websocket-version", "sec-websocket-extensions":
		default:
			fwdHeaders[k] = v
		}
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	for k, v := range p.headers {
		switch strings.ToLower(k) {
		case "upgrade", "connection", "sec-websocket-key", "sec-websocket-version", "sec-websocket-extensions":
		default:
			fwdHeaders[k] = []string{v}
		}
	}
	serverConn, _, err := websocket.DefaultDialer.Dial(targetURL.String(), fwdHeaders)
	if err != nil {
		clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "upstream failed"))
		return
	}
	defer serverConn.Close()
	errc := make(chan error, 2)
	relay := func(dst, src *websocket.Conn, direction string) {
		for {
			t, msg, err := src.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			runtime.EventsEmit(p.context, "ws-frames", WSFrame{
				URL:       targetURL.String(),
				Direction: direction,
				MsgType:   t,
				Payload:   string(msg),
			})
			if err := dst.WriteMessage(t, msg); err != nil {
				errc <- err
				return
			}
		}
	}
	go relay(serverConn, clientConn, "send")
	go relay(clientConn, serverConn, "recv")
	<-errc
}

func newConnResponseWriter(conn *tls.Conn) *connResponseWriter {
	return &connResponseWriter{conn: conn, header: make(http.Header), statusCode: 200}
}

func (rw *connResponseWriter) Header() http.Header { return rw.header }

func (rw *connResponseWriter) WriteHeader(status int) {
	rw.statusCode = status
}

func (rw *connResponseWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return len(b), nil
}

func (rw *connResponseWriter) flush() error {
	rw.header.Set("Content-Length", strconv.Itoa(len(rw.body)))
	if _, err := fmt.Fprintf(rw.conn, "HTTP/1.1 %d %s\r\n", rw.statusCode, http.StatusText(rw.statusCode)); err != nil {
		return err
	}
	if err := rw.header.Write(rw.conn); err != nil {
		return err
	}
	fmt.Fprintf(rw.conn, "\r\n")
	_, err := rw.conn.Write(rw.body)
	return err
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		if !hopHeaders[k] {
			for _, v := range vv {
				dst.Add(k, v)
			}
		}
	}
}
