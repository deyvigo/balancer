package execute

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deyvigo/balanceador/balancer/internal"
	"github.com/deyvigo/balanceador/balancer/internal/config"
	"github.com/deyvigo/balanceador/balancer/internal/loadbalancer"
)

type Execute struct {
	inputChannel <-chan []internal.PlanResult
	logger       *slog.Logger
	lb           *loadbalancer.LoadBalancer
}

func NewExecute(
	inputChannel <-chan []internal.PlanResult,
	logger *slog.Logger,
	lb *loadbalancer.LoadBalancer,
) *Execute {
	return &Execute{
		inputChannel: inputChannel,
		logger:       logger,
		lb:           lb,
	}
}

func (e *Execute) Start(ctx context.Context) {
	go func() {
		e.logger.Info("Execute started")
		// log.Println("[Execute] Execute started")
		for {
			select {
			case <-ctx.Done():
				return
			case plan, ok := <-e.inputChannel:
				if !ok {
					return
				}
				e.executeBatch(plan)
			}
		}
	}()
}

func (e *Execute) executeBatch(plan []internal.PlanResult) {
	e.logger.Info(fmt.Sprintf("Executing plan: %v", plan))
	e.logger.Info(fmt.Sprintf("Executing plan batch size: %d", len(plan)))
	limitConfig := config.GetRateLimiterConfig()
	for _, item := range plan {
		if item.BackendId == -1 {
			switch item.Action {
			case "MITIGATE_ATTACK":
				e.logger.Warn("EXECUTE: Activando Protocolo de Defensa (Low Rate Limit)")
				// Llamamos al método que creaste en el LoadBalancer
				e.lb.UpdateRateLimit(limitConfig.AttackRate, limitConfig.AttackBurst)
			case "NORMAL_OPERATION":
				e.logger.Info("EXECUTE: Restaurando Operación Normal (High Rate Limit)")
				e.lb.UpdateRateLimit(limitConfig.NormalRate, limitConfig.NormalBurst)
			default:
				e.logger.Warn("Acción global desconocida", "action", item.Action)
			}
		} else {
			switch item.Action {
			case "ATTEMPT_RESTART":
				// e.logger.Info(fmt.Sprintf("EXECUTE: [Simulación] Reiniciando contenedor Backend %d...", item.BackendId))
				// dockerClient.ContainerRestart(...)
			case "THROTTLE_TRAFFIC":
				// e.logger.Info(fmt.Sprintf("EXECUTE: [Simulación] Sacando temporalmente Backend %d del pool...", item.BackendId))
				// e.lb.RemoveBackend(item.BackendId)
			case "ENSURE_ACTIVE":
				// No hacer nada, solo verificar
			}
		}
	}
}
