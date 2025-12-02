package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

type PrettyHandler struct {
	handler slog.Handler
	module  string
	w       io.Writer
	mu      *sync.Mutex
}

func NewPrettyHandler(out io.Writer, module string) *PrettyHandler {
	return &PrettyHandler{
		handler: slog.NewTextHandler(out, nil),
		module:  module,
		w:       out,
		mu:      &sync.Mutex{},
	}
}

func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	timeStr := r.Time.Format("2006/01/02 15:04:05")
	level := r.Level.String()
	h.mu.Lock()

	defer h.mu.Unlock()
	_, err := fmt.Fprintf(h.w, "%s %-5s [%s] %s\n", timeStr, level, h.module, r.Message)
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Simplificación para este ejemplo
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h // Simplificación para este ejemplo
}

type RateLimiterConfig struct {
	NormalRate  float64 `json:"normal_rate"`
	NormalBurst float64 `json:"normal_burst"`
	AttackRate  float64 `json:"attack_rate"`
	AttackBurst float64 `json:"attack_burst"`
}

type AnalyzerConfig struct {
	HighTrafficThreshold float64 `json:"high_traffic_threshold"`
	AttackThreshold      float64 `json:"attack_threshold"`
}

type Config struct {
	AppName     string            `json:"app_name"`
	Logging     map[string]bool   `json:"logging"`
	RateLimiter RateLimiterConfig `json:"rate_limiter"`
	Analyzer    AnalyzerConfig    `json:"analyzer"`
}

var globalConfig Config

func LoadConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(&globalConfig)
}

func GetModuleLogger(moduleName string) *slog.Logger {
	isEnabled, exists := globalConfig.Logging[moduleName]

	if !exists || !isEnabled {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	handler := NewPrettyHandler(os.Stdout, moduleName)
	return slog.New(handler)
}

func GetRateLimiterConfig() RateLimiterConfig {
	return globalConfig.RateLimiter
}

func GetAnalyzerConfig() AnalyzerConfig {
	return globalConfig.Analyzer
}
