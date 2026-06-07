package main

import (
	"context"
	"fmt"
	"orbita/profiles"
	"orbita/proxy"
	"os"
)

// App struct
type App struct {
	ctx   context.Context
	proxy *proxy.Proxy
	store *profiles.ProfileStore
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("no home directory found", err)
		return
	}
	configPath := homeDir + "/.config/orbita/profiles.json"
	store, err := profiles.Load(configPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	a.store = store
	if env := a.store.ActiveEnv(); env != nil {
		a.proxy.SetHeaders(env.Headers)
		a.proxy.SetRules(env.RewriteRules)
	}
}

// shut down
func (a *App) shutdown(ctx context.Context) {
	a.proxy.Stop()
}

// IPC methods
func (a *App) GetProxyAddr() string {
	return a.proxy.Addr()
}

func (a *App) GetRewriteRules() []profiles.RewriteRule {
	return a.proxy.GetRules()
}

func (a *App) SetRewriteRules(rules []profiles.RewriteRule) {
	a.proxy.SetRules(rules)
}

func (a *App) GetEnvironments() []profiles.Environment {
	if a.store == nil {
		return nil
	}
	return a.store.Environments
}

func (a *App) SetActiveEnv(name string) error {
	if a.store == nil {
		return fmt.Errorf("No config found")
	}
	if err := a.store.SetActiveEnv(name); err != nil {
		return err
	}
	if env := a.store.ActiveEnv(); env != nil {
		a.proxy.SetHeaders(env.Headers)
		a.proxy.SetRules(env.RewriteRules)
	}
	return nil
}

func (a *App) GetActiveEnv() *profiles.Environment {
	if a.store == nil {
		return nil
	}
	return a.store.ActiveEnv()
}

func (a *App) AddEnvironment(env profiles.Environment) error {
	if a.store == nil {
		return fmt.Errorf("No config found")
	}
	a.store.Environments = append(a.store.Environments, env)
	return a.store.Save()
}

func (a *App) DeleteEnvironment(name string) error {
	if a.store == nil {
		return nil
	}
	idx := -1
	for i, e := range a.store.Environments {
		if e.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("No env found: %q", name)
	}
	a.store.Environments = append(a.store.Environments[:idx], a.store.Environments[idx+1:]...)
	return a.store.Save()
}

func (a *App) UpdateEnvironment(env profiles.Environment) error {
	if a.store == nil {
		return fmt.Errorf("No config found")
	}
	for i, e := range a.store.Environments {
		if e.Name == env.Name {
			a.store.Environments[i] = env
			if err := a.store.Save(); err != nil {
				return err
			}
			if a.store.Active == env.Name {
				a.proxy.SetHeaders(env.Headers)
				a.proxy.SetRules(env.RewriteRules)
			}
			return nil
		}
	}
	return fmt.Errorf("Environment not found")
}
