package ratelimiter

import (
	"sync"
	"time"
)

// TokenBucket implementa el algoritmo Token Bucket para rate limiting
type TokenBucket struct {
	capacity   int64      // Capacidad máxima del bucket
	tokens     int64      // Tokens actuales
	refillRate int64      // Tokens por segundo
	lastRefill time.Time  // Último refill
	mu         sync.Mutex // Mutex para concurrencia
}

// NewTokenBucket crea un nuevo token bucket
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow verifica si se puede permitir una petición
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Calcular tokens a agregar
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate

	// Refill bucket sin exceder capacidad
	tb.tokens += tokensToAdd
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	// Verificar si hay tokens disponibles
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// GetTokens retorna la cantidad actual de tokens
func (tb *TokenBucket) GetTokens() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens
}

// SlidingWindow implementa rate limiting con ventana deslizante
type SlidingWindow struct {
	requests []time.Time   // Timestamps de peticiones
	limit    int           // Límite de peticiones
	window   time.Duration // Tamaño de ventana
	mu       sync.Mutex    // Mutex para concurrencia
}

// NewSlidingWindow crea una nueva ventana deslizante
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

// Allow verifica si se puede permitir una petición
func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// Limpiar peticiones antiguas
	var validRequests []time.Time
	for _, req := range sw.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	sw.requests = validRequests

	// Verificar límite
	if len(sw.requests) >= sw.limit {
		return false
	}

	// Agregar petición actual
	sw.requests = append(sw.requests, now)
	return true
}

// GetCurrentRate retorna la tasa actual de peticiones
func (sw *SlidingWindow) GetCurrentRate() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	count := 0
	for _, req := range sw.requests {
		if req.After(cutoff) {
			count++
		}
	}

	return count
}
