package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal/analyze"
	"github.com/deyvigo/balanceador/balancer/internal/config"
	"github.com/deyvigo/balanceador/balancer/internal/execute"
	"github.com/deyvigo/balanceador/balancer/internal/loadbalancer"
	"github.com/deyvigo/balanceador/balancer/internal/monitor"
	"github.com/deyvigo/balanceador/balancer/internal/plan"
	"github.com/deyvigo/balanceador/balancer/internal/web"
)

func main() {
	// load config
	if err := config.LoadConfig("config.json"); err != nil {
		panic(err)
	}

	// create loggers
	monitorLogger := config.GetModuleLogger("monitor")
	analyzeLogger := config.GetModuleLogger("analyze")
	planLogger := config.GetModuleLogger("plan")
	executeLogger := config.GetModuleLogger("execute")

	backends := []string{
		"http://localhost:8080",
		"http://localhost:8081",
		"http://localhost:8082",
	}

	alpha := 0.2
	period := 5 * time.Second
	timeout := 2 * time.Second

	failureThreshold := 3
	openStateTimeout := 10 * time.Second

	mon := monitor.NewMonitor(backends, period, alpha, timeout, monitorLogger, failureThreshold, openStateTimeout)
	analyzer := analyze.NewAnalyzer(mon.GetUpdatesChannel(), analyzeLogger)
	plan := plan.NewPlan(analyzer.GetUpdatesChannel(), planLogger)
	execute := execute.NewExecute(plan.GetUpdatesChannel(), executeLogger)

	// Crear balanceador Weighted Round Robin
	wrr := loadbalancer.NewWeightedRoundRobin()

	// Crear servidor WebSocket
	wsServer := &web.WebSocketServer{
		Monitor: mon,
	}

	// Contexto para manejar shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Iniciar polling de métricas
	mon.StartPolling(ctx)
	analyzer.Start(ctx)
	plan.Start(ctx)
	execute.Start(ctx)

	// Actualizar pesos del load balancer periódicamente
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics := mon.SnapshotMetrics()
				wrr.UpdateMetrics(metrics)

				// Log de pesos calculados
				weights := wrr.GetBackendWeights()
				log.Printf("[LoadBalancer] Pesos actualizados: %+v", weights)
			}
		}
	}()

	// Handler principal: Balanceo de carga con Weighted Round Robin
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backendURL, ok := wrr.NextBackend()
		if !ok {
			http.Error(w, "No hay backends disponibles", http.StatusServiceUnavailable)
			log.Println("[LoadBalancer] ⚠️  No hay backends disponibles")
			return
		}

		target, err := url.Parse(backendURL)
		if err != nil {
			http.Error(w, "Error al parsear URL del backend", http.StatusInternalServerError)
			log.Printf("[LoadBalancer] ❌ Error parseando URL %s: %v", backendURL, err)
			return
		}

		// Crear reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[LoadBalancer] ❌ Error en proxy a %s: %v", target, err)
			http.Error(w, fmt.Sprintf("Error conectando con backend: %v", err), http.StatusBadGateway)
		}

		log.Printf("[LoadBalancer] ➡️  %s → %s", r.URL.Path, backendURL)
		proxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/metrics/ws", wsServer.MetricsHandler)

	addr := ":9000"
	srv := &http.Server{Addr: addr, Handler: nil}

	go func() {
		log.Printf("Server runing in %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdowning server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}

	log.Println("Server off")
}
