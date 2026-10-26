package collector


type Assets struct{}

func (c *Assets) Name() string { return "assets" }

func (c *Assets) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	resources := ctx.Tracker.Resources()
	if len(resources) == 0 {
		return map[string]interface{}{
			"totalPageSize": 0, "numberOfAssets": 0,
		}, nil
	}

	var total int64
	var largest *ResourceEntry
	over1MB, over500KB := 0, 0
	sizes := make([]int64, 0, len(resources))

	for _, r := range resources {
		total += r.TransferSize
		sizes = append(sizes, r.TransferSize)
		if largest == nil || r.TransferSize > largest.TransferSize {
			largest = r
		}
		if r.TransferSize > 1024*1024 {
			over1MB++
		}
		if r.TransferSize > 500*1024 {
			over500KB++
		}
	}

	avg := int64(0)
	if len(sizes) > 0 {
		avg = total / int64(len(sizes))
	}

	out := map[string]interface{}{
		"totalPageSize":       total,
		"averageAssetSize":    avg,
		"numberOfAssets":      len(resources),
		"numberOfAssetsOver1MB":  over1MB,
		"numberOfAssetsOver500KB": over500KB,
	}
	if largest != nil {
		out["largestAssetName"] = baseName(largest.URL)
		out["largestAssetUrl"] = largest.URL
		out["largestAssetType"] = largest.ResourceType
		out["largestAssetSize"] = largest.TransferSize
	}

	return out, nil
}

func baseName(raw string) string {
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == '/' {
			return raw[i+1:]
		}
	}
	return raw
}
