package breaker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal"
)

// CircuitState representa los estados del circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig configuración del circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold   int           `json:"failure_threshold"`    // Número de fallos para abrir circuito
	ErrorRateThreshold float64       `json:"error_rate_threshold"` // Porcentaje de error para abrir (0-1)
	OpenTimeout        time.Duration `json:"open_timeout_s"`       // Tiempo antes de intentar half-open
	HalfOpenMaxCalls   int           `json:"half_open_max_calls"`  // Máx llamadas en estado half-open
	MinRequestCount    int           `json:"min_request_count"`    // Mín requests antes de evaluar
}

// DefaultCircuitBreakerConfig configuración por defecto
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:   5,
		ErrorRateThreshold: 0.5, // 50%
		OpenTimeout:        30 * time.Second,
		HalfOpenMaxCalls:   3,
		MinRequestCount:    5,
	}
}

// CircuitBreaker implementa el patrón circuit breaker
type CircuitBreaker struct {
	config        CircuitBreakerConfig
	state         CircuitState
	failureCount  int
	lastFailTime  time.Time
	nextAttempt   time.Time
	halfOpenCalls int
	totalCalls    int
	successCount  int
	mu            sync.RWMutex
}

// NewCircuitBreaker crea un nuevo circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// CanExecute determina si se puede ejecutar una llamada
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return time.Now().After(cb.nextAttempt)
	case StateHalfOpen:
		return cb.halfOpenCalls < cb.config.HalfOpenMaxCalls
	}
	return false
}

// OnSuccess registra una ejecución exitosa
func (cb *CircuitBreaker) OnSuccess() *internal.Decision {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalCalls++
	cb.successCount++

	switch cb.state {
	case StateHalfOpen:
		cb.halfOpenCalls++
		if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			// Todas las llamadas half-open fueron exitosas, cerrar circuito
			return cb.transitionToClosed()
		}
		return &internal.Decision{
			Severity: "info",
			Reason:   "Circuit breaker half-open: success recorded",
			Time:     time.Now(),
		}
	case StateClosed:
		// Reset failure count en estado cerrado tras éxito
		cb.failureCount = 0
		return nil
	default:
		return nil
	}
}

// OnFailure registra una ejecución fallida
func (cb *CircuitBreaker) OnFailure() *internal.Decision {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalCalls++
	cb.failureCount++
	cb.lastFailTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.shouldOpen() {
			return cb.transitionToOpen()
		}
		return &internal.Decision{
			Severity: "warning",
			Reason:   fmt.Sprintf("Circuit breaker failure count: %d/%d", cb.failureCount, cb.config.FailureThreshold),
			Time:     time.Now(),
		}
	case StateHalfOpen:
		// Cualquier fallo en half-open vuelve a open
		return cb.transitionToOpen()
	default:
		return nil
	}
}

// OnTimeout registra un timeout (tratado como fallo)
func (cb *CircuitBreaker) OnTimeout() *internal.Decision {
	return cb.OnFailure()
}

// shouldOpen determina si el circuito debe abrirse
func (cb *CircuitBreaker) shouldOpen() bool {
	// Verificar threshold de fallos consecutivos
	if cb.failureCount >= cb.config.FailureThreshold {
		return true
	}

	// Verificar threshold de error rate si hay suficientes requests
	if cb.totalCalls >= cb.config.MinRequestCount {
		errorRate := float64(cb.failureCount) / float64(cb.totalCalls)
		if errorRate >= cb.config.ErrorRateThreshold {
			return true
		}
	}

	return false
}

// transitionToOpen transiciona el circuito a estado abierto
func (cb *CircuitBreaker) transitionToOpen() *internal.Decision {
	cb.state = StateOpen
	cb.nextAttempt = time.Now().Add(cb.config.OpenTimeout)

	log.Printf("[circuit-breaker] Circuit OPENED - failures: %d, error_rate: %.2f",
		cb.failureCount, float64(cb.failureCount)/float64(cb.totalCalls))

	return &internal.Decision{
		Severity: "critical",
		Reason:   fmt.Sprintf("Circuit breaker opened - %d failures, error rate: %.2f", cb.failureCount, float64(cb.failureCount)/float64(cb.totalCalls)),
		Time:     time.Now(),
	}
}

// transitionToClosed transiciona el circuito a estado cerrado
func (cb *CircuitBreaker) transitionToClosed() *internal.Decision {
	cb.state = StateClosed
	cb.failureCount = 0
	cb.halfOpenCalls = 0
	cb.totalCalls = 0
	cb.successCount = 0

	log.Printf("[circuit-breaker] Circuit CLOSED - recovery successful")

	return &internal.Decision{
		Severity: "info",
		Reason:   "Circuit breaker closed - service recovered",
		Time:     time.Now(),
	}
}

// TryHalfOpen intenta transicionar a half-open si es tiempo
func (cb *CircuitBreaker) TryHalfOpen() *internal.Decision {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Now().After(cb.nextAttempt) {
		cb.state = StateHalfOpen
		cb.halfOpenCalls = 0

		log.Printf("[circuit-breaker] Circuit HALF-OPEN - testing service")

		return &internal.Decision{
			Severity: "info",
			Reason:   "Circuit breaker half-open - testing backend",
			Time:     time.Now(),
		}
	}
	return nil
}

// GetState devuelve el estado actual del circuit breaker
func (cb *CircuitBreaker) GetState() (CircuitState, int, float64) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var errorRate float64
	if cb.totalCalls > 0 {
		errorRate = float64(cb.failureCount) / float64(cb.totalCalls)
	}

	return cb.state, cb.failureCount, errorRate
}

// Reset resetea el circuit breaker al estado inicial
func (cb *CircuitBreaker) Reset() *internal.Decision {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.halfOpenCalls = 0
	cb.totalCalls = 0
	cb.successCount = 0

	log.Printf("[circuit-breaker] Circuit RESET from %s to CLOSED", oldState)

	return &internal.Decision{
		Severity: "info",
		Reason:   fmt.Sprintf("Circuit breaker manually reset from %s", oldState),
		Time:     time.Now(),
	}
}

// GetStats devuelve estadísticas del circuit breaker
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var errorRate float64
	if cb.totalCalls > 0 {
		errorRate = float64(cb.failureCount) / float64(cb.totalCalls)
	}

	return map[string]interface{}{
		"state":           cb.state.String(),
		"failure_count":   cb.failureCount,
		"total_calls":     cb.totalCalls,
		"success_count":   cb.successCount,
		"error_rate":      errorRate,
		"half_open_calls": cb.halfOpenCalls,
		"next_attempt":    cb.nextAttempt.Format(time.RFC3339),
		"last_fail_time":  cb.lastFailTime.Format(time.RFC3339),
	}
}
