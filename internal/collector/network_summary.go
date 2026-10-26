package collector

type NetworkSummary struct{}

func (c *NetworkSummary) Name() string { return "network" }

func (c *NetworkSummary) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	resources := ctx.Tracker.Resources()

	total := len(resources)
	failed, success, redirects := 0, 0, 0
	var totalDown, totalUp int64
	var totalDuration float64
	var timed int
	var slowCount int
	var largest *ResourceEntry

	for _, r := range resources {
		if isFailed(r) {
			failed++
		} else if isSuccess(r) {
			success++
		}
		if r.Status >= 300 && r.Status < 400 {
			redirects++
		}
		totalDown += r.TransferSize
		if r.Duration > 0 {
			totalDuration += r.Duration
			timed++
			if r.Duration > 3000 {
				slowCount++
			}
		}
		if largest == nil || r.TransferSize > largest.TransferSize {
			largest = r
		}
	}

	avg := 0.0
	if timed > 0 {
		avg = totalDuration / float64(timed)
	}

	out := map[string]interface{}{
		"totalRequests":      total,
		"failedRequests":     failed,
		"successfulRequests": success,
		"redirectRequests":   redirects,
		"averageResponseTime": round2(avg),
		"slowRequestsCount":  slowCount,
		"totalDownloadedBytes": totalDown,
		"totalUploadedBytes":   totalUp, // browser does not easily expose upload body size
	}
	if largest != nil {
		out["largestResponse"] = map[string]interface{}{
			"url":          largest.URL,
			"transferSize": largest.TransferSize,
			"type":         largest.ResourceType,
			"status":       largest.Status,
		}
	}
	return out, nil
}

func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}
