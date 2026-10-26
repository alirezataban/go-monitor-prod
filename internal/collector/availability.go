package collector

import (
	"fmt"
)

// Availability = HTTP status + redirects + timing (package 1).
type Availability struct{}

func (c *Availability) Name() string { return "availability" }

func (c *Availability) Collect(ctx *Context) (interface{}, error) {
	data := map[string]interface{}{}

	// --- HTTP ---
	httpInfo := map[string]interface{}{}
	if ctx.Response != nil {
		status := ctx.Response.Status()
		httpInfo["statusCode"] = status
		httpInfo["statusFamily"] = statusFamily(status)
		httpInfo["ok"] = ctx.Response.Ok()
		httpInfo["statusText"] = ctx.Response.StatusText()

		req := ctx.Response.Request()
		chain := []string{}
		redirectCount := 0
		cur := req
		for cur != nil && cur.RedirectedFrom() != nil {
			from := cur.RedirectedFrom()
			chain = append([]string{from.URL()}, chain...)
			redirectCount++
			cur = from
			if redirectCount > 20 {
				break
			}
		}
		if len(chain) > 0 {
			chain = append(chain, ctx.Page.URL())
		}
		httpInfo["finalUrl"] = ctx.Page.URL()
		httpInfo["redirectCount"] = redirectCount
		httpInfo["redirectChain"] = chain
		httpInfo["redirectLoop"] = hasRedirectLoop(chain)

		// HTTP version via JS if available
		ver, _ := ctx.Page.Evaluate(`() => {
			const n = performance.getEntriesByType('navigation')[0];
			return n ? (n.nextHopProtocol || null) : null;
		}`)
		if ver != nil {
			httpInfo["httpVersion"] = ver
		}
	} else {
		httpInfo["statusCode"] = 0
		httpInfo["ok"] = false
		httpInfo["finalUrl"] = ctx.Page.URL()
	}
	data["http"] = httpInfo

	// --- Timing ---
	timing, err := ctx.Page.Evaluate(`() => {
		const t = performance.getEntriesByType('navigation')[0];
		if (!t) return null;
		return {
			dnsLookup:        Math.round(t.domainLookupEnd - t.domainLookupStart),
			tcpConnect:       Math.round(t.connectEnd - t.connectStart),
			tlsHandshake:     Math.round(t.secureConnectionStart > 0 ? t.connectEnd - t.secureConnectionStart : 0),
			ttfb:             Math.round(t.responseStart - t.requestStart),
			domContentLoaded: Math.round(t.domContentLoadedEventEnd - t.startTime),
			loadEvent:        Math.round(t.loadEventEnd - t.startTime),
			fullyLoaded:      Math.round(t.loadEventEnd - t.startTime),
			transferSize:     t.transferSize || 0
		};
	}`)
	if err != nil {
		return data, fmt.Errorf("timing evaluate: %w", err)
	}
	if timing != nil {
		data["timing"] = timing
	} else {
		data["timing"] = map[string]interface{}{"available": false}
	}

	return data, nil
}

func statusFamily(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

func hasRedirectLoop(chain []string) bool {
	seen := map[string]bool{}
	for _, u := range chain {
		if seen[u] {
			return true
		}
		seen[u] = true
	}
	return false
}
