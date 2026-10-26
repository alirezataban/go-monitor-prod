package collector

type ResourceTypes struct{}

func (c *ResourceTypes) Name() string { return "resources" }

func (c *ResourceTypes) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	counts := map[string]int{
		"script": 0, "stylesheet": 0, "image": 0, "font": 0,
		"fetch": 0, "xhr": 0, "media": 0, "document": 0, "other": 0,
	}
	for _, r := range ctx.Tracker.Resources() {
		t := r.ResourceType
		if _, ok := counts[t]; ok {
			counts[t]++
		} else {
			counts["other"]++
		}
	}
	return map[string]interface{}{
		"scriptCount":      counts["script"],
		"stylesheetCount":  counts["stylesheet"],
		"imageCount":       counts["image"],
		"fontCount":        counts["font"],
		"fetchCount":       counts["fetch"],
		"xhrCount":         counts["xhr"],
		"mediaCount":       counts["media"],
		"documentCount":    counts["document"],
		"otherCount":       counts["other"],
	}, nil
}
