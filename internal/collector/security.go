package collector

import (
	"strings"
)

type Security struct{}

func (c *Security) Name() string { return "security" }

func (c *Security) Collect(ctx *Context) (interface{}, error) {
	headers := map[string]interface{}{}
	get := func(name string) string {
		if ctx.Response == nil {
			return ""
		}
		v, _ := ctx.Response.HeaderValue(name)
		return v
	}

	hsts := get("strict-transport-security")
	csp := get("content-security-policy")
	xfo := get("x-frame-options")
	xcto := get("x-content-type-options")
	rp := get("referrer-policy")
	pp := get("permissions-policy")
	if pp == "" {
		pp = get("feature-policy")
	}
	xxss := get("x-xss-protection")

	headers["hsts"] = hsts != ""
	headers["hstsValue"] = hsts
	headers["csp"] = csp != ""
	headers["cspValue"] = truncate(csp, 200)
	headers["xFrameOptions"] = xfo != ""
	headers["xFrameOptionsValue"] = xfo
	headers["xContentTypeOptions"] = strings.EqualFold(xcto, "nosniff")
	headers["referrerPolicy"] = rp != ""
	headers["referrerPolicyValue"] = rp
	headers["permissionsPolicy"] = pp != ""
	headers["xXssProtection"] = xxss != ""

	// Content security: mixed content from resources
	mixed := []string{}
	insecure := []string{}
	if ctx.Tracker != nil {
		pageIsHTTPS := strings.HasPrefix(strings.ToLower(ctx.URL), "https://")
		for _, r := range ctx.Tracker.Resources() {
			if pageIsHTTPS && strings.HasPrefix(strings.ToLower(r.URL), "http://") {
				mixed = append(mixed, r.URL)
				insecure = append(insecure, r.URL)
			}
		}
	}
	if len(mixed) > 15 {
		mixed = mixed[:15]
	}
	if len(insecure) > 15 {
		insecure = insecure[:15]
	}

	return map[string]interface{}{
		"headers": headers,
		"contentSecurity": map[string]interface{}{
			"mixedContentDetected": len(mixed) > 0,
			"mixedContentCount":    len(mixed),
			"mixedContentUrls":     mixed,
			"insecureHttpResources": insecure,
		},
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
