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

	mon := monitor.NewMonitor(backends, period, alpha, timeout, monitorLogger)
	analyzer := analyze.NewAnalyzer(mon.GetUpdatesChannel(), analyzeLogger)
	plan := plan.NewPlan(analyzer.GetUpdatesChannel(), planLogger)
	execute := execute.NewExecute(plan.GetUpdatesChannel(), executeLogger)

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
