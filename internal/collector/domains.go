package collector

type Domains struct{}

func (c *Domains) Name() string { return "domains" }

func (c *Domains) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	resources := ctx.Tracker.Resources()
	mainHost := ctx.Tracker.MainHost()

	unique := map[string]bool{}
	thirdParty := map[string]bool{}
	cdn := map[string]bool{}
	extScript := map[string]bool{}
	extImage := map[string]bool{}
	extFont := map[string]bool{}

	cdnHints := []string{"cloudflare", "cloudfront", "akamai", "fastly", "cdn.", "jsdelivr", "unpkg", "googleapis", "gstatic", "bootstrapcdn", "fontawesome"}

	for _, r := range resources {
		if r.Domain == "" {
			continue
		}
		unique[r.Domain] = true
		if r.IsThirdParty {
			thirdParty[r.Domain] = true
			for _, h := range cdnHints {
				if containsFold(r.Domain, h) || containsFold(r.URL, h) {
					cdn[r.Domain] = true
					break
				}
			}
			switch r.ResourceType {
			case "script":
				extScript[r.Domain] = true
			case "image":
				extImage[r.Domain] = true
			case "font":
				extFont[r.Domain] = true
			}
		}
	}

	return map[string]interface{}{
		"uniqueDomainsCount":    len(unique),
		"thirdPartyDomainsCount": len(thirdParty),
		"cdnDomainsCount":       len(cdn),
		"externalScriptDomains": keys(extScript),
		"externalImageDomains":  keys(extImage),
		"externalFontDomains":   keys(extFont),
		"mainHost":              mainHost,
	}, nil
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	// simple case-insensitive search
	ls, lsub := len(s), len(sub)
	for i := 0; i+lsub <= ls; i++ {
		match := true
		for j := 0; j < lsub; j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
