package collector

type Exceptions struct{}

func (c *Exceptions) Name() string { return "exceptions" }

func (c *Exceptions) Collect(ctx *Context) (interface{}, error) {
	if ctx.Tracker == nil {
		return map[string]interface{}{"available": false}, nil
	}
	list := ctx.Tracker.Exceptions()

	counts := map[string]int{
		"TypeError": 0, "ReferenceError": 0, "SyntaxError": 0,
		"UnhandledPromiseRejection": 0, "Error": 0,
	}
	messages := make([]string, 0, len(list))
	for _, e := range list {
		if _, ok := counts[e.Type]; ok {
			counts[e.Type]++
		} else {
			counts["Error"]++
		}
		messages = append(messages, e.Message)
	}
	if len(messages) > 30 {
		messages = messages[:30]
	}

	return map[string]interface{}{
		"typeErrorCount":              counts["TypeError"],
		"referenceErrorCount":         counts["ReferenceError"],
		"syntaxErrorCount":            counts["SyntaxError"],
		"unhandledPromiseRejections":  counts["UnhandledPromiseRejection"],
		"totalJsExceptions":           len(list),
		"exceptionMessages":           messages,
	}, nil
}
