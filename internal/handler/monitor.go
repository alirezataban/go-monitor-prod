package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/alirezataban/monitor-engine-go/internal/browser"
	"github.com/alirezataban/monitor-engine-go/internal/collector"
	"github.com/alirezataban/monitor-engine-go/internal/config"
	"github.com/alirezataban/monitor-engine-go/internal/report"
)

type MonitorHandler struct {
	mgr      *browser.Manager
	cfg      *config.Config
	registry *collector.Registry
}

func NewMonitorHandler(mgr *browser.Manager, cfg *config.Config) *MonitorHandler {
	return NewMonitorHandlerWithRegistry(mgr, cfg, collector.DefaultRegistry())
}

func NewMonitorHandlerWithRegistry(mgr *browser.Manager, cfg *config.Config, registry *collector.Registry) *MonitorHandler {
	return &MonitorHandler{mgr: mgr, cfg: cfg, registry: registry}
}

type MonitorRequest struct {
	URL         string   `json:"url"`
	Domain      string   `json:"domain"`
	Collectors  []string `json:"collectors"` // package names
	TimeoutSec  int      `json:"timeout"`
	IgnoreHTTPS bool     `json:"insecure"`
	UserAgent   string   `json:"userAgent"`
	ViewportW   int      `json:"viewportWidth"`
	ViewportH   int      `json:"viewportHeight"`
	WaitUntil   string   `json:"waitUntil"`
}

func (h *MonitorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req, err := h.parseRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	rep := h.run(req)
	writeJSON(w, http.StatusOK, rep)
}

func (h *MonitorHandler) parseRequest(r *http.Request) (*MonitorRequest, error) {
	req := &MonitorRequest{
		TimeoutSec: int(h.cfg.DefaultTimeout.Seconds()),
		WaitUntil:  "load",
	}

	if r.Method == http.MethodPost && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			return nil, fmt.Errorf("invalid json body: %w", err)
		}
	}

	q := r.URL.Query()
	if v := q.Get("url"); v != "" {
		req.URL = v
	}
	if v := q.Get("domain"); v != "" {
		req.Domain = v
	}
	if v := q.Get("collectors"); v != "" {
		req.Collectors = splitCSV(v)
	}
	// alias: packages=
	if v := q.Get("packages"); v != "" {
		req.Collectors = splitCSV(v)
	}
	if v := q.Get("timeout"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			req.TimeoutSec = n
		}
	}
	if v := q.Get("insecure"); v == "1" || v == "true" {
		req.IgnoreHTTPS = true
	}
	if v := q.Get("ua"); v != "" {
		req.UserAgent = v
	}
	if v := q.Get("wait"); v != "" {
		req.WaitUntil = v
	}

	if req.URL == "" && req.Domain == "" {
		return nil, fmt.Errorf("url or domain is required")
	}
	if req.URL == "" {
		d := strings.TrimSpace(req.Domain)
		d = strings.TrimPrefix(d, "https://")
		d = strings.TrimPrefix(d, "http://")
		req.URL = "https://" + d
	}

	if len(req.Collectors) == 0 {
		req.Collectors = collector.DefaultPackages
	}

	if req.TimeoutSec <= 0 {
		req.TimeoutSec = int(h.cfg.DefaultTimeout.Seconds())
	}
	if req.TimeoutSec > 120 {
		req.TimeoutSec = 120
	}

	return req, nil
}

func (h *MonitorHandler) run(req *MonitorRequest) *report.Report {
	rep := report.New(req.URL)
	timeout := time.Duration(req.TimeoutSec) * time.Second

	session, err := h.mgr.NewSession(timeout, req.UserAgent, req.IgnoreHTTPS, req.ViewportW, req.ViewportH)
	if err != nil {
		rep.Finish(err)
		return rep
	}
	defer session.Close()

	page := session.Page
	tracker := collector.NewTracker(req.URL)
	tracker.Attach(page)

	waitUntil := waitUntilFromString(req.WaitUntil)
	resp, err := page.Goto(req.URL, playwright.PageGotoOptions{
		WaitUntil: waitUntil,
		Timeout:   playwright.Float(float64(timeout.Milliseconds())),
	})
	if err != nil {
		fatalErr := navigationError(err)
		rep.Finish(fatalErr)
		return rep
	}

	time.Sleep(350 * time.Millisecond)
	tracker.EnrichFromPerformance(page)

	ctx := &collector.Context{
		Page:     page,
		Response: resp,
		URL:      req.URL,
		Tracker:  tracker,
	}

	h.collect(rep, ctx, req.Collectors, nil)

	rep.Finish(nil)
	return rep
}

func navigationError(err error) error {
	if errors.Is(err, playwright.ErrTimeout) {
		return errors.New("Site timed out")
	}
	if strings.Contains(strings.ToUpper(err.Error()), "CONNECTION_REFUSED") {
		return errors.New("Connection refused")
	}
	if strings.Contains(strings.ToUpper(err.Error()), "ERR_NAME_NOT_RESOLVED") {
		return errors.New("Site name could not be resolved")
	}
	return fmt.Errorf("navigation: %w", err)
}

func (h *MonitorHandler) collect(rep *report.Report, ctx *collector.Context, names []string, allowed map[string]bool) {
	for _, name := range names {
		if allowed != nil && !allowed[name] {
			continue
		}
		start := time.Now()
		col, ok := h.registry.Get(name)
		if !ok {
			rep.Add(name, false, 0, nil, fmt.Errorf("unknown package: %s", name))
			continue
		}
		data, colErr := col.Collect(ctx)
		rep.Add(name, colErr == nil, time.Since(start), data, colErr)
	}
}

func waitUntilFromString(s string) *playwright.WaitUntilState {
	switch s {
	case "domcontentloaded":
		return playwright.WaitUntilStateDomcontentloaded
	case "networkidle":
		return playwright.WaitUntilStateNetworkidle
	default:
		return playwright.WaitUntilStateLoad
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

type HealthHandler struct {
	mgr *browser.Manager
}

func NewHealthHandler(mgr *browser.Manager) *HealthHandler {
	return &HealthHandler{mgr: mgr}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ready := h.mgr.Ready()
	status := http.StatusOK
	body := map[string]interface{}{"status": "ok", "browser": "ready"}
	if !ready {
		status = http.StatusServiceUnavailable
		body = map[string]interface{}{"status": "unavailable", "browser": "not ready"}
	}
	writeJSON(w, status, body)
}

func CollectorsHandler(w http.ResponseWriter, r *http.Request) {
	CollectorsHandlerWithRegistry(w, r, collector.DefaultRegistry())
}

func CollectorsHandlerWithRegistry(w http.ResponseWriter, r *http.Request, registry *collector.Registry) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"packages":    registry.Names(),
		"default":     collector.DefaultPackages,
		"description": registry.Descriptions(),
	})
}
