package main

import (
	"context"
	"fmt"
	"orbita/profiles"
	"orbita/proxy"
	"os"
	"os/exec"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx       context.Context
	proxy     *proxy.Proxy
	store     *profiles.ProfileStore
	envConfig *profiles.EnvConfig
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
	a.proxy.SetContext(ctx)

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
	ca, err := proxy.LoadOrGenerate(homeDir+"/.config/orbita/ca.crt", homeDir+"/.config/orbita/ca.key")
	if err != nil {
		fmt.Println("CA init error")
		return
	}
	a.proxy.SetCA(ca)

	if goruntime.GOOS == "darwin" {
		check := exec.Command("security", "verify-cert", "-c", homeDir+"/.config/orbita/ca.crt")
		if check.Run() != nil {
			exec.Command("osascript", "-e", `do shell script "security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain `+homeDir+`/.config/orbita/ca.crt" with administrator privileges`).Run()
		}
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

func (a *App) GetMocks() []proxy.MockRule {
	return a.proxy.GetMock()
}

func (a *App) SetMocks(mocks []proxy.MockRule) {
	a.proxy.SetMocks(mocks)
}

func (a *App) OpenInChrome() error {
	addr := a.proxy.Addr()
	if addr == "" {
		return fmt.Errorf("proxy not running")
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command(
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"--proxy-server="+addr,
			"--disable-quic",
			"--remote-debugging-port=9222",
			"--user-data-dir=/tmp/orbita-chrome",
		)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "chrome", "--proxy-server="+addr, "--user-data-dir=%TEMP%\\oribta-chrome")
	case "linux":
		cmd = exec.Command("google-chrome", "--proxy-server="+addr, "--user-data-dir=/tmp/orbita-chrome")
	default:
		return fmt.Errorf("unsupported OS: %s", goruntime.GOOS)
	}
	return cmd.Start()
}

func (a *App) ImportEnvConfig(path string) error {
	cfg, err := profiles.ParseEnvConfig(path)
	if err != nil {
		return err
	}
	a.envConfig = cfg
	return nil
}

func (a *App) GetEnvConfigNames() []string {
	if a.envConfig == nil {
		return nil
	}
	names := make([]string, 0, len(a.envConfig.Environments))
	for name := range a.envConfig.Environments {
		names = append(names, name)
	}
	return names
}

func (a *App) ApplyEnvMapping(fromEnv, toEnv string) error {
	if a.envConfig == nil {
		return fmt.Errorf("no env config loaded")
	}
	from := a.envConfig.Environments[fromEnv].URLs
	to := a.envConfig.Environments[toEnv].URLs
	existing := a.proxy.GetRules()
	for key, fromURL := range from {
		if toURL, ok := to[key]; ok && fromURL != "" && toURL != "" {
			found := false
			for i, r := range existing {
				if r.From == fromURL {
					existing[i].To = toURL
					found = true
					break
				}
			}
			if !found {
				existing = append(existing, profiles.RewriteRule{From: fromURL, To: toURL})
			}
		}
	}
	a.proxy.SetRules(existing)
	if a.store != nil {
		if env := a.store.ActiveEnv(); env != nil {
			env.RewriteRules = existing
			if err := a.store.Save(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) OpenFilePicker() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Environment Config",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})
	return path, err
}
