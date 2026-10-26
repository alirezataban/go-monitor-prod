package report

import (
	"time"
)

type Summary struct {
	Success    bool   `json:"success"`
	StartedAt  int64  `json:"startedAt"`
	FinishedAt int64  `json:"finishedAt"`
	Duration   int64  `json:"duration"` // ms
	Error      string `json:"error,omitempty"`
}

type CollectorResult struct {
	Success  bool        `json:"success"`
	Duration int64       `json:"duration"` // ms
	Data     interface{} `json:"data,omitempty"`
	Error    string      `json:"error,omitempty"`
}

type Report struct {
	Summary    Summary                    `json:"summary"`
	URL        string                     `json:"url"`
	Collectors map[string]CollectorResult `json:"collectors"`
}

func New(url string) *Report {
	return &Report{
		Summary: Summary{
			StartedAt: time.Now().UnixMilli(),
		},
		URL:        url,
		Collectors: make(map[string]CollectorResult),
	}
}

func (r *Report) Finish(err error) {
	r.Summary.FinishedAt = time.Now().UnixMilli()
	r.Summary.Duration = r.Summary.FinishedAt - r.Summary.StartedAt
	if err != nil {
		r.Summary.Success = false
		r.Summary.Error = err.Error()
	} else {
		// success only if no fatal error; individual collectors may still fail
		r.Summary.Success = true
	}
}

func (r *Report) Add(name string, success bool, duration time.Duration, data interface{}, err error) {
	cr := CollectorResult{
		Success:  success,
		Duration: duration.Milliseconds(),
		Data:     data,
	}
	if err != nil {
		cr.Success = false
		cr.Error = err.Error()
	}
	r.Collectors[name] = cr
}
