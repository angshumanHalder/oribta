package main

import (
	"context"
	"fmt"
	"orbita/proxy"
)

// App struct
type App struct {
	ctx   context.Context
	proxy *proxy.Proxy
}

// NewApp creates a new App application struct
func NewApp() *App {
	p := proxy.New(nil)
	return &App{
		proxy: p,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	addr, err := a.proxy.Start()
	if err != nil {
		fmt.Println("proxy start error", err)
		return
	}
	fmt.Println("proxy listening on ", addr)
}

// shut down
func (a *App) shutdown(ctx context.Context) {
	a.proxy.Stop()
}

// IPC methods
func (a *App) GetProxyAddr() string {
	return a.proxy.Addr()
}

func (a *App) GetRewriteRules() []proxy.RewriteRule {
	return a.proxy.GetRules()
}

func (a *App) SetRewriteRules(rules []proxy.RewriteRule) {
	a.proxy.SetRules(rules)
}
