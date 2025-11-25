package ratelimiter

import (
	"net"
	"sync"
	"time"
)

// RateLimiterType define el tipo de rate limiter
type RateLimiterType int

const (
	TokenBucketType RateLimiterType = iota
	SlidingWindowType
)

// RateLimiterConfig configuración para rate limiting
type RateLimiterConfig struct {
	Enabled     bool     `json:"enabled"`
	Type        string   `json:"type"`          // "token_bucket" o "sliding_window"
	GlobalLimit int64    `json:"global_limit"`  // Límite global de peticiones
	PerIPLimit  int64    `json:"per_ip_limit"`  // Límite por IP
	WindowSizeS int      `json:"window_size_s"` // Tamaño de ventana en segundos
	RefillRate  int64    `json:"refill_rate"`   // Tasa de refill para token bucket
	Whitelist   []string `json:"whitelist"`     // IPs en whitelist
}

// GetWindowSize retorna el tamaño de ventana como time.Duration
func (r RateLimiterConfig) GetWindowSize() time.Duration {
	return time.Duration(r.WindowSizeS) * time.Second
}

// DefaultRateLimiterConfig configuración por defecto
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Enabled:     true,
		Type:        "token_bucket",
		GlobalLimit: 1000,                         // 1000 peticiones por ventana
		PerIPLimit:  100,                          // 100 peticiones por IP por ventana
		WindowSizeS: 60,                           // Ventana de 60 segundos
		RefillRate:  10,                           // 10 tokens por segundo
		Whitelist:   []string{"127.0.0.1", "::1"}, // Localhost en whitelist
	}
}

// RateLimiter interface para diferentes implementaciones
type RateLimiter interface {
	Allow() bool
}

// RateLimiterManager maneja rate limiting global y por IP
type RateLimiterManager struct {
	config        RateLimiterConfig
	globalLimiter RateLimiter
	ipLimiters    map[string]RateLimiter
	whitelist     map[string]bool
	mu            sync.RWMutex
}

// NewRateLimiterManager crea un nuevo manager de rate limiting
func NewRateLimiterManager(config RateLimiterConfig) *RateLimiterManager {
	manager := &RateLimiterManager{
		config:     config,
		ipLimiters: make(map[string]RateLimiter),
		whitelist:  make(map[string]bool),
	}

	// Configurar whitelist
	for _, ip := range config.Whitelist {
		manager.whitelist[ip] = true
	}

	// Crear limitador global
	switch config.Type {
	case "token_bucket":
		manager.globalLimiter = NewTokenBucket(config.GlobalLimit, config.RefillRate)
	case "sliding_window":
		manager.globalLimiter = NewSlidingWindow(int(config.GlobalLimit), config.GetWindowSize())
	default:
		manager.globalLimiter = NewTokenBucket(config.GlobalLimit, config.RefillRate)
	}

	return manager
}

// Allow verifica si se permite una petición desde una IP específica
func (rlm *RateLimiterManager) Allow(clientIP string) bool {
	if !rlm.config.Enabled {
		return true
	}

	// Extraer IP real de la dirección
	ip := rlm.extractIP(clientIP)

	// Verificar whitelist
	if rlm.whitelist[ip] {
		return true
	}

	// Verificar límite global
	if !rlm.globalLimiter.Allow() {
		return false
	}

	// Verificar límite por IP
	return rlm.allowForIP(ip)
}

// allowForIP verifica el límite específico para una IP
func (rlm *RateLimiterManager) allowForIP(ip string) bool {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	// Obtener o crear limitador para IP
	ipLimiter, exists := rlm.ipLimiters[ip]
	if !exists {
		switch rlm.config.Type {
		case "token_bucket":
			ipLimiter = NewTokenBucket(rlm.config.PerIPLimit, rlm.config.RefillRate/10) // Menor tasa por IP
		case "sliding_window":
			ipLimiter = NewSlidingWindow(int(rlm.config.PerIPLimit), rlm.config.GetWindowSize())
		default:
			ipLimiter = NewTokenBucket(rlm.config.PerIPLimit, rlm.config.RefillRate/10)
		}
		rlm.ipLimiters[ip] = ipLimiter
	}

	return ipLimiter.Allow()
}

// extractIP extrae la IP real de una dirección que puede incluir puerto
func (rlm *RateLimiterManager) extractIP(clientAddr string) string {
	if host, _, err := net.SplitHostPort(clientAddr); err == nil {
		return host
	}
	return clientAddr
}

// GetStats retorna estadísticas del rate limiter
func (rlm *RateLimiterManager) GetStats() map[string]interface{} {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()

	stats := map[string]interface{}{
		"enabled":      rlm.config.Enabled,
		"type":         rlm.config.Type,
		"global_limit": rlm.config.GlobalLimit,
		"per_ip_limit": rlm.config.PerIPLimit,
		"active_ips":   len(rlm.ipLimiters),
	}

	// Agregar tokens del limitador global si es token bucket
	if tb, ok := rlm.globalLimiter.(*TokenBucket); ok {
		stats["global_tokens"] = tb.GetTokens()
	}

	return stats
}

// Cleanup limpia limitadores inactivos (llamar periódicamente)
func (rlm *RateLimiterManager) Cleanup() {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	// Implementar lógica de limpieza si es necesario
	// Por simplicidad, mantenemos todos los limitadores por ahora
}
