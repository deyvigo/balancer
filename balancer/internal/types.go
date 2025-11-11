package internal

import (
	"net/url"
	"time"
)

type Metrics struct {
	Id          int     `json:"id"`
	URL         string  `json:"url"`
	Alive       bool    `json:"alive"`
	EMAMs       float64 `json:"ema_ms"`
	ErrorRate   float64 `json:"error_rate"`
	LastChecked string  `json:"last_checked"`
}

type Backend struct {
	URL       *url.URL  `json:"url"`
	Alive     bool      `json:"alive"`
	EMAms     float64   `json:"ema_ms"`
	ErrorRate float64   `json:"error_rate"`
	CheckedAt time.Time `json:"checket_at"`
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
