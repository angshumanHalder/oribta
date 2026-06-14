package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"orbita/cdp"
	"orbita/pac"
	"orbita/profiles"
	"orbita/proxy"
	"os"
	"os/exec"
	goruntime "runtime"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	proxy      *proxy.Proxy
	store      *profiles.ProfileStore
	envConfig  *profiles.EnvConfig
	recorder   *cdp.Recorder
	pacDomains []string
	pacAddr    string
	pacMu      sync.Mutex
	mocksPath  string
}

// NewApp creates a new App application struct
func NewApp() *App {
	p := proxy.New(nil)
	return &App{
		proxy:    p,
		recorder: cdp.NewRecorder(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.proxy.SetContext(ctx)
	a.proxy.SetNetworkHook(func(method, url string, status int, body, contentType string) {
		a.recorder.AddNetworkEvent(cdp.NetworkEvent{
			Method:      method,
			URL:         url,
			Status:      status,
			Body:        body,
			ContentType: contentType,
		})
	})

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

	a.mocksPath = homeDir + "/.config/orbita/mocks.json"
	if data, err := os.ReadFile(a.mocksPath); err == nil {
		var mocks []proxy.MockRule
		if err := json.Unmarshal(data, &mocks); err != nil {
			fmt.Println("mocks.json corrupted:", err)
		} else if len(mocks) > 0 {
			a.proxy.SetMocks(mocks)
		}
	}

	a.store = store
	if len(a.store.PACDomains) > 0 {
		a.pacMu.Lock()
		a.pacDomains = a.store.PACDomains
		a.pacMu.Unlock()
	}
	if env := a.store.ActiveEnv(); env != nil {
		a.proxy.SetHeaders(env.Headers)
		a.proxy.SetRules(env.RewriteRules)
	}

	addr, err := a.proxy.Start()
	if err != nil {
		fmt.Println("proxy start error", err)
		return
	}
	pacLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		a.pacAddr = pacLn.Addr().String()
		mux := http.NewServeMux()
		mux.HandleFunc("/proxy.pac", func(w http.ResponseWriter, r *http.Request) {
			a.pacMu.Lock()
			domains := a.pacDomains
			a.pacMu.Unlock()
			w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
			fmt.Fprintf(w, pac.Generate(domains, a.proxy.Addr()))
		})
		go http.Serve(pacLn, mux)
	}
	fmt.Println("proxy listening on ", addr)
	fmt.Println("PAC server ", a.pacAddr)

	if goruntime.GOOS == "darwin" {
		go func() {
			check := exec.Command("security", "verify-cert", "-c", homeDir+"/.config/orbita/ca.crt")
			if check.Run() != nil {
				exec.Command("osascript", "-e", `do shell script "security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain `+homeDir+`/.config/orbita/ca.crt" with administrator privileges`).Run()
			}
		}()
	}
}

// shut down
func (a *App) shutdown(ctx context.Context) {
	a.proxy.Stop()
	a.recorder.Close()
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
		return []profiles.Environment{}
	}
	if a.store.Environments == nil {
		return []profiles.Environment{}
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
	if a.mocksPath != "" {
		if data, err := json.Marshal(mocks); err == nil {
			if err := os.WriteFile(a.mocksPath, data, 0644); err != nil {
				fmt.Println("failed to save mocks:", err)
			}
		}
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "mocks-updated", mocks)
	}
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
			"--proxy-pac-url=http://"+a.pacAddr+"/proxy.pac",
			"--disable-quic",
			"--remote-debugging-port=9222",
			"--user-data-dir=/tmp/orbita-chrome",
		)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "chrome", "--proxy-pac-url=http://"+a.pacAddr+"/proxy.pac", "--user-data-dir=%TEMP%\\oribta-chrome")
	case "linux":
		cmd = exec.Command("google-chrome", "--proxy-pac-url=http://"+a.pacAddr+"/proxy.pac", "--user-data-dir=/tmp/orbita-chrome")
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
	domains := extractDomains(cfg)
	a.pacMu.Lock()
	defer a.pacMu.Unlock()
	for _, d := range domains {
		found := false
		for _, existing := range a.pacDomains {
			if d == existing {
				found = true
				break
			}
		}
		if !found {
			a.pacDomains = append(a.pacDomains, d)
		}
	}
	if a.store != nil {
		a.store.PACDomains = a.pacDomains
		a.store.Save()
	}
	return nil
}

func (a *App) GetEnvConfigNames() []string {
	if a.envConfig == nil {
		return []string{}
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

func (a *App) StartRecording() error {
	return a.recorder.Start()
}

func (a *App) StopRecording() string {
	session := a.recorder.Stop()
	return cdp.GeneratePlaywright(session)
}

func (a *App) GetPACAddr() string {
	return a.pacAddr
}

func (a *App) GetPACDomains() []string {
	a.pacMu.Lock()
	defer a.pacMu.Unlock()
	if a.pacDomains == nil {
		return []string{}
	}
	return a.pacDomains
}

func (a *App) AddPACDomain(domain string) {
	a.pacMu.Lock()
	defer a.pacMu.Unlock()
	for _, d := range a.pacDomains {
		if d == domain {
			return
		}
	}
	a.pacDomains = append(a.pacDomains, domain)
	if a.store != nil {
		a.store.PACDomains = a.pacDomains
		a.store.Save()
	}
}

func (a *App) RemovePACDomain(domain string) {
	a.pacMu.Lock()
	defer a.pacMu.Unlock()
	for i, d := range a.pacDomains {
		if d == domain {
			a.pacDomains = append(a.pacDomains[:i], a.pacDomains[i+1:]...)
			if a.store != nil {
				a.store.PACDomains = a.pacDomains
				a.store.Save()
			}
			return
		}
	}
}

func extractDomains(cfg *profiles.EnvConfig) []string {
	seen := map[string]bool{}
	var domains []string
	add := func(rawURL string) {
		u, err := url.Parse(rawURL)
		if err == nil && u.Host != "" {
			host := u.Hostname()
			if !seen[host] {
				seen[host] = true
				domains = append(domains, host)
			}
		}
	}
	for _, env := range cfg.Environments {
		for _, u := range env.URLs {
			add(u)
		}
	}
	return domains
}
