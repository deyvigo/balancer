package internal

import (
	"net/url"
	"time"
)

type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateOpen     CircuitState = "OPEN"
	StateHalfOpen CircuitState = "HALF_OPEN"
)

type Metrics struct {
	Id           int          `json:"id"`
	URL          string       `json:"url"`
	Alive        bool         `json:"alive"`
	EMAMs        float64      `json:"ema_ms"`
	ErrorRate    float64      `json:"error_rate"`
	LastChecked  string       `json:"last_checked"`
	CircuitState CircuitState `json:"circuit_state"`
}

type BalancerStats struct {
	TotalReqs   uint64  `json:"total_reqs"`
	BlockedReqs uint64  `json:"blocked_reqs"`
	RPS         float64 `json:"rps"`
}

type SystemStatus struct {
	Backends []Metrics     `json:"backends"`
	LB       BalancerStats `json:"balancer"`
}

type Backend struct {
	URL             *url.URL     `json:"url"`
	Alive           bool         `json:"alive"`
	EMAms           float64      `json:"ema_ms"`
	ErrorRate       float64      `json:"error_rate"`
	CheckedAt       time.Time    `json:"checket_at"`
	Failures        int          `json:"failures"`
	CircuitState    CircuitState `json:"circuit_state"`
	LastStateChange time.Time    `json:"last_state_change"`
}

type Decision struct {
	URL      string    `json:"url"`
	Severity string    `json:"severity"` // "info","warning","critical"
	Reason   string    `json:"reason"`
	Time     time.Time `json:"time"`
}

type Action struct {
	URL  string            `json:"url"`
	Type string            `json:"type"` // "drain","restart","notify","noop"
	Meta map[string]string `json:"meta"`
}

type AnalysisResult struct {
	BackendId    int
	Status       string // "HEALTHY","DEGRADED","DOWN"
	Reason       string
	CircuitState CircuitState
}

type PlanResult struct {
	BackendId int
	Action    string
}
