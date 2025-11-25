package optimizer

import (
	"log"
	"math"
	"sync"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal"
)

// WeightCalculationConfig configuración para el cálculo de pesos adaptativos
type WeightCalculationConfig struct {
	Enabled               bool    `json:"enabled"`
	MinWeight             float64 `json:"min_weight"`        // Peso mínimo (ej: 0.1)
	MaxWeight             float64 `json:"max_weight"`        // Peso máximo (ej: 5.0)
	LatencyWeight         float64 `json:"latency_weight"`    // Importancia de latencia (0-1)
	ErrorRateWeight       float64 `json:"error_rate_weight"` // Importancia de error rate (0-1)
	AdaptationSpeed       float64 `json:"adaptation_speed"`  // Velocidad de adaptación (0-1)
	LatencyTargetMs       float64 `json:"latency_target_ms"` // Latencia objetivo en ms
	MaxErrorRate          float64 `json:"max_error_rate"`    // Error rate máximo aceptable
	UpdateIntervalSeconds int     `json:"update_interval_s"` // Intervalo de actualización
}

// DefaultWeightCalculationConfig configuración por defecto
func DefaultWeightCalculationConfig() WeightCalculationConfig {
	return WeightCalculationConfig{
		Enabled:               true,
		MinWeight:             0.1,
		MaxWeight:             5.0,
		LatencyWeight:         0.6,
		ErrorRateWeight:       0.4,
		AdaptationSpeed:       0.1,
		LatencyTargetMs:       100.0,
		MaxErrorRate:          0.1,
		UpdateIntervalSeconds: 10,
	}
}

// BackendWeight representa el peso calculado para un backend
type BackendWeight struct {
	URL               string    `json:"url"`
	Weight            float64   `json:"weight"`
	LatencyScore      float64   `json:"latency_score"`
	ErrorRateScore    float64   `json:"error_rate_score"`
	CombinedScore     float64   `json:"combined_score"`
	LastUpdated       time.Time `json:"last_updated"`
	PreviousWeight    float64   `json:"previous_weight"`
	WeightChange      float64   `json:"weight_change"`
	RecommendedWeight float64   `json:"recommended_weight"`
}

// WeightOptimizer calcula pesos adaptativos basados en métricas
type WeightOptimizer struct {
	config   WeightCalculationConfig
	weights  map[string]*BackendWeight
	mu       sync.RWMutex
	stopChan chan struct{}
	running  bool
}

// NewWeightOptimizer crea un nuevo optimizador de pesos
func NewWeightOptimizer(config WeightCalculationConfig) *WeightOptimizer {
	return &WeightOptimizer{
		config:   config,
		weights:  make(map[string]*BackendWeight),
		stopChan: make(chan struct{}),
	}
}

// Start inicia el proceso de optimización de pesos
func (wo *WeightOptimizer) Start() {
	if !wo.config.Enabled {
		log.Printf("[optimizer] Adaptive weights disabled")
		return
	}

	wo.mu.Lock()
	if wo.running {
		wo.mu.Unlock()
		return
	}
	wo.running = true
	wo.mu.Unlock()

	go wo.optimizationLoop()
	log.Printf("[optimizer] Adaptive weight optimization started (interval: %ds)", wo.config.UpdateIntervalSeconds)
}

// Stop detiene el proceso de optimización
func (wo *WeightOptimizer) Stop() {
	wo.mu.Lock()
	defer wo.mu.Unlock()

	if wo.running {
		close(wo.stopChan)
		wo.running = false
		log.Printf("[optimizer] Adaptive weight optimization stopped")
	}
}

// UpdateWeights actualiza los pesos basado en métricas actuales
func (wo *WeightOptimizer) UpdateWeights(metrics []internal.Metrics) map[string]float64 {
	if !wo.config.Enabled || len(metrics) == 0 {
		return wo.getDefaultWeights(metrics)
	}

	wo.mu.Lock()
	defer wo.mu.Unlock()

	newWeights := make(map[string]float64)
	now := time.Now()

	for _, metric := range metrics {
		if !metric.Alive {
			// Backend muerto = peso mínimo
			newWeights[metric.URL] = wo.config.MinWeight
			continue
		}

		// Calcular scores individuales
		latencyScore := wo.calculateLatencyScore(metric.EMAMs)
		errorRateScore := wo.calculateErrorRateScore(metric.ErrorRate)

		// Score combinado
		combinedScore := (wo.config.LatencyWeight * latencyScore) +
			(wo.config.ErrorRateWeight * errorRateScore)

		// Convertir score a peso
		recommendedWeight := wo.scoreToWeight(combinedScore)

		// Aplicar suavizado si existe peso previo
		var finalWeight float64
		if prevWeight, exists := wo.weights[metric.URL]; exists {
			finalWeight = wo.smoothWeight(prevWeight.Weight, recommendedWeight)
		} else {
			finalWeight = recommendedWeight
		}

		// Clamp al rango permitido
		finalWeight = math.Max(wo.config.MinWeight, math.Min(wo.config.MaxWeight, finalWeight))

		// Actualizar estructura de peso
		previousWeight := 1.0
		if prev, exists := wo.weights[metric.URL]; exists {
			previousWeight = prev.Weight
		}

		wo.weights[metric.URL] = &BackendWeight{
			URL:               metric.URL,
			Weight:            finalWeight,
			LatencyScore:      latencyScore,
			ErrorRateScore:    errorRateScore,
			CombinedScore:     combinedScore,
			LastUpdated:       now,
			PreviousWeight:    previousWeight,
			WeightChange:      finalWeight - previousWeight,
			RecommendedWeight: recommendedWeight,
		}

		newWeights[metric.URL] = finalWeight
	}

	wo.logWeightChanges()
	return newWeights
}

// calculateLatencyScore calcula score basado en latencia (mayor = peor)
func (wo *WeightOptimizer) calculateLatencyScore(latencyMs float64) float64 {
	if latencyMs <= 0 {
		return 1.0
	}

	// Score inverso: menor latencia = mayor score
	// Usar función exponencial para penalizar latencias altas
	ratio := latencyMs / wo.config.LatencyTargetMs
	score := math.Exp(-ratio * 2) // Factor 2 para curva más pronunciada

	return math.Max(0.01, math.Min(1.0, score))
}

// calculateErrorRateScore calcula score basado en error rate (mayor = peor)
func (wo *WeightOptimizer) calculateErrorRateScore(errorRate float64) float64 {
	if errorRate <= 0 {
		return 1.0
	}

	if errorRate >= wo.config.MaxErrorRate {
		return 0.01 // Casi cero pero no cero
	}

	// Score lineal inverso
	score := 1.0 - (errorRate / wo.config.MaxErrorRate)
	return math.Max(0.01, math.Min(1.0, score))
}

// scoreToWeight convierte un score combinado (0-1) a peso (min-max)
func (wo *WeightOptimizer) scoreToWeight(score float64) float64 {
	// Mapeo lineal de score [0,1] a peso [min,max]
	weightRange := wo.config.MaxWeight - wo.config.MinWeight
	weight := wo.config.MinWeight + (score * weightRange)

	return math.Max(wo.config.MinWeight, math.Min(wo.config.MaxWeight, weight))
}

// smoothWeight aplica suavizado exponencial al peso
func (wo *WeightOptimizer) smoothWeight(previousWeight, recommendedWeight float64) float64 {
	// EMA: new = α * recommended + (1-α) * previous
	alpha := wo.config.AdaptationSpeed
	return alpha*recommendedWeight + (1-alpha)*previousWeight
}

// getDefaultWeights devuelve pesos por defecto (todos iguales)
func (wo *WeightOptimizer) getDefaultWeights(metrics []internal.Metrics) map[string]float64 {
	weights := make(map[string]float64)
	defaultWeight := 1.0

	for _, metric := range metrics {
		if metric.Alive {
			weights[metric.URL] = defaultWeight
		} else {
			weights[metric.URL] = wo.config.MinWeight
		}
	}

	return weights
}

// GetWeights devuelve los pesos actuales
func (wo *WeightOptimizer) GetWeights() map[string]*BackendWeight {
	wo.mu.RLock()
	defer wo.mu.RUnlock()

	weights := make(map[string]*BackendWeight)
	for url, weight := range wo.weights {
		// Hacer copia para evitar modificaciones concurrentes
		weights[url] = &BackendWeight{
			URL:               weight.URL,
			Weight:            weight.Weight,
			LatencyScore:      weight.LatencyScore,
			ErrorRateScore:    weight.ErrorRateScore,
			CombinedScore:     weight.CombinedScore,
			LastUpdated:       weight.LastUpdated,
			PreviousWeight:    weight.PreviousWeight,
			WeightChange:      weight.WeightChange,
			RecommendedWeight: weight.RecommendedWeight,
		}
	}

	return weights
}

// GetWeightForBackend devuelve el peso de un backend específico
func (wo *WeightOptimizer) GetWeightForBackend(url string) float64 {
	wo.mu.RLock()
	defer wo.mu.RUnlock()

	if weight, exists := wo.weights[url]; exists {
		return weight.Weight
	}
	return 1.0 // peso por defecto
}

// optimizationLoop loop principal de optimización
func (wo *WeightOptimizer) optimizationLoop() {
	ticker := time.NewTicker(time.Duration(wo.config.UpdateIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wo.stopChan:
			return
		case <-ticker.C:
			// Las métricas se actualizan desde el exterior via UpdateWeights
			// Este loop podría hacer limpieza o análisis adicional
		}
	}
}

// logWeightChanges registra cambios significativos en los pesos
func (wo *WeightOptimizer) logWeightChanges() {
	for url, weight := range wo.weights {
		if math.Abs(weight.WeightChange) > 0.1 { // Log solo cambios > 10%
			log.Printf("[optimizer] %s weight: %.2f -> %.2f (Δ%.2f) | latency_score: %.2f, error_score: %.2f",
				url, weight.PreviousWeight, weight.Weight, weight.WeightChange,
				weight.LatencyScore, weight.ErrorRateScore)
		}
	}
}

// GetStats devuelve estadísticas del optimizador
func (wo *WeightOptimizer) GetStats() map[string]interface{} {
	wo.mu.RLock()
	defer wo.mu.RUnlock()

	stats := map[string]interface{}{
		"enabled":        wo.config.Enabled,
		"running":        wo.running,
		"backends_count": len(wo.weights),
		"config":         wo.config,
	}

	if len(wo.weights) > 0 {
		var totalWeight, avgLatencyScore, avgErrorScore float64
		for _, weight := range wo.weights {
			totalWeight += weight.Weight
			avgLatencyScore += weight.LatencyScore
			avgErrorScore += weight.ErrorRateScore
		}

		count := float64(len(wo.weights))
		stats["total_weight"] = totalWeight
		stats["avg_weight"] = totalWeight / count
		stats["avg_latency_score"] = avgLatencyScore / count
		stats["avg_error_score"] = avgErrorScore / count
	}

	return stats
}
