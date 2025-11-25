package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal/ratelimiter"
)

// BackendConfig representa la configuración de un backend
type BackendConfig struct {
	URL     string  `json:"url"`
	Weight  float64 `json:"weight"`
	Enabled bool    `json:"enabled"`
}

// ProxyConfig representa la configuración del proxy
type ProxyConfig struct {
	Algorithm     string        `json:"algorithm"` // "round_robin", "weighted_round_robin", "least_connections"
	RetryAttempts int           `json:"retry_attempts"`
	RetryDelay    time.Duration `json:"retry_delay_ms"`
	Timeout       time.Duration `json:"timeout_ms"`
	Port          int           `json:"port"`
}

// MonitorConfig representa la configuración del monitor
type MonitorConfig struct {
	Alpha    float64 `json:"alpha"`     // Factor EMA
	PeriodS  int     `json:"period_s"`  // Periodo de polling en segundos
	TimeoutS int     `json:"timeout_s"` // Timeout en segundos
}

// GetPeriod retorna el periodo como time.Duration
func (m MonitorConfig) GetPeriod() time.Duration {
	return time.Duration(m.PeriodS) * time.Second
}

// GetTimeout retorna el timeout como time.Duration
func (m MonitorConfig) GetTimeout() time.Duration {
	return time.Duration(m.TimeoutS) * time.Second
}

// WebConfig representa la configuración del servidor web
type WebConfig struct {
	MetricsPort int `json:"metrics_port"` // Puerto para WebSocket y Admin API
}

// CircuitBreakerConfig representa la configuración del circuit breaker
type CircuitBreakerConfig struct {
	Enabled            bool          `json:"enabled"`
	FailureThreshold   int           `json:"failure_threshold"`    // Número de fallos para abrir circuito
	ErrorRateThreshold float64       `json:"error_rate_threshold"` // Porcentaje de error para abrir (0-1)
	OpenTimeout        time.Duration `json:"open_timeout_s"`       // Tiempo antes de intentar half-open
	HalfOpenMaxCalls   int           `json:"half_open_max_calls"`  // Máx llamadas en estado half-open
	MinRequestCount    int           `json:"min_request_count"`    // Mín requests antes de evaluar
}

// WeightOptimizationConfig representa la configuración de pesos adaptativos
type WeightOptimizationConfig struct {
	Enabled               bool    `json:"enabled"`
	MinWeight             float64 `json:"min_weight"`
	MaxWeight             float64 `json:"max_weight"`
	LatencyWeight         float64 `json:"latency_weight"`
	ErrorRateWeight       float64 `json:"error_rate_weight"`
	AdaptationSpeed       float64 `json:"adaptation_speed"`
	LatencyTargetMs       float64 `json:"latency_target_ms"`
	MaxErrorRate          float64 `json:"max_error_rate"`
	UpdateIntervalSeconds int     `json:"update_interval_s"`
}

// Config representa la configuración completa del balanceador
type Config struct {
	Backends           []BackendConfig               `json:"backends"`
	Proxy              ProxyConfig                   `json:"proxy"`
	Monitor            MonitorConfig                 `json:"monitor"`
	Web                WebConfig                     `json:"web"`
	CircuitBreaker     CircuitBreakerConfig          `json:"circuit_breaker"`
	WeightOptimization WeightOptimizationConfig      `json:"weight_optimization"`
	RateLimit          ratelimiter.RateLimiterConfig `json:"rate_limit"`
}

// DefaultConfig devuelve la configuración por defecto
func DefaultConfig() *Config {
	return &Config{
		Backends: []BackendConfig{
			{URL: "http://localhost:8080", Weight: 1.0, Enabled: true},
			{URL: "http://localhost:8081", Weight: 1.0, Enabled: true},
			{URL: "http://localhost:8082", Weight: 1.0, Enabled: true},
		},
		Proxy: ProxyConfig{
			Algorithm:     "round_robin",
			RetryAttempts: 2,
			RetryDelay:    100 * time.Millisecond,
			Timeout:       10 * time.Second,
			Port:          8089,
		},
		Monitor: MonitorConfig{
			Alpha:    0.3,
			PeriodS:  5,
			TimeoutS: 10,
		},
		Web: WebConfig{
			MetricsPort: 9000,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:            true,
			FailureThreshold:   5,
			ErrorRateThreshold: 0.5,
			OpenTimeout:        30 * time.Second,
			HalfOpenMaxCalls:   3,
			MinRequestCount:    5,
		},
		WeightOptimization: WeightOptimizationConfig{
			Enabled:               true,
			MinWeight:             0.1,
			MaxWeight:             5.0,
			LatencyWeight:         0.6,
			ErrorRateWeight:       0.4,
			AdaptationSpeed:       0.1,
			LatencyTargetMs:       100.0,
			MaxErrorRate:          0.1,
			UpdateIntervalSeconds: 10,
		},
		RateLimit: ratelimiter.DefaultRateLimiterConfig(),
	}
}

// LoadConfig carga la configuración desde un archivo JSON
func LoadConfig(filePath string) (*Config, error) {
	// Si el archivo no existe, crear uno con la configuración por defecto
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		defaultConfig := DefaultConfig()
		if err := SaveConfig(filePath, defaultConfig); err != nil {
			return nil, fmt.Errorf("error creating default config: %w", err)
		}
		fmt.Printf("Created default configuration file: %s\n", filePath)
		return defaultConfig, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validar configuración
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveConfig guarda la configuración en un archivo JSON
func SaveConfig(filePath string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// Validate valida la configuración
func (c *Config) Validate() error {
	if len(c.Backends) == 0 {
		return fmt.Errorf("at least one backend must be configured")
	}

	for i, backend := range c.Backends {
		if backend.URL == "" {
			return fmt.Errorf("backend %d: URL cannot be empty", i)
		}
		if backend.Weight < 0 {
			return fmt.Errorf("backend %d: weight cannot be negative", i)
		}
	}

	if c.Proxy.RetryAttempts < 0 {
		return fmt.Errorf("proxy retry_attempts cannot be negative")
	}

	if c.Proxy.Port <= 0 || c.Proxy.Port > 65535 {
		return fmt.Errorf("proxy port must be between 1 and 65535")
	}

	if c.Web.MetricsPort <= 0 || c.Web.MetricsPort > 65535 {
		return fmt.Errorf("web metrics_port must be between 1 and 65535")
	}

	if c.Monitor.Alpha < 0 || c.Monitor.Alpha > 1 {
		return fmt.Errorf("monitor alpha must be between 0 and 1")
	}

	return nil
}

// GetEnabledBackends devuelve solo los backends habilitados
func (c *Config) GetEnabledBackends() []string {
	var enabled []string
	for _, backend := range c.Backends {
		if backend.Enabled {
			enabled = append(enabled, backend.URL)
		}
	}
	return enabled
}

// GetBackendWeight devuelve el peso de un backend por URL
func (c *Config) GetBackendWeight(url string) float64 {
	for _, backend := range c.Backends {
		if backend.URL == url {
			return backend.Weight
		}
	}
	return 1.0 // peso por defecto
}
