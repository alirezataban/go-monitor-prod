package collector

import (
	"fmt"
	"sync"
)

// CollectorDefinition associates a collector with its public API description.
type CollectorDefinition struct {
	Collector   Collector
	Description string
}

// Registry provides concurrency-safe collector lookup and API metadata.
type Registry struct {
	mu           sync.RWMutex
	collectors   map[string]Collector
	descriptions map[string]string
	names        []string
}

func NewRegistry(definitions ...CollectorDefinition) (*Registry, error) {
	registry := &Registry{
		collectors:   make(map[string]Collector, len(definitions)),
		descriptions: make(map[string]string, len(definitions)),
		names:        make([]string, 0, len(definitions)),
	}
	for _, definition := range definitions {
		if err := registry.Register(definition.Collector, definition.Description); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(c Collector, description string) error {
	if c == nil {
		return fmt.Errorf("collector is required")
	}
	name := c.Name()
	if name == "" {
		return fmt.Errorf("collector name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.collectors[name]; exists {
		return fmt.Errorf("collector %q is already registered", name)
	}
	r.collectors[name] = c
	r.descriptions[name] = description
	r.names = append(r.names, name)
	return nil
}

func (r *Registry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[name]
	return c, ok
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.names...)
}

func (r *Registry) Descriptions() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.descriptions))
	for name, description := range r.descriptions {
		out[name] = description
	}
	return out
}

// DefaultPackages is the default set of metrics returned when none are selected.
var DefaultPackages = []string{
	"availability", "ssl", "network", "breakdown", "failures",
	"console", "exceptions", "security", "page", "performance",
}

var defaultRegistry = mustNewDefaultRegistry()

// PackageList preserves the original static package list for callers that use it.
var PackageList = defaultRegistry.Names()

func mustNewDefaultRegistry() *Registry {
	registry, err := NewRegistry(
		CollectorDefinition{Collector: &Availability{}, Description: "HTTP status, redirects, timing"},
		CollectorDefinition{Collector: &SSL{}, Description: "Certificate, TLS, HSTS, OCSP"},
		CollectorDefinition{Collector: &NetworkSummary{}, Description: "Request counts, sizes, slow requests"},
		CollectorDefinition{Collector: &Breakdown{}, Description: "JS/CSS/Image/Font/XHR breakdown"},
		CollectorDefinition{Collector: &Assets{}, Description: "Largest asset, page size"},
		CollectorDefinition{Collector: &Domains{}, Description: "Unique & third-party domains"},
		CollectorDefinition{Collector: &Failures{}, Description: "Broken JS/CSS/images/APIs"},
		CollectorDefinition{Collector: &ConsolePkg{}, Description: "Console error/warning counts"},
		CollectorDefinition{Collector: &Exceptions{}, Description: "JS exceptions by type"},
		CollectorDefinition{Collector: &Security{}, Description: "Security headers + mixed content"},
		CollectorDefinition{Collector: &Redirects{}, Description: "Redirect chain & HTTP→HTTPS"},
		CollectorDefinition{Collector: &Downloads{}, Description: "Download attempts & executables"},
		CollectorDefinition{Collector: &Malware{}, Description: "Heuristic malware indicators"},
		CollectorDefinition{Collector: &PageInfo{}, Description: "Title, meta, OpenGraph"},
		CollectorDefinition{Collector: &SEO{}, Description: "robots.txt, sitemap, noindex"},
		CollectorDefinition{Collector: &Tech{}, Description: "CMS, framework, analytics, CDN"},
		CollectorDefinition{Collector: &PerfStats{}, Description: "Size percentiles, slowest request"},
		CollectorDefinition{Collector: &ResourceTypes{}, Description: "Counts by resource type"},
		CollectorDefinition{Collector: &ThirdParty{}, Description: "Third-party size & failures"},
	)
	if err != nil {
		panic(err)
	}
	return registry
}

func DefaultRegistry() *Registry {
	return defaultRegistry
}

// All returns a snapshot of every registered collector.
func All() map[string]Collector {
	out := make(map[string]Collector)
	for _, name := range defaultRegistry.Names() {
		c, _ := defaultRegistry.Get(name)
		out[name] = c
	}
	return out
}

func AllNames() []string {
	return defaultRegistry.Names()
}
