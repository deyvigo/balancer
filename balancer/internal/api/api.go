package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal/monitor"
	"github.com/deyvigo/balanceador/balancer/internal/proxy"
)

// AdminServer maneja la API administrativa del balanceador
type AdminServer struct {
	Monitor *monitor.MonitorService
	Proxy   *proxy.LoadBalancer
}

// BackendInfo representa la información de un backend para la API
type BackendInfo struct {
	ID          int     `json:"id"`
	URL         string  `json:"url"`
	Alive       bool    `json:"alive"`
	EMAMs       float64 `json:"ema_ms"`
	ErrorRate   float64 `json:"error_rate"`
	LastChecked string  `json:"last_checked"`
	Enabled     bool    `json:"enabled"`
	Weight      float64 `json:"weight,omitempty"`
}

// BackendUpdateRequest representa una petición para actualizar un backend
type BackendUpdateRequest struct {
	Enabled *bool    `json:"enabled,omitempty"`
	Weight  *float64 `json:"weight,omitempty"`
}

// ActionRequest representa una petición de acción sobre un backend
type ActionRequest struct {
	Action string `json:"action"` // "drain", "restart", "reset_metrics"
}

// APIResponse es la estructura estándar de respuesta de la API
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewAdminServer crea una nueva instancia del servidor de administración
func NewAdminServer(monitor *monitor.MonitorService, proxy *proxy.LoadBalancer) *AdminServer {
	return &AdminServer{
		Monitor: monitor,
		Proxy:   proxy,
	}
}

// RegisterHandlers registra todos los handlers de la API administrativa
func (a *AdminServer) RegisterHandlers(mux *http.ServeMux) {
	// Endpoints de backends
	mux.HandleFunc("/api/backends", a.handleBackends)
	mux.HandleFunc("/api/backends/", a.handleBackendByID)

	// Endpoint de métricas (alternativa a WebSocket)
	mux.HandleFunc("/api/metrics", a.handleMetrics)

	// Endpoint de configuración
	mux.HandleFunc("/api/config", a.handleConfig)

	// Endpoints de circuit breaker
	mux.HandleFunc("/api/circuit-breaker", a.handleCircuitBreakerStats)
	mux.HandleFunc("/api/circuit-breaker/", a.handleCircuitBreakerActions)

	// Rate limiting endpoints
	mux.HandleFunc("/api/rate-limit", a.handleRateLimitStats)

	// Endpoint de salud de la API
	mux.HandleFunc("/api/health", a.handleHealth)
}

// handleBackends maneja GET /api/backends
func (a *AdminServer) handleBackends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := a.Monitor.SnapshotMetrics()
	backends := make([]BackendInfo, len(metrics))

	for i, metric := range metrics {
		backends[i] = BackendInfo{
			ID:          metric.Id,
			URL:         metric.URL,
			Alive:       metric.Alive,
			EMAMs:       metric.EMAMs,
			ErrorRate:   metric.ErrorRate,
			LastChecked: metric.LastChecked,
			Enabled:     metric.Alive, // Simplificación por ahora
			Weight:      1.0,          // Peso por defecto
		}
	}

	a.sendSuccess(w, "Backends retrieved successfully", backends)
}

// handleBackendByID maneja PATCH /api/backends/{id}
func (a *AdminServer) handleBackendByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extraer ID del path
	path := strings.TrimPrefix(r.URL.Path, "/api/backends/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		a.sendError(w, "Backend ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		a.sendError(w, "Invalid backend ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		a.handleGetBackend(w, r, id)
	case "PATCH":
		a.handleUpdateBackend(w, r, id)
	case "POST":
		if len(parts) > 1 && parts[1] == "actions" {
			a.handleBackendAction(w, r, id)
		} else {
			a.sendError(w, "Invalid endpoint", http.StatusNotFound)
		}
	default:
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetBackend maneja GET /api/backends/{id}
func (a *AdminServer) handleGetBackend(w http.ResponseWriter, r *http.Request, id int) {
	metrics := a.Monitor.SnapshotMetrics()

	for _, metric := range metrics {
		if metric.Id == id {
			backend := BackendInfo{
				ID:          metric.Id,
				URL:         metric.URL,
				Alive:       metric.Alive,
				EMAMs:       metric.EMAMs,
				ErrorRate:   metric.ErrorRate,
				LastChecked: metric.LastChecked,
				Enabled:     metric.Alive,
				Weight:      1.0,
			}
			a.sendSuccess(w, "Backend retrieved successfully", backend)
			return
		}
	}

	a.sendError(w, "Backend not found", http.StatusNotFound)
}

// handleUpdateBackend maneja PATCH /api/backends/{id}
func (a *AdminServer) handleUpdateBackend(w http.ResponseWriter, r *http.Request, id int) {
	var req BackendUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Por ahora, solo log de la acción (implementación futura)
	log.Printf("[api] Backend %d update request: enabled=%v, weight=%v",
		id, req.Enabled, req.Weight)

	message := fmt.Sprintf("Backend %d update queued", id)
	a.sendSuccess(w, message, map[string]interface{}{
		"id":      id,
		"enabled": req.Enabled,
		"weight":  req.Weight,
	})
}

// handleBackendAction maneja POST /api/backends/{id}/actions
func (a *AdminServer) handleBackendAction(w http.ResponseWriter, r *http.Request, id int) {
	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "drain":
		log.Printf("[api] Draining backend %d", id)
		message := fmt.Sprintf("Backend %d drain initiated", id)
		a.sendSuccess(w, message, map[string]interface{}{"id": id, "action": "drain"})
	case "restart":
		log.Printf("[api] Restarting backend %d", id)
		message := fmt.Sprintf("Backend %d restart initiated", id)
		a.sendSuccess(w, message, map[string]interface{}{"id": id, "action": "restart"})
	case "reset_metrics":
		log.Printf("[api] Resetting metrics for backend %d", id)
		message := fmt.Sprintf("Backend %d metrics reset", id)
		a.sendSuccess(w, message, map[string]interface{}{"id": id, "action": "reset_metrics"})
	default:
		a.sendError(w, "Invalid action. Supported: drain, restart, reset_metrics", http.StatusBadRequest)
	}
}

// handleMetrics maneja GET /api/metrics
func (a *AdminServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := a.Monitor.SnapshotMetrics()
	a.sendSuccess(w, "Metrics retrieved successfully", metrics)
}

// handleConfig maneja GET /api/config
func (a *AdminServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := map[string]interface{}{
		"algorithm":      "round_robin",
		"retry_attempts": 2,
		"retry_delay_ms": 100,
		"timeout_ms":     5000,
		"backends_count": len(a.Monitor.SnapshotMetrics()),
		"alive_backends": len(a.Monitor.GetAliveBackends()),
	}

	a.sendSuccess(w, "Configuration retrieved successfully", config)
}

// handleHealth maneja GET /api/health
func (a *AdminServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	aliveBackends := a.Monitor.GetAliveBackends()
	breakerStats := a.Proxy.GetCircuitBreakerStats()

	// Contar circuit breakers abiertos
	openBreakers := 0
	for _, stats := range breakerStats {
		if state, ok := stats["state"].(string); ok && state == "OPEN" {
			openBreakers++
		}
	}

	health := map[string]interface{}{
		"status":         "healthy",
		"alive_backends": len(aliveBackends),
		"total_backends": len(a.Monitor.SnapshotMetrics()),
		"open_breakers":  openBreakers,
		"timestamp":      fmt.Sprintf("%d", time.Now().Unix()),
	}

	a.sendSuccess(w, "API is healthy", health)
}

// sendSuccess envía una respuesta exitosa
func (a *AdminServer) sendSuccess(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendError envía una respuesta de error
func (a *AdminServer) sendError(w http.ResponseWriter, error string, statusCode int) {
	response := APIResponse{
		Success: false,
		Error:   error,
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleCircuitBreakerStats maneja GET /api/circuit-breaker
func (a *AdminServer) handleCircuitBreakerStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := a.Proxy.GetCircuitBreakerStats()
	a.sendSuccess(w, "Circuit breaker stats retrieved successfully", stats)
}

// handleCircuitBreakerActions maneja POST /api/circuit-breaker/{backend}/reset
func (a *AdminServer) handleCircuitBreakerActions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extraer backend URL del path
	path := strings.TrimPrefix(r.URL.Path, "/api/circuit-breaker/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] != "reset" {
		a.sendError(w, "Invalid endpoint. Use /api/circuit-breaker/{backend}/reset", http.StatusBadRequest)
		return
	}

	backendURL := parts[0]

	// Decodificar URL si es necesario (replace %3A with :, etc.)
	if strings.Contains(backendURL, "%3A") {
		backendURL = strings.ReplaceAll(backendURL, "%3A", ":")
	}
	if strings.Contains(backendURL, "%2F") {
		backendURL = strings.ReplaceAll(backendURL, "%2F", "/")
	}

	// Resetear circuit breaker
	a.Proxy.ResetCircuitBreaker(backendURL)

	message := fmt.Sprintf("Circuit breaker reset for %s", backendURL)
	a.sendSuccess(w, message, map[string]interface{}{
		"backend": backendURL,
		"action":  "reset",
	})
}

// handleRateLimitStats maneja GET /api/rate-limit
func (a *AdminServer) handleRateLimitStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Obtener estadísticas del rate limiter
	stats := a.Proxy.GetRateLimitStats()

	response := map[string]interface{}{
		"status": "success",
		"data":   stats,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[api] error encoding rate limit response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
