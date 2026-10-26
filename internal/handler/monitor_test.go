package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alirezataban/monitor-engine-go/internal/collector"
	"github.com/alirezataban/monitor-engine-go/internal/config"
	"github.com/alirezataban/monitor-engine-go/internal/report"
)

func TestParseRequestJSONAndQueryPrecedence(t *testing.T) {
	cfg := &config.Config{DefaultTimeout: 30 * time.Second, UserAgent: "default-agent"}
	h := NewMonitorHandlerWithRegistry(nil, cfg, newTestRegistry(t))
	body := `{"url":"https://body.example","collectors":["page"],"timeout":40,"insecure":false,"userAgent":"json-agent","viewportWidth":1280,"viewportHeight":720,"waitUntil":"domcontentloaded"}`
	r := httptest.NewRequest(http.MethodPost, "/monitor?url=https%3A%2F%2Fquery.example&packages=availability%2Cssl&timeout=15&insecure=true&ua=query-agent&wait=networkidle", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	got, err := h.parseRequest(r)
	if err != nil {
		t.Fatalf("parseRequest() error = %v", err)
	}
	if got.URL != "https://query.example" {
		t.Errorf("URL = %q, want query value", got.URL)
	}
	if strings.Join(got.Collectors, ",") != "availability,ssl" {
		t.Errorf("Collectors = %v, want [availability ssl]", got.Collectors)
	}
	if got.TimeoutSec != 15 || !got.IgnoreHTTPS || got.UserAgent != "query-agent" || got.WaitUntil != "networkidle" {
		t.Errorf("query options not applied: %+v", got)
	}
	if got.ViewportW != 1280 || got.ViewportH != 720 {
		t.Errorf("JSON viewport was not preserved: %+v", got)
	}
}

func TestParseRequestDefaultsAndDomain(t *testing.T) {
	cfg := &config.Config{DefaultTimeout: 25 * time.Second}
	h := NewMonitorHandlerWithRegistry(nil, cfg, newTestRegistry(t))
	r := httptest.NewRequest(http.MethodGet, "/monitor?domain=example.com", nil)

	got, err := h.parseRequest(r)
	if err != nil {
		t.Fatalf("parseRequest() error = %v", err)
	}
	if got.URL != "https://example.com" {
		t.Errorf("URL = %q, want https://example.com", got.URL)
	}
	if got.TimeoutSec != 25 {
		t.Errorf("TimeoutSec = %d, want 25", got.TimeoutSec)
	}
	if strings.Join(got.Collectors, ",") != strings.Join(collector.DefaultPackages, ",") {
		t.Errorf("default collectors = %v, want %v", got.Collectors, collector.DefaultPackages)
	}
}

func TestParseRequestRejectsMissingTargetAndInvalidJSON(t *testing.T) {
	h := NewMonitorHandlerWithRegistry(nil, &config.Config{DefaultTimeout: time.Second}, newTestRegistry(t))

	_, err := h.parseRequest(httptest.NewRequest(http.MethodGet, "/monitor", nil))
	if err == nil || err.Error() != "url or domain is required" {
		t.Errorf("missing target error = %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/monitor", strings.NewReader("{"))
	r.Header.Set("Content-Type", "application/json")
	if _, err := h.parseRequest(r); err == nil || !strings.HasPrefix(err.Error(), "invalid json body:") {
		t.Errorf("invalid JSON error = %v", err)
	}
}

type recordingCollector struct {
	name  string
	order *[]string
}

func newTestRegistry(t *testing.T, definitions ...collector.CollectorDefinition) *collector.Registry {
	t.Helper()
	registry, err := collector.NewRegistry(definitions...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
}

func (c recordingCollector) Name() string { return c.name }

func (c recordingCollector) Collect(*collector.Context) (interface{}, error) {
	*c.order = append(*c.order, c.name)
	return map[string]string{"collector": c.name}, nil
}

func TestCollectorsRunSequentiallyAndUnknownNamesAreReported(t *testing.T) {
	order := []string{}
	registry := newTestRegistry(t,
		collector.CollectorDefinition{Collector: recordingCollector{name: "first", order: &order}},
		collector.CollectorDefinition{Collector: recordingCollector{name: "second", order: &order}},
	)
	h := NewMonitorHandlerWithRegistry(nil, &config.Config{}, registry)
	rep := report.New("https://example.com")

	h.collect(rep, &collector.Context{}, []string{"second", "missing", "first"}, nil)

	if strings.Join(order, ",") != "second,first" {
		t.Errorf("collector order = %v, want [second first]", order)
	}
	if len(rep.Collectors) != 3 {
		t.Fatalf("got %d collector results, want 3", len(rep.Collectors))
	}
	if result := rep.Collectors["missing"]; result.Success || result.Error != "unknown package: missing" {
		t.Errorf("unknown collector result = %+v", result)
	}
}

func TestReportResponseContract(t *testing.T) {
	rep := report.New("https://example.com")
	rep.Add("availability", true, time.Millisecond, map[string]int{"statusCode": 200}, nil)
	rep.Finish(nil)

	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, rep)
	var got map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is invalid JSON: %v", err)
	}
	for _, key := range []string{"summary", "url", "collectors"} {
		if _, ok := got[key]; !ok {
			t.Errorf("response missing top-level key %q", key)
		}
	}
	if len(got) != 3 {
		t.Errorf("response has %d top-level keys, want 3", len(got))
	}
}
