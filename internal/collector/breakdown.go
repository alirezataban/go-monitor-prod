package collector

type Breakdown struct{}

func (c *Breakdown) Name() string { return "breakdown" }

func (c *Breakdown) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	resources := ctx.Tracker.Resources()

	type agg struct {
		Count      int   `json:"count"`
		TotalSize  int64 `json:"totalSize"`
		FailedCount int  `json:"failedCount"`
	}

	js, css, img, font, xhr, media := agg{}, agg{}, agg{}, agg{}, agg{}, agg{}
	var largestImg *ResourceEntry

	for _, r := range resources {
		switch r.ResourceType {
		case "script":
			js.Count++
			js.TotalSize += r.TransferSize
			if isFailed(r) {
				js.FailedCount++
			}
		case "stylesheet":
			css.Count++
			css.TotalSize += r.TransferSize
			if isFailed(r) {
				css.FailedCount++
			}
		case "image":
			img.Count++
			img.TotalSize += r.TransferSize
			if isFailed(r) {
				img.FailedCount++
			}
			if largestImg == nil || r.TransferSize > largestImg.TransferSize {
				largestImg = r
			}
		case "font":
			font.Count++
			font.TotalSize += r.TransferSize
			if isFailed(r) {
				font.FailedCount++
			}
		case "xhr", "fetch":
			xhr.Count++
			xhr.TotalSize += r.TransferSize
			if isFailed(r) {
				xhr.FailedCount++
			}
		case "media":
			media.Count++
			media.TotalSize += r.TransferSize
		}
	}

	out := map[string]interface{}{
		"javascript": map[string]interface{}{
			"count": js.Count, "totalSize": js.TotalSize, "failedCount": js.FailedCount,
		},
		"css": map[string]interface{}{
			"count": css.Count, "totalSize": css.TotalSize, "failedCount": css.FailedCount,
		},
		"images": map[string]interface{}{
			"count": img.Count, "totalSize": img.TotalSize, "failedCount": img.FailedCount,
		},
		"fonts": map[string]interface{}{
			"count": font.Count, "totalSize": font.TotalSize, "failedCount": font.FailedCount,
		},
		"xhrFetch": map[string]interface{}{
			"apiRequestCount": xhr.Count, "failedApiCount": xhr.FailedCount, "totalSize": xhr.TotalSize,
		},
		"media": map[string]interface{}{
			"videoAudioCount": media.Count, "totalSize": media.TotalSize,
		},
	}
	if largestImg != nil {
		out["images"].(map[string]interface{})["largestImage"] = map[string]interface{}{
			"url": largestImg.URL, "size": largestImg.TransferSize,
		}
	}
	return out, nil
}
