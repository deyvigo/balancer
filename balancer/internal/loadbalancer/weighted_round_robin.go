package loadbalancer

import (
	"math"
	"sync"

	"github.com/deyvigo/balanceador/balancer/internal"
)

// WeightedRoundRobin implementa un balanceador con Round Robin Ponderado
// Los pesos se calculan dinámicamente según las métricas de cada backend
type WeightedRoundRobin struct {
	mu            sync.RWMutex
	backends      []BackendWeight
	currentIndex  int
	currentWeight int
	maxWeight     int
	gcd           int
}

// BackendWeight representa un backend con su peso calculado
type BackendWeight struct {
	ID     int
	URL    string
	Weight int
}

// NewWeightedRoundRobin crea una nueva instancia del balanceador
func NewWeightedRoundRobin() *WeightedRoundRobin {
	return &WeightedRoundRobin{
		backends:      make([]BackendWeight, 0),
		currentIndex:  -1,
		currentWeight: 0,
	}
}

// UpdateMetrics actualiza las métricas y recalcula los pesos dinámicamente
func (wrr *WeightedRoundRobin) UpdateMetrics(metrics []internal.Metrics) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	wrr.backends = make([]BackendWeight, 0, len(metrics))

	// Calcular peso para cada backend
	for _, m := range metrics {
		weight := wrr.calculateWeight(m)
		if weight > 0 { // Solo incluir backends disponibles
			wrr.backends = append(wrr.backends, BackendWeight{
				ID:     m.Id,
				URL:    m.URL,
				Weight: weight,
			})
		}
	}

	// Recalcular valores para el algoritmo
	if len(wrr.backends) > 0 {
		wrr.maxWeight = wrr.getMaxWeight()
		wrr.gcd = wrr.getGCD()
		wrr.currentIndex = -1
		wrr.currentWeight = 0
	}
}

// calculateWeight calcula el peso de un backend según sus métricas
// Factores:
// - Latencia (EMAMs): menor latencia = mayor peso
// - Tasa de error (ErrorRate): menos errores = mayor peso
// - Estado del circuito: OPEN = 0, HALF_OPEN = peso mínimo, CLOSED = normal
// - Alive: si está caído = 0
func (wrr *WeightedRoundRobin) calculateWeight(m internal.Metrics) int {
	// Backend caído o circuito abierto = sin peso
	if !m.Alive || m.CircuitState == internal.StateOpen {
		return 0
	}

	// Backend en prueba (HALF_OPEN) = peso mínimo para testear
	if m.CircuitState == internal.StateHalfOpen {
		return 1
	}

	// Peso base
	baseWeight := 100.0

	// Factor de latencia: menor latencia = mayor peso
	// Fórmula: 1 / (1 + latencia/100)
	// Ejemplo: 10ms → 0.91, 50ms → 0.67, 200ms → 0.33
	latencyFactor := 1.0
	if m.EMAMs > 0 {
		latencyFactor = 1.0 / (1.0 + m.EMAMs/100.0)
	}

	// Factor de error: menos errores = mayor peso
	// ErrorRate está entre 0 y 1
	// Ejemplo: 0% error → 1.0, 20% error → 0.8, 50% error → 0.5
	errorFactor := 1.0 - m.ErrorRate

	// Calcular peso final combinando ambos factores
	finalWeight := baseWeight * latencyFactor * errorFactor

	// Asegurar que el peso esté entre 1 y 100
	weight := int(math.Max(1, math.Min(100, finalWeight)))

	return weight
}

// NextBackend selecciona el siguiente backend usando Weighted Round Robin
// Retorna la URL del backend seleccionado y true si hay backends disponibles
func (wrr *WeightedRoundRobin) NextBackend() (string, bool) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.backends) == 0 {
		return "", false
	}

	// Algoritmo Weighted Round Robin clásico
	for {
		wrr.currentIndex = (wrr.currentIndex + 1) % len(wrr.backends)

		if wrr.currentIndex == 0 {
			wrr.currentWeight = wrr.currentWeight - wrr.gcd
			if wrr.currentWeight <= 0 {
				wrr.currentWeight = wrr.maxWeight
				if wrr.currentWeight == 0 {
					return "", false
				}
			}
		}

		if wrr.backends[wrr.currentIndex].Weight >= wrr.currentWeight {
			return wrr.backends[wrr.currentIndex].URL, true
		}
	}
}

// GetBackendWeights devuelve los pesos actuales (para debugging/logging)
func (wrr *WeightedRoundRobin) GetBackendWeights() []BackendWeight {
	wrr.mu.RLock()
	defer wrr.mu.RUnlock()

	weights := make([]BackendWeight, len(wrr.backends))
	copy(weights, wrr.backends)
	return weights
}

// getMaxWeight encuentra el peso máximo entre todos los backends
func (wrr *WeightedRoundRobin) getMaxWeight() int {
	max := 0
	for _, b := range wrr.backends {
		if b.Weight > max {
			max = b.Weight
		}
	}
	return max
}

// getGCD calcula el máximo común divisor de todos los pesos
func (wrr *WeightedRoundRobin) getGCD() int {
	if len(wrr.backends) == 0 {
		return 0
	}

	divisor := wrr.backends[0].Weight
	for i := 1; i < len(wrr.backends); i++ {
		divisor = gcd(divisor, wrr.backends[i].Weight)
	}
	return divisor
}

// gcd calcula el máximo común divisor usando el algoritmo de Euclides
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
