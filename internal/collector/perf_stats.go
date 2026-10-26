package collector

import "sort"

type PerfStats struct{}

func (c *PerfStats) Name() string { return "performance" }

func (c *PerfStats) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	resources := ctx.Tracker.Resources()
	if len(resources) == 0 {
		return map[string]interface{}{
			"minResourceSize": 0, "maxResourceSize": 0, "averageResourceSize": 0,
		}, nil
	}

	sizes := make([]int64, 0, len(resources))
	var total int64
	var slowest, fastest *ResourceEntry

	for _, r := range resources {
		sizes = append(sizes, r.TransferSize)
		total += r.TransferSize
		if r.Duration > 0 {
			if slowest == nil || r.Duration > slowest.Duration {
				slowest = r
			}
			if fastest == nil || r.Duration < fastest.Duration {
				fastest = r
			}
		}
	}

	sort.Slice(sizes, func(i, j int) bool { return sizes[i] < sizes[j] })
	min, max := sizes[0], sizes[len(sizes)-1]
	avg := total / int64(len(sizes))
	median := sizes[len(sizes)/2]
	p95idx := int(float64(len(sizes)) * 0.95)
	if p95idx >= len(sizes) {
		p95idx = len(sizes) - 1
	}
	p95 := sizes[p95idx]

	out := map[string]interface{}{
		"minResourceSize":           min,
		"maxResourceSize":           max,
		"averageResourceSize":       avg,
		"medianResourceSize":        median,
		"p95ResourceSize":           p95,
	}
	if slowest != nil {
		out["slowestRequest"] = map[string]interface{}{
			"url": slowest.URL, "durationMs": slowest.Duration, "type": slowest.ResourceType,
		}
	}
	if fastest != nil {
		out["fastestRequest"] = map[string]interface{}{
			"url": fastest.URL, "durationMs": fastest.Duration, "type": fastest.ResourceType,
		}
	}
	return out, nil
}
