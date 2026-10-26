package collector

import (
	"github.com/mxschmitt/playwright-go"
)

// Context is passed to every collector for one job.
type Context struct {
	Page     playwright.Page
	Response playwright.Response
	URL      string
	Tracker  *Tracker
}

// Collector collects one package of metrics.
type Collector interface {
	Name() string
	Collect(ctx *Context) (interface{}, error)
}
