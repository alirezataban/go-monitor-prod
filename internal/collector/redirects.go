package collector

import (
	"net/url"
	"strings"
)

type Redirects struct{}

func (c *Redirects) Name() string { return "redirects" }

func (c *Redirects) Collect(ctx *Context) (interface{}, error) {
	chain := []string{}
	count := 0
	httpToHttps := false
	suspicious := false

	if ctx.Response != nil {
		req := ctx.Response.Request()
		cur := req
		for cur != nil && cur.RedirectedFrom() != nil {
			from := cur.RedirectedFrom()
			chain = append([]string{from.URL()}, chain...)
			count++
			cur = from
			if count > 20 {
				break
			}
		}
		if len(chain) > 0 {
			chain = append(chain, ctx.Page.URL())
		}

		// HTTP → HTTPS
		if len(chain) >= 2 {
			first, _ := url.Parse(chain[0])
			last, _ := url.Parse(chain[len(chain)-1])
			if first != nil && last != nil {
				if first.Scheme == "http" && last.Scheme == "https" {
					httpToHttps = true
				}
			}
		}
	}

	// suspicious: redirect to different registrable-ish domain
	mainHost := ""
	if u, err := url.Parse(ctx.URL); err == nil {
		mainHost = strings.ToLower(u.Hostname())
	}
	for _, u := range chain {
		h := extractHost(u)
		if h != "" && mainHost != "" && h != mainHost && !strings.HasSuffix(h, "."+mainHost) {
			// allow www variant
			if strings.TrimPrefix(h, "www.") != strings.TrimPrefix(mainHost, "www.") {
				suspicious = true
				break
			}
		}
	}

	return map[string]interface{}{
		"httpToHttpsRedirect":      httpToHttps,
		"redirectChain":            chain,
		"redirectCount":            count,
		"redirectToSuspiciousDomain": suspicious,
		"redirectLoop":             hasRedirectLoop(chain),
	}, nil
}
