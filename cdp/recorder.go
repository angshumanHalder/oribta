package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type NavigationEvent struct {
	URL string
}

type ClickEvent struct {
	Selector string
	X        float64
	Y        float64
}

type NetworkEvent struct {
	Method      string
	URL         string
	Status      int
	Body        string
	ContentType string
}

type InputEvent struct {
	Selector string
	Value    string
	Masked   bool
}

type EventType string

const (
	EventNavigation EventType = "navigation"
	EventClick      EventType = "click"
	EventInput      EventType = "input"
	EventNetwork    EventType = "network"
)

const jsScript = `(function() {
	console.log('__orbita__' + JSON.stringify({ type: 'navigation', url: window.location.href }));
	var _push = history.pushState.bind(history)
		history.pushState = function(state, title, url) {
		_push(state, title, url);
		console.log('__orbita__' + JSON.stringify({ type: 'navigation', url: url || window.location.href }));
	}
	window.addEventListener('popstate', function() {
    	console.log('__orbita__' + JSON.stringify({ type: 'navigation', url: window.location.href }));
	})
	function sel(el) {
		if (el.id) return '#' + el.id;
		if (el.dataset && el.dataset.testid) return '[data-testid="' + el.dataset.testid + '"]';
		if (el.name) return '[name="' + el.name + '"]';
		return el.tagName.toLowerCase();
	}
	document.addEventListener('click', function(e) {
		console.log('__orbita__' + JSON.stringify({ type: 'click', selector: sel(e.target), x: e.clientX, y: e.clientY }));
	});
	document.addEventListener('change', function(e) {
		let masked = e.target.type === 'password';
		console.log('__orbita__' + JSON.stringify({ type: 'input', selector: sel(e.target), value: masked ? '' : e.target.value, masked: masked }));
	});
}())`

type Event struct {
	Type       EventType
	Timestamp  int64
	Navigation *NavigationEvent
	Click      *ClickEvent
	Input      *InputEvent
	Network    *NetworkEvent
}

type RecordSession struct {
	Events []Event
	mu     sync.RWMutex
}

type Recorder struct {
	session     *RecordSession
	cancel      context.CancelFunc
	allocCancel context.CancelFunc
	mu          sync.RWMutex
}

func NewRecorder() *Recorder {
	return &Recorder{}
}

func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.session != nil {
		return nil
	}
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), "ws://localhost:9222")
	r.allocCancel = allocCancel

	var tabID string
	if resp, err := http.Get("http://localhost:9222/json"); err == nil {
		var tabs []struct {
			ID   string `json:"id"`
			URL  string `json:"url"`
			Type string `json:"type"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
			fmt.Println("cdp: failed to decode tabs:", err)
		}
		resp.Body.Close()
		for _, t := range tabs {
			if t.Type == "page" && t.URL != "" && t.URL != "about:blank" {
				tabID = t.ID
				break
			}
		}
	}
	r.session = &RecordSession{}

	var ctx context.Context
	var cancel context.CancelFunc
	if tabID != "" {
		ctx, cancel = chromedp.NewContext(allocCtx, chromedp.WithTargetID(target.ID(tabID)))
	} else {
		ctx, cancel = chromedp.NewContext(allocCtx)
	}
	r.cancel = cancel

	chromedp.ListenTarget(ctx, func(ev any) {
		r.mu.RLock()
		sess := r.session
		r.mu.RUnlock()
		if sess == nil {
			return
		}
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			r.handleConsoleEvent(sess, e)
		}
	})

	go func() {
		if err := chromedp.Run(ctx,
			runtime.Enable(),
			chromedp.ActionFunc(func(ctx context.Context) error {
				_, err := page.AddScriptToEvaluateOnNewDocument(jsScript).Do(ctx)
				return err
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				_, _, err := runtime.Evaluate(jsScript).Do(ctx)
				return err
			}),
		); err != nil {
			fmt.Println("cdp run error:", err)
		}
	}()
	return nil
}

func (r *Recorder) Stop() *RecordSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.session
	r.session = nil
	return s
}

func (r *Recorder) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.allocCancel != nil {
		r.allocCancel()
		r.allocCancel = nil
	}
	r.session = nil
}

func (r *Recorder) AddNetworkEvent(e NetworkEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.session == nil {
		return
	}
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	r.session.Events = append(r.session.Events, Event{
		Type:      EventNetwork,
		Timestamp: time.Now().UnixMilli(),
		Network:   &e,
	})
}

func (r *Recorder) handleConsoleEvent(sess *RecordSession, e *runtime.EventConsoleAPICalled) {
	if len(e.Args) == 0 {
		return
	}
	raw := strings.Trim(string(e.Args[0].Value), `"`)
	raw = strings.ReplaceAll(raw, `\"`, `"`) // unescape JSON string
	if !strings.HasPrefix(raw, "__orbita__") {
		return
	}
	payload := strings.TrimPrefix(raw, "__orbita__")

	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return
	}

	typ, _ := m["type"].(string)
	sess.mu.Lock()
	defer sess.mu.Unlock()

	switch typ {
	case "navigation":
		url, _ := m["url"].(string)
		sess.Events = append(sess.Events, Event{
			Type:       EventNavigation,
			Timestamp:  time.Now().UnixMilli(),
			Navigation: &NavigationEvent{URL: url},
		})
	case "click":
		selector, _ := m["selector"].(string)
		x, _ := m["x"].(float64)
		y, _ := m["y"].(float64)
		sess.Events = append(sess.Events, Event{
			Type:      EventClick,
			Timestamp: time.Now().UnixMilli(),
			Click:     &ClickEvent{Selector: selector, X: x, Y: y},
		})
	case "input":
		selector, _ := m["selector"].(string)
		value, _ := m["value"].(string)
		masked, _ := m["masked"].(bool)
		sess.Events = append(sess.Events, Event{
			Type:      EventInput,
			Timestamp: time.Now().UnixMilli(),
			Input:     &InputEvent{Selector: selector, Value: value, Masked: masked},
		})
	}
}
