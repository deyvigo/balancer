package breaker

import (
	"sync"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal"
)

// BreakerManager maneja múltiples circuit breakers, uno por backend
type BreakerManager struct {
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
	mu       sync.RWMutex
}

// NewBreakerManager crea un nuevo manager de circuit breakers
func NewBreakerManager(config CircuitBreakerConfig) *BreakerManager {
	return &BreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// GetOrCreateBreaker obtiene o crea un circuit breaker para un backend
func (bm *BreakerManager) GetOrCreateBreaker(backendURL string) *CircuitBreaker {
	bm.mu.RLock()
	breaker, exists := bm.breakers[backendURL]
	bm.mu.RUnlock()

	if exists {
		return breaker
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Double-check después del lock
	if breaker, exists := bm.breakers[backendURL]; exists {
		return breaker
	}

	// Crear nuevo circuit breaker
	breaker = NewCircuitBreaker(bm.config)
	bm.breakers[backendURL] = breaker
	return breaker
}

// CanExecute verifica si se puede ejecutar una llamada a un backend
func (bm *BreakerManager) CanExecute(backendURL string) bool {
	breaker := bm.GetOrCreateBreaker(backendURL)

	// Intentar transición a half-open si es momento
	if decision := breaker.TryHalfOpen(); decision != nil {
		// Log de la decisión si es necesario
		_ = decision
	}

	return breaker.CanExecute()
}

// OnSuccess registra una ejecución exitosa para un backend
func (bm *BreakerManager) OnSuccess(backendURL string) *internal.Decision {
	breaker := bm.GetOrCreateBreaker(backendURL)
	decision := breaker.OnSuccess()

	if decision != nil {
		decision.URL = backendURL
	}

	return decision
}

// OnFailure registra una ejecución fallida para un backend
func (bm *BreakerManager) OnFailure(backendURL string) *internal.Decision {
	breaker := bm.GetOrCreateBreaker(backendURL)
	decision := breaker.OnFailure()

	if decision != nil {
		decision.URL = backendURL
	}

	return decision
}

// OnTimeout registra un timeout para un backend
func (bm *BreakerManager) OnTimeout(backendURL string) *internal.Decision {
	breaker := bm.GetOrCreateBreaker(backendURL)
	decision := breaker.OnTimeout()

	if decision != nil {
		decision.URL = backendURL
	}

	return decision
}

// ResetBreaker resetea un circuit breaker específico
func (bm *BreakerManager) ResetBreaker(backendURL string) *internal.Decision {
	breaker := bm.GetOrCreateBreaker(backendURL)
	decision := breaker.Reset()

	if decision != nil {
		decision.URL = backendURL
	}

	return decision
}

// GetBreakerState obtiene el estado de un circuit breaker
func (bm *BreakerManager) GetBreakerState(backendURL string) (CircuitState, int, float64) {
	breaker := bm.GetOrCreateBreaker(backendURL)
	return breaker.GetState()
}

// GetAllStates obtiene el estado de todos los circuit breakers
func (bm *BreakerManager) GetAllStates() map[string]map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	states := make(map[string]map[string]interface{})
	for url, breaker := range bm.breakers {
		states[url] = breaker.GetStats()
	}

	return states
}

// GetAvailableBackends filtra backends según el estado de sus circuit breakers
func (bm *BreakerManager) GetAvailableBackends(backends []string) []string {
	var available []string

	for _, backend := range backends {
		if bm.CanExecute(backend) {
			available = append(available, backend)
		}
	}

	return available
}

// PerformHealthCheck ejecuta verificaciones periódicas de circuit breakers
func (bm *BreakerManager) PerformHealthCheck() []*internal.Decision {
	var decisions []*internal.Decision

	bm.mu.RLock()
	breakers := make(map[string]*CircuitBreaker)
	for url, breaker := range bm.breakers {
		breakers[url] = breaker
	}
	bm.mu.RUnlock()

	for url, breaker := range breakers {
		if decision := breaker.TryHalfOpen(); decision != nil {
			decision.URL = url
			decisions = append(decisions, decision)
		}
	}

	return decisions
}

// ExecuteWithBreaker ejecuta una función con protección de circuit breaker
func (bm *BreakerManager) ExecuteWithBreaker(backendURL string, fn func() error) (*internal.Decision, error) {
	if !bm.CanExecute(backendURL) {
		return &internal.Decision{
			URL:      backendURL,
			Severity: "warning",
			Reason:   "Circuit breaker is OPEN - request blocked",
			Time:     time.Now(),
		}, &CircuitBreakerOpenError{Backend: backendURL}
	}

	err := fn()
	var decision *internal.Decision

	if err != nil {
		decision = bm.OnFailure(backendURL)
	} else {
		decision = bm.OnSuccess(backendURL)
	}

	return decision, err
}

// CircuitBreakerOpenError error cuando el circuit breaker está abierto
type CircuitBreakerOpenError struct {
	Backend string
}

func (e *CircuitBreakerOpenError) Error() string {
	return "circuit breaker is open for backend: " + e.Backend
}

// IsCircuitBreakerOpen verifica si un error es por circuit breaker abierto
func IsCircuitBreakerOpen(err error) bool {
	_, ok := err.(*CircuitBreakerOpenError)
	return ok
}
