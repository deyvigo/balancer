package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal/breaker"
	"github.com/deyvigo/balanceador/balancer/internal/monitor"
	"github.com/deyvigo/balanceador/balancer/internal/optimizer"
	"github.com/deyvigo/balanceador/balancer/internal/ratelimiter"
)

// LoadBalancingAlgorithm define los algoritmos de balanceo disponibles
type LoadBalancingAlgorithm int

const (
	RoundRobin LoadBalancingAlgorithm = iota
	WeightedRoundRobin
	LeastConnections
)

// ProxyConfig configuración del proxy
type ProxyConfig struct {
	Algorithm     LoadBalancingAlgorithm
	RetryAttempts int
	RetryDelay    time.Duration
	Timeout       time.Duration
}

// LoadBalancer es el proxy reverso principal
type LoadBalancer struct {
	monitor       *monitor.MonitorService
	config        ProxyConfig
	roundRobinIdx uint64
	client        *http.Client
	breakerMgr    *breaker.BreakerManager
	optimizer     *optimizer.WeightOptimizer
	rateLimiter   *ratelimiter.RateLimiterManager
}

// NewLoadBalancer crea una nueva instancia del balanceador
func NewLoadBalancer(monitor *monitor.MonitorService, config ProxyConfig, breakerConfig breaker.CircuitBreakerConfig, optimizerConfig optimizer.WeightCalculationConfig, rateLimitConfig ratelimiter.RateLimiterConfig) *LoadBalancer {
	return &LoadBalancer{
		monitor: monitor,
		config:  config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		breakerMgr:  breaker.NewBreakerManager(breakerConfig),
		optimizer:   optimizer.NewWeightOptimizer(optimizerConfig),
		rateLimiter: ratelimiter.NewRateLimiterManager(rateLimitConfig),
	}
}

// GetOptimizer devuelve el optimizador de pesos
func (lb *LoadBalancer) GetOptimizer() *optimizer.WeightOptimizer {
	return lb.optimizer
}

// GetRateLimitStats devuelve estadísticas del rate limiter
func (lb *LoadBalancer) GetRateLimitStats() map[string]interface{} {
	return lb.rateLimiter.GetStats()
}

// Handler es el handler HTTP principal que maneja todas las peticiones
func (lb *LoadBalancer) Handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Verificar rate limiting
	clientIP := r.RemoteAddr
	if !lb.rateLimiter.Allow(clientIP) {
		log.Printf("[proxy] rate limit exceeded for IP: %s", clientIP)
		http.Error(w, "Rate Limit Exceeded", http.StatusTooManyRequests)
		return
	}

	// Obtener backend disponible
	backend, err := lb.selectBackend()
	if err != nil {
		log.Printf("[proxy] no hay backends disponibles: %v", err)
		http.Error(w, "Service Temporarily Unavailable", http.StatusServiceUnavailable)
		return
	}

	// Intentar proxy con retries
	err = lb.proxyRequest(w, r, backend)
	if err != nil {
		log.Printf("[proxy] error al hacer proxy a %s: %v", backend, err)

		// Intentar con otro backend si hay retries disponibles
		if lb.config.RetryAttempts > 0 {
			for i := 0; i < lb.config.RetryAttempts; i++ {
				time.Sleep(lb.config.RetryDelay)

				retryBackend, retryErr := lb.selectBackend()
				if retryErr != nil {
					continue
				}

				// Evitar el mismo backend que falló
				if retryBackend == backend {
					continue
				}

				err = lb.proxyRequest(w, r, retryBackend)
				if err == nil {
					log.Printf("[proxy] retry exitoso a %s después de %d intentos", retryBackend, i+1)
					return
				}
			}
		}

		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	duration := time.Since(start)
	log.Printf("[proxy] %s %s -> %s (%v)", r.Method, r.URL.Path, backend, duration)
}

// selectBackend selecciona un backend basado en el algoritmo configurado
func (lb *LoadBalancer) selectBackend() (string, error) {
	backends := lb.monitor.GetAliveBackends()
	if len(backends) == 0 {
		return "", fmt.Errorf("no hay backends disponibles")
	}

	// Filtrar backends por circuit breaker
	availableBackends := lb.breakerMgr.GetAvailableBackends(backends)
	if len(availableBackends) == 0 {
		return "", fmt.Errorf("no hay backends disponibles (circuit breakers abiertos)")
	}

	switch lb.config.Algorithm {
	case RoundRobin:
		return lb.roundRobinSelect(availableBackends), nil
	case WeightedRoundRobin:
		return lb.weightedRoundRobinSelect(availableBackends), nil
	case LeastConnections:
		// TODO: implementar least connections
		return lb.roundRobinSelect(availableBackends), nil
	default:
		return lb.roundRobinSelect(availableBackends), nil
	}
}

// roundRobinSelect implementa round robin simple
func (lb *LoadBalancer) roundRobinSelect(backends []string) string {
	if len(backends) == 0 {
		return ""
	}

	idx := atomic.AddUint64(&lb.roundRobinIdx, 1)
	return backends[(idx-1)%uint64(len(backends))]
}

// weightedRoundRobinSelect implementa weighted round robin basado en métricas adaptativas
func (lb *LoadBalancer) weightedRoundRobinSelect(backends []string) string {
	if len(backends) == 0 {
		return ""
	}

	// Actualizar pesos adaptativos con métricas actuales
	metrics := lb.monitor.SnapshotMetrics()
	adaptiveWeights := lb.optimizer.UpdateWeights(metrics)

	// Construir mapa de pesos para backends disponibles
	weights := make(map[string]float64)
	totalWeight := 0.0

	for _, backend := range backends {
		// Usar peso adaptativo si existe, sino peso por defecto
		if weight, exists := adaptiveWeights[backend]; exists && weight > 0 {
			weights[backend] = weight
			totalWeight += weight
		} else {
			// Peso por defecto para backends sin métricas
			weights[backend] = 1.0
			totalWeight += 1.0
		}
	}

	// Si no hay peso total, usar round robin
	if totalWeight == 0 {
		return lb.roundRobinSelect(backends)
	}

	// Selección weighted random
	target := atomic.AddUint64(&lb.roundRobinIdx, 1)
	normalized := float64(target%1000) / 1000.0 * totalWeight

	cumulative := 0.0
	for backend, weight := range weights {
		cumulative += weight
		if normalized <= cumulative {
			return backend
		}
	}

	// Fallback
	return backends[0]
}

// proxyRequest hace el proxy de la petición al backend seleccionado
func (lb *LoadBalancer) proxyRequest(w http.ResponseWriter, r *http.Request, backendURL string) error {
	target, err := url.Parse(backendURL)
	if err != nil {
		return fmt.Errorf("URL inválida %s: %w", backendURL, err)
	}

	var proxyError error
	var statusCode int

	// Usar circuit breaker para ejecutar la petición
	decision, err := lb.breakerMgr.ExecuteWithBreaker(backendURL, func() error {
		// Crear proxy reverso
		proxy := httputil.NewSingleHostReverseProxy(target)

		// Configurar director para modificar headers
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			req.Header.Set("X-Origin-Host", target.Host)
			req.Header.Set("X-Forwarded-Proto", "http")
			if clientIP := getClientIP(r); clientIP != "" {
				req.Header.Set("X-Forwarded-For", clientIP)
			}
		}

		// Capturar respuesta para registrar estado
		responseWriter := &responseCapture{ResponseWriter: w}

		// Configurar manejo de errores
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[proxy] error handler para %s: %v", backendURL, err)
			proxyError = err
			statusCode = http.StatusBadGateway
		}

		// Ejecutar proxy
		proxy.ServeHTTP(responseWriter, r)
		statusCode = responseWriter.statusCode

		// Considerar error si status >= 500 (error del servidor)
		if statusCode >= 500 {
			proxyError = fmt.Errorf("server error: status %d", statusCode)
		}

		return proxyError
	})

	// Log de la decisión del circuit breaker si existe
	if decision != nil && decision.Severity == "critical" {
		log.Printf("[circuit-breaker] %s", decision.Reason)
	}

	return err
}

// responseCapture captura el código de estado de la respuesta
type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

// GetCircuitBreakerStats obtiene estadísticas de todos los circuit breakers
func (lb *LoadBalancer) GetCircuitBreakerStats() map[string]map[string]interface{} {
	return lb.breakerMgr.GetAllStates()
}

// ResetCircuitBreaker resetea un circuit breaker específico
func (lb *LoadBalancer) ResetCircuitBreaker(backendURL string) {
	if decision := lb.breakerMgr.ResetBreaker(backendURL); decision != nil {
		log.Printf("[circuit-breaker] %s", decision.Reason)
	}
}

// StartOptimizer inicia el optimizador de pesos adaptativos
func (lb *LoadBalancer) StartOptimizer() {
	lb.optimizer.Start()
}

// StopOptimizer detiene el optimizador de pesos adaptativos
func (lb *LoadBalancer) StopOptimizer() {
	lb.optimizer.Stop()
}

// GetAdaptiveWeights obtiene pesos adaptativos calculados
func (lb *LoadBalancer) GetAdaptiveWeights() map[string]*optimizer.BackendWeight {
	return lb.optimizer.GetWeights()
}

// UpdateAdaptiveWeights actualiza pesos basado en métricas actuales
func (lb *LoadBalancer) UpdateAdaptiveWeights() {
	metrics := lb.monitor.SnapshotMetrics()
	lb.optimizer.UpdateWeights(metrics)
}

// getClientIP extrae la IP del cliente de los headers
func getClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		return strings.Split(xForwardedFor, ",")[0]
	}

	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	return r.RemoteAddr
}
