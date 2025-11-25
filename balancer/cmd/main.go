package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal/api"
	"github.com/deyvigo/balanceador/balancer/internal/breaker"
	"github.com/deyvigo/balanceador/balancer/internal/config"
	"github.com/deyvigo/balanceador/balancer/internal/monitor"
	"github.com/deyvigo/balanceador/balancer/internal/optimizer"
	"github.com/deyvigo/balanceador/balancer/internal/proxy"
	"github.com/deyvigo/balanceador/balancer/internal/web"
)

func main() {
	// Cargar configuración desde archivo
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	log.Printf("Loaded configuration with %d backends", len(cfg.Backends))

	// Crear monitor con configuración
	backends := cfg.GetEnabledBackends()
	mon := monitor.NewMonitor(backends, cfg.Monitor.GetPeriod(), cfg.Monitor.Alpha, cfg.Monitor.GetTimeout())

	// Determinar algoritmo desde configuración
	var algorithm proxy.LoadBalancingAlgorithm
	switch cfg.Proxy.Algorithm {
	case "weighted_round_robin":
		algorithm = proxy.WeightedRoundRobin
	case "least_connections":
		algorithm = proxy.LeastConnections
	default:
		algorithm = proxy.RoundRobin
	}

	// Configuración del proxy desde archivo
	proxyConfig := proxy.ProxyConfig{
		Algorithm:     algorithm,
		RetryAttempts: cfg.Proxy.RetryAttempts,
		RetryDelay:    cfg.Proxy.RetryDelay,
		Timeout:       cfg.Proxy.Timeout,
	}

	// Configuración del circuit breaker
	breakerConfig := breaker.CircuitBreakerConfig{
		FailureThreshold:   cfg.CircuitBreaker.FailureThreshold,
		ErrorRateThreshold: cfg.CircuitBreaker.ErrorRateThreshold,
		OpenTimeout:        cfg.CircuitBreaker.OpenTimeout,
		HalfOpenMaxCalls:   cfg.CircuitBreaker.HalfOpenMaxCalls,
		MinRequestCount:    cfg.CircuitBreaker.MinRequestCount,
	}

	// Configuración del optimizador de pesos
	optimizerConfig := optimizer.WeightCalculationConfig{
		Enabled:               cfg.WeightOptimization.Enabled,
		MinWeight:             cfg.WeightOptimization.MinWeight,
		MaxWeight:             cfg.WeightOptimization.MaxWeight,
		LatencyWeight:         cfg.WeightOptimization.LatencyWeight,
		ErrorRateWeight:       cfg.WeightOptimization.ErrorRateWeight,
		AdaptationSpeed:       cfg.WeightOptimization.AdaptationSpeed,
		LatencyTargetMs:       cfg.WeightOptimization.LatencyTargetMs,
		MaxErrorRate:          cfg.WeightOptimization.MaxErrorRate,
		UpdateIntervalSeconds: cfg.WeightOptimization.UpdateIntervalSeconds,
	}

	// Crear load balancer con circuit breaker, optimizador y rate limiting
	lb := proxy.NewLoadBalancer(mon, proxyConfig, breakerConfig, optimizerConfig, cfg.RateLimit)

	// Iniciar optimizador de pesos adaptativos
	lb.GetOptimizer().Start()

	// Crear servidor WebSocket para métricas
	wsServer := &web.WebSocketServer{
		Monitor: mon,
	}

	// Crear servidor de administración
	adminServer := api.NewAdminServer(mon, lb)

	// Contexto para manejar shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Iniciar polling de métricas
	mon.StartPolling(ctx)

	// Servidor de métricas WebSocket y Admin API (puerto configurable)
	metricsMux := http.NewServeMux()
	metricsMux.HandleFunc("/metrics/ws", wsServer.MetricsHandler)
	adminServer.RegisterHandlers(metricsMux)
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Web.MetricsPort),
		Handler: metricsMux,
	}

	// Servidor de proxy HTTP (puerto configurable)
	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/", lb.Handler)
	proxyServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Proxy.Port),
		Handler: proxyMux,
	}

	// Grupo de goroutines para servidores
	var wg sync.WaitGroup

	// Iniciar servidor de métricas
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Metrics server (WebSocket + Admin API) running on :%d", cfg.Web.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Iniciar servidor de proxy
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Proxy server running on :%d", cfg.Proxy.Port)
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Proxy server error: %v", err)
		}
	}()

	// Esperar señal de shutdown
	<-ctx.Done()
	log.Println("Shutting down servers...")

	// Shutdown con timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown servidores
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Metrics server shutdown error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := proxyServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Proxy server shutdown error: %v", err)
		}
	}()

	wg.Wait()
	log.Println("Servers stopped")
}
