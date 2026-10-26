package collector

import (
	"encoding/json"
	"log"
	"net/url"
	"strconv"
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
	Type   string `json:"type"`
	Text   string `json:"text"`
	URL    string `json:"url,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
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
			entry := &ResourceEntry{
				URL:          u,
				Status:       status,
				ContentType:  ct,
				ResourceType: resp.Request().ResourceType(),
				Method:       resp.Request().Method(),
				Domain:       domain,
				IsThirdParty: t.mainHost != "" && domain != "" && domain != t.mainHost && !strings.HasSuffix(domain, "."+t.mainHost),
			}
			t.resources = append(t.resources, entry)
			t.byURL[u] = entry
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
			entry := &ResourceEntry{
				URL:          u,
				Method:       req.Method(),
				ResourceType: req.ResourceType(),
				Failed:       true,
				Error:        errMsg,
				Domain:       domain,
				IsThirdParty: t.mainHost != "" && domain != "" && domain != t.mainHost && !strings.HasSuffix(domain, "."+t.mainHost),
			}
			t.resources = append(t.resources, entry)
			t.byURL[u] = entry
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

// EnrichFromPerformance copies browser PerformanceResourceTiming metrics into
// the same ResourceEntry objects created by Playwright request/response events.
// Call this after navigation and after the page has had time to load resources.
func (t *Tracker) EnrichFromPerformance(page playwright.Page) {
	result, err := page.Evaluate(`() => performance.getEntriesByType('resource').map(e => ({
		name: e.name,
		duration: e.duration,
		transferSize: e.transferSize,
		encodedBodySize: e.encodedBodySize,
		decodedBodySize: e.decodedBodySize
	}))`)
	if err != nil {
		log.Printf("[PERF DEBUG] evaluate error: %v", err)
		return
	}
	if result == nil {
		log.Printf("[PERF DEBUG] result is nil")
		return
	}

	list, ok := result.([]interface{})
	if !ok {
		log.Printf("[PERF DEBUG] unexpected result type: %T", result)
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	matched := 0
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := m["name"].(string)
		if name == "" {
			continue
		}

		e, found := t.byURL[name]
		if !found {
			log.Printf("[PERF DEBUG] unmatched URL: %s", name)
			continue
		}

		matched++
		e.Duration = numberAsFloat64(m["duration"])
		e.TransferSize = numberAsInt64(m["transferSize"])
		e.EncodedSize = numberAsInt64(m["encodedBodySize"])
		e.DecodedSize = numberAsInt64(m["decodedBodySize"])

		log.Printf("[PERF DEBUG] url=%s transfer=%d encoded=%d decoded=%d duration=%.2f",
			name, e.TransferSize, e.EncodedSize, e.DecodedSize, e.Duration)
	}

	log.Printf("[PERF DEBUG] entries=%d matched=%d", len(list), matched)
}

func numberAsFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case json.Number:
		f, _ := strconv.ParseFloat(string(n), 64)
		return f
	default:
		return 0
	}
}

func numberAsInt64(v interface{}) int64 {
	f := numberAsFloat64(v)
	return int64(f)
}

func (t *Tracker) Resources() []*ResourceEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*ResourceEntry, len(t.resources))
	for i, resource := range t.resources {
		resourceCopy := *resource
		out[i] = &resourceCopy
	}
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
	for i, download := range t.downloads {
		downloadCopy := make(map[string]interface{}, len(download))
		for key, value := range download {
			downloadCopy[key] = value
		}
		out[i] = downloadCopy
	}
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
	for _, typ := range types {
		set[typ] = true
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
