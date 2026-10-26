package collector

type ConsolePkg struct{}

func (c *ConsolePkg) Name() string { return "console" }

func (c *ConsolePkg) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	msgs := ctx.Tracker.Console()

	counts := map[string]int{
		"error": 0, "warning": 0, "log": 0, "info": 0, "debug": 0, "total": len(msgs),
	}
	for _, m := range msgs {
		if _, ok := counts[m.Type]; ok {
			counts[m.Type]++
		}
	}

	// limit messages in output
	outMsgs := msgs
	if len(outMsgs) > 40 {
		outMsgs = outMsgs[:40]
	}

	return map[string]interface{}{
		"consoleErrorCount":   counts["error"],
		"consoleWarningCount": counts["warning"],
		"consoleLogCount":     counts["log"],
		"consoleInfoCount":    counts["info"],
		"total":               counts["total"],
		"messages":            outMsgs,
	}, nil
}
