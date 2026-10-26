package collector

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type SEO struct{}

func (c *SEO) Name() string { return "seo" }

func (c *SEO) Collect(ctx *Context) (interface{}, error) {
	base, err := url.Parse(ctx.URL)
	if err != nil {
		return nil, err
	}

	robotsExists := headOK(base, "/robots.txt")
	sitemapExists := headOK(base, "/sitemap.xml")

	// noindex / nofollow from meta
	flags, _ := ctx.Page.Evaluate(`() => {
		const el = document.querySelector('meta[name="robots"]');
		const content = el ? (el.getAttribute('content') || '').toLowerCase() : '';
		return {
			noindex: content.includes('noindex'),
			nofollow: content.includes('nofollow')
		};
	}`)

	out := map[string]interface{}{
		"robotsTxtExists": robotsExists,
		"sitemapXmlExists": sitemapExists,
	}
	if m, ok := flags.(map[string]interface{}); ok {
		out["noindex"] = m["noindex"]
		out["nofollow"] = m["nofollow"]
	} else {
		out["noindex"] = false
		out["nofollow"] = false
	}
	return out, nil
}

func headOK(base *url.URL, path string) bool {
	u := *base
	u.Path = path
	u.RawQuery = ""
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "MonitorEngine-Go/2.0")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
