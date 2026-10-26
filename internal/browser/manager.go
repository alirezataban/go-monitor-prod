package browser

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/alirezataban/monitor-engine-go/internal/config"
)

// Manager keeps a single long-lived browser.
// Every job gets a brand-new BrowserContext (isolated, no cache/cookies/storage).
type Manager struct {
	cfg     *config.Config
	pw      *playwright.Playwright
	browser playwright.Browser
	mu      sync.Mutex
	sem     chan struct{} // concurrency limiter
	ready   bool
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg: cfg,
		sem: make(chan struct{}, cfg.MaxConcurrent),
	}
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("playwright run: %w", err)
	}
	m.pw = pw

	opts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(m.cfg.Headless),
		Args: []string{
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-extensions",
			"--disable-background-networking",
			"--disable-default-apps",
			"--disable-sync",
			"--disable-translate",
			"--metrics-recording-only",
			"--mute-audio",
			"--no-first-run",
			"--safebrowsing-disable-auto-update",
			"--disable-component-update",
			"--disable-domain-reliability",
			"--disable-features=AudioServiceOutOfProcess,IsolateOrigins,site-per-process",
			"--disk-cache-size=0",
			"--media-cache-size=0",
			"--disable-application-cache",
			"--disable-cache",
		},
	}
	if m.cfg.ChromePath != "" {
		opts.ExecutablePath = playwright.String(m.cfg.ChromePath)
	}

	browser, err := pw.Chromium.Launch(opts)
	if err != nil {
		_ = pw.Stop()
		return fmt.Errorf("launch browser: %w", err)
	}
	m.browser = browser
	m.ready = true
	log.Println("[browser] started (single shared instance)")
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.browser != nil {
		_ = m.browser.Close()
		m.browser = nil
	}
	if m.pw != nil {
		_ = m.pw.Stop()
		m.pw = nil
	}
	m.ready = false
	log.Println("[browser] stopped")
}

func (m *Manager) Ready() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ready && m.browser != nil
}

// Session is an isolated context+page for one monitoring job.
// Always call Close() when done (preferably with defer).
type Session struct {
	Context playwright.BrowserContext
	Page    playwright.Page
	mgr     *Manager
}

func (s *Session) Close() {
	if s.Page != nil {
		_ = s.Page.Close()
	}
	if s.Context != nil {
		_ = s.Context.Close()
	}
	// release concurrency slot
	select {
	case <-s.mgr.sem:
	default:
	}
}

// NewSession creates a completely fresh context (no shared cookies, cache, storage).
func (m *Manager) NewSession(timeout time.Duration, userAgent string, ignoreHTTPS bool, vw, vh int) (*Session, error) {
	if !m.Ready() {
		return nil, fmt.Errorf("browser not ready")
	}

	// limit concurrent sessions
	select {
	case m.sem <- struct{}{}:
	case <-time.After(timeout):
		return nil, fmt.Errorf("too many concurrent checks (max %d)", m.cfg.MaxConcurrent)
	}

	m.mu.Lock()
	browser := m.browser
	m.mu.Unlock()

	if userAgent == "" {
		userAgent = m.cfg.UserAgent
	}
	if vw <= 0 {
		vw = 1920
	}
	if vh <= 0 {
		vh = 1080
	}

	ctxOpts := playwright.BrowserNewContextOptions{
		UserAgent:         playwright.String(userAgent),
		IgnoreHttpsErrors: playwright.Bool(ignoreHTTPS),
		Viewport: &playwright.Size{
			Width:  vw,
			Height: vh,
		},
		// Force offline cache off / no service worker persistence across jobs
		JavaScriptEnabled: playwright.Bool(true),
		BypassCSP:         playwright.Bool(false),
	}

	context, err := browser.NewContext(ctxOpts)
	if err != nil {
		<-m.sem
		return nil, fmt.Errorf("new context: %w", err)
	}

	// Extra isolation: clear any residual permissions / geolocation
	_ = context.ClearCookies()

	page, err := context.NewPage()
	if err != nil {
		_ = context.Close()
		<-m.sem
		return nil, fmt.Errorf("new page: %w", err)
	}

	page.SetDefaultTimeout(float64(timeout.Milliseconds()))
	page.SetDefaultNavigationTimeout(float64(timeout.Milliseconds()))

	return &Session{
		Context: context,
		Page:    page,
		mgr:     m,
	}, nil
}
