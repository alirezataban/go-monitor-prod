package collector

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// ResourceEntry holds data for one network request/response.
type ResourceEntry struct {
	URL          string  `json:"url"`
	Method       string  `json:"method"`
	ResourceType string  `json:"resourceType"`
	Status       int     `json:"status"`
	Failed       bool    `json:"failed"`
	Error        string  `json:"error,omitempty"`
	ContentType  string  `json:"contentType,omitempty"`
	FromCache    bool    `json:"fromCache,omitempty"`
	StartTime    float64 `json:"-"` // relative ms
	Duration     float64 `json:"durationMs,omitempty"`
	TransferSize int64   `json:"transferSize,omitempty"`
	EncodedSize  int64   `json:"encodedSize,omitempty"`
	DecodedSize  int64   `json:"decodedSize,omitempty"`
	Domain       string  `json:"domain,omitempty"`
	IsThirdParty bool    `json:"isThirdParty,omitempty"`
}

// ConsoleEntry is one console message.
type ConsoleEntry struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	URL      string `json:"url,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

// ExceptionEntry is a page JS error.
type ExceptionEntry struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// Tracker captures network, console and exception events for one session.
// Attach before navigation; read after.
type Tracker struct {
	mu         sync.Mutex
	resources  []*ResourceEntry
	byURL      map[string]*ResourceEntry
	console    []ConsoleEntry
	exceptions []ExceptionEntry
	downloads  []map[string]interface{}
	mainHost   string
}

func NewTracker(pageURL string) *Tracker {
	host := ""
	if u, err := url.Parse(pageURL); err == nil {
		host = strings.ToLower(u.Hostname())
	}
	return &Tracker{
		resources:  make([]*ResourceEntry, 0, 128),
		byURL:      make(map[string]*ResourceEntry),
		console:    make([]ConsoleEntry, 0, 32),
		exceptions: make([]ExceptionEntry, 0, 8),
		downloads:  make([]map[string]interface{}, 0),
		mainHost:   host,
	}
}

func (t *Tracker) Attach(page playwright.Page) {
	page.OnRequest(func(req playwright.Request) {
		u := req.URL()
		domain := extractHost(u)
		entry := &ResourceEntry{
			URL:          u,
			Method:       req.Method(),
			ResourceType: req.ResourceType(),
			Domain:       domain,
			IsThirdParty: t.mainHost != "" && domain != "" && domain != t.mainHost && !strings.HasSuffix(domain, "."+t.mainHost),
			StartTime:    float64(time.Now().UnixMilli()),
		}
		t.mu.Lock()
		t.resources = append(t.resources, entry)
		t.byURL[u] = entry
		t.mu.Unlock()
	})

	page.OnResponse(func(resp playwright.Response) {
		u := resp.URL()
		status := resp.Status()
		ct, _ := resp.HeaderValue("content-type")
		t.mu.Lock()
		if e, ok := t.byURL[u]; ok {
			e.Status = status
			e.ContentType = ct
		} else {
			domain := extractHost(u)
			t.resources = append(t.resources, &ResourceEntry{
				URL:          u,
				Status:       status,
				ContentType:  ct,
				ResourceType: resp.Request().ResourceType(),
				Method:       resp.Request().Method(),
				Domain:       domain,
				IsThirdParty: t.mainHost != "" && domain != "" && domain != t.mainHost,
			})
		}
		t.mu.Unlock()
	})

	page.OnRequestFailed(func(req playwright.Request) {
		u := req.URL()
		errMsg := ""
		if f := req.Failure(); f != nil {
			errMsg = f.Error()
		}
		t.mu.Lock()
		if e, ok := t.byURL[u]; ok {
			e.Failed = true
			e.Error = errMsg
		} else {
			domain := extractHost(u)
			t.resources = append(t.resources, &ResourceEntry{
				URL:          u,
				Method:       req.Method(),
				ResourceType: req.ResourceType(),
				Failed:       true,
				Error:        errMsg,
				Domain:       domain,
				IsThirdParty: t.mainHost != "" && domain != "" && domain != t.mainHost,
			})
		}
		t.mu.Unlock()
	})

	page.OnConsole(func(msg playwright.ConsoleMessage) {
		loc := msg.Location()
		t.mu.Lock()
		t.console = append(t.console, ConsoleEntry{
			Type:   msg.Type(),
			Text:   msg.Text(),
			URL:    loc.URL,
			Line:   loc.LineNumber,
			Column: loc.ColumnNumber,
		})
		t.mu.Unlock()
	})

	page.OnPageError(func(err error) {
		msg := err.Error()
		typ := classifyException(msg)
		t.mu.Lock()
		t.exceptions = append(t.exceptions, ExceptionEntry{Message: msg, Type: typ})
		t.mu.Unlock()
	})

	page.OnDownload(func(d playwright.Download) {
		t.mu.Lock()
		t.downloads = append(t.downloads, map[string]interface{}{
			"url":      d.URL(),
			"filename": d.SuggestedFilename(),
		})
		t.mu.Unlock()
	})
}

// EnrichFromPerformance pulls transferSize / duration from PerformanceResourceTiming.
func (t *Tracker) EnrichFromPerformance(page playwright.Page) {
	result, err := page.Evaluate(`() => {
		return performance.getEntriesByType('resource').map(e => ({
			name: e.name,
			duration: Math.round(e.duration),
			transferSize: e.transferSize || 0,
			encodedBodySize: e.encodedBodySize || 0,
			decodedBodySize: e.decodedBodySize || 0
		}));
	}`)
	if err != nil || result == nil {
		return
	}
	list, ok := result.([]interface{})
	if !ok {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if e, ok := t.byURL[name]; ok {
			if d, ok := m["duration"].(float64); ok {
				e.Duration = d
			}
			if s, ok := m["transferSize"].(float64); ok {
				e.TransferSize = int64(s)
			}
			if s, ok := m["encodedBodySize"].(float64); ok {
				e.EncodedSize = int64(s)
			}
			if s, ok := m["decodedBodySize"].(float64); ok {
				e.DecodedSize = int64(s)
			}
		}
	}
}

func (t *Tracker) Resources() []*ResourceEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*ResourceEntry, len(t.resources))
	copy(out, t.resources)
	return out
}

func (t *Tracker) Console() []ConsoleEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]ConsoleEntry, len(t.console))
	copy(out, t.console)
	return out
}

func (t *Tracker) Exceptions() []ExceptionEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]ExceptionEntry, len(t.exceptions))
	copy(out, t.exceptions)
	return out
}

func (t *Tracker) Downloads() []map[string]interface{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]map[string]interface{}, len(t.downloads))
	copy(out, t.downloads)
	return out
}

func (t *Tracker) MainHost() string { return t.mainHost }

func extractHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func classifyException(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "typeerror"):
		return "TypeError"
	case strings.Contains(lower, "referenceerror"):
		return "ReferenceError"
	case strings.Contains(lower, "syntaxerror"):
		return "SyntaxError"
	case strings.Contains(lower, "unhandled") && strings.Contains(lower, "rejection"):
		return "UnhandledPromiseRejection"
	default:
		return "Error"
	}
}

// Helper filters
func filterByType(resources []*ResourceEntry, types ...string) []*ResourceEntry {
	set := map[string]bool{}
	for _, t := range types {
		set[t] = true
	}
	out := make([]*ResourceEntry, 0)
	for _, r := range resources {
		if set[r.ResourceType] {
			out = append(out, r)
		}
	}
	return out
}

func isFailed(r *ResourceEntry) bool {
	return r.Failed || (r.Status >= 400 && r.Status > 0)
}

func isSuccess(r *ResourceEntry) bool {
	return !r.Failed && r.Status > 0 && r.Status < 400
}
