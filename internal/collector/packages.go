package collector

// Package = sellable unit. Each maps to one collector name.
// Client can request packages or individual names (same thing).

var PackageList = []string{
	"availability", // 1. Availability & Connectivity + Timing
	"ssl",          // 2. SSL
	"network",      // 3. Network Requests summary
	"breakdown",    // 4. Request Breakdown (JS/CSS/Img/Font/XHR)
	"assets",       // 5. Assets
	"domains",      // 6. Domain Analysis
	"failures",     // 7. Failed Resources
	"console",      // 8. Console Monitoring
	"exceptions",   // 9. JavaScript Exceptions
	"security",     // 10+11. Security Headers + Content Security
	"redirects",    // 12. Redirect Monitoring
	"downloads",    // 13. Download Monitoring
	"malware",      // 14. Malware Indicators (heuristic)
	"page",         // 15. Page Information
	"seo",          // 16. SEO Basics
	"tech",         // 17. Technology Detection
	"performance",  // 18. Performance Statistics
	"resources",    // 19. Resource Types counts
	"thirdparty",   // 20. Third-party Monitoring
}

// DefaultPackages – sensible starter set for most monitoring.
var DefaultPackages = []string{
	"availability", "ssl", "network", "breakdown", "failures",
	"console", "exceptions", "security", "page", "performance",
}

// All returns every registered collector.
func All() map[string]Collector {
	return map[string]Collector{
		"availability": &Availability{},
		"ssl":          &SSL{},
		"network":      &NetworkSummary{},
		"breakdown":    &Breakdown{},
		"assets":       &Assets{},
		"domains":      &Domains{},
		"failures":     &Failures{},
		"console":      &ConsolePkg{},
		"exceptions":   &Exceptions{},
		"security":     &Security{},
		"redirects":    &Redirects{},
		"downloads":    &Downloads{},
		"malware":      &Malware{},
		"page":         &PageInfo{},
		"seo":          &SEO{},
		"tech":         &Tech{},
		"performance":  &PerfStats{},
		"resources":    &ResourceTypes{},
		"thirdparty":   &ThirdParty{},
	}
}

func AllNames() []string {
	return append([]string{}, PackageList...)
}
