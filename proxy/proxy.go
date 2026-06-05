package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
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

type RewriteRule struct {
	From string
	To   string
}

type Proxy struct {
	server *http.Server
	ln     net.Listener
	rules  []RewriteRule
	mu     sync.RWMutex
}

func New(rules []RewriteRule) *Proxy {
	return &Proxy{
		rules: rules,
	}
}

func (p *Proxy) Start() (string, error) {
	mux := http.NewServeMux()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	p.server = &http.Server{Handler: mux}
	p.ln = ln

	mux.HandleFunc("/", p.handle)

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

func (p *Proxy) SetRules(rules []RewriteRule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rules = rules
}
func (p *Proxy) GetRules() []RewriteRule {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.rules
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
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

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
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
	outReq.Host = targetURL.Host
	res, err := http.DefaultTransport.RoundTrip(outReq)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	copyHeaders(w.Header(), res.Header)
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
	serverConn, _, err := websocket.DefaultDialer.Dial(targetURL.String(), fwdHeaders)
	if err != nil {
		clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "upstream failed"))
		return
	}
	defer serverConn.Close()
	errc := make(chan error, 2)
	relay := func(dst, src *websocket.Conn) {
		for {
			t, msg, err := src.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := dst.WriteMessage(t, msg); err != nil {
				errc <- err
				return
			}
		}
	}
	go relay(serverConn, clientConn)
	go relay(clientConn, serverConn)
	<-errc
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
