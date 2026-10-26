package collector

type ThirdParty struct{}

func (c *ThirdParty) Name() string { return "thirdparty" }

func (c *ThirdParty) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	resources := ctx.Tracker.Resources()

	count := 0
	var size int64
	failures := 0
	slowDomains := map[string]float64{} // domain -> max duration

	for _, r := range resources {
		if !r.IsThirdParty {
			continue
		}
		count++
		size += r.TransferSize
		if isFailed(r) {
			failures++
		}
		if r.Duration > 2000 {
			if d, ok := slowDomains[r.Domain]; !ok || r.Duration > d {
				slowDomains[r.Domain] = r.Duration
			}
		}
	}

	slowList := []map[string]interface{}{}
	for dom, dur := range slowDomains {
		slowList = append(slowList, map[string]interface{}{
			"domain": dom, "maxDurationMs": dur,
		})
	}

	return map[string]interface{}{
		"thirdPartyRequestCount": count,
		"thirdPartySize":         size,
		"thirdPartyFailures":     failures,
		"slowThirdPartyDomains":  slowList,
	}, nil
}
