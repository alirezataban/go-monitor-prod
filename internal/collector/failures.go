package collector

type Failures struct{}

func (c *Failures) Name() string { return "failures" }

func (c *Failures) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	resources := ctx.Tracker.Resources()

	var brokenJS, brokenCSS, brokenImg, brokenFont, brokenAPI []map[string]interface{}

	add := func(list *[]map[string]interface{}, r *ResourceEntry) {
		*list = append(*list, map[string]interface{}{
			"url":    r.URL,
			"status": r.Status,
			"error":  r.Error,
		})
	}

	for _, r := range resources {
		if !isFailed(r) {
			continue
		}
		switch r.ResourceType {
		case "script":
			add(&brokenJS, r)
		case "stylesheet":
			add(&brokenCSS, r)
		case "image":
			add(&brokenImg, r)
		case "font":
			add(&brokenFont, r)
		case "xhr", "fetch":
			add(&brokenAPI, r)
		}
	}

	total := len(brokenJS) + len(brokenCSS) + len(brokenImg) + len(brokenFont) + len(brokenAPI)

	// limit lists
	limit := func(list []map[string]interface{}, n int) []map[string]interface{} {
		if len(list) > n {
			return list[:n]
		}
		return list
	}

	return map[string]interface{}{
		"brokenJsFiles":       limit(brokenJS, 20),
		"brokenCssFiles":      limit(brokenCSS, 20),
		"brokenImages":        limit(brokenImg, 20),
		"brokenFonts":         limit(brokenFont, 10),
		"brokenApis":          limit(brokenAPI, 20),
		"totalFailedResources": total,
		"counts": map[string]int{
			"js": len(brokenJS), "css": len(brokenCSS), "images": len(brokenImg),
			"fonts": len(brokenFont), "apis": len(brokenAPI),
		},
	}, nil
}
