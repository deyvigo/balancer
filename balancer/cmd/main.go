package main

import (
	"context"
	"log"
	"net/http"
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

	limiterConf := config.GetRateLimiterConfig()

	// create loggers
	monitorLogger := config.GetModuleLogger("monitor")
	analyzeLogger := config.GetModuleLogger("analyze")
	planLogger := config.GetModuleLogger("plan")
	executeLogger := config.GetModuleLogger("execute")

	backends := []string{
		"http://127.0.0.1:8080",
		"http://127.0.0.1:8081",
		"http://127.0.0.1:8082",
	}

	alpha := 0.2
	period := 1 * time.Second
	timeout := 2 * time.Second

	// aaaaaaa
	failureThreshold := 3
	openStateTimeout := 10 * time.Second
	// Crear load balancer para redirigir el tráfico hacia las réplicas
	lb := loadbalancer.NewLoadBalancer(backends)
	lb.UpdateRateLimit(limiterConf.NormalRate, limiterConf.NormalBurst)

	mon := monitor.NewMonitor(backends, period, alpha, timeout, monitorLogger, failureThreshold, openStateTimeout, lb)
	analyzer := analyze.NewAnalyzer(mon.GetUpdatesChannel(), analyzeLogger)
	plan := plan.NewPlan(analyzer.GetUpdatesChannel(), planLogger)
	execute := execute.NewExecute(plan.GetUpdatesChannel(), executeLogger, lb)

	// Crear servidor WebSocket
	wsServer := &web.WebSocketServer{
		Monitor: mon,
	}

	// Crear mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics/ws", wsServer.MetricsHandler)
	mux.Handle("/", lb)

	// Contexto para manejar shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Iniciar polling de métricas
	mon.StartPolling(ctx)
	analyzer.Start(ctx)
	plan.Start(ctx)
	execute.Start(ctx)

	addr := ":9000"
	srv := &http.Server{Addr: addr, Handler: mux}

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
