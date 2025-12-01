package plan

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type Plan struct {
	inputChannel  <-chan []internal.AnalysisResult
	outputChannel chan []internal.PlanResult
	logger        *slog.Logger
	lastStatuses  map[int]string
}

func NewPlan(inputChannel <-chan []internal.AnalysisResult, logger *slog.Logger) *Plan {
	return &Plan{
		inputChannel:  inputChannel,
		outputChannel: make(chan []internal.PlanResult, 10),
		logger:        logger,
		lastStatuses:  make(map[int]string),
	}
}

func (p *Plan) GetUpdatesChannel() <-chan []internal.PlanResult {
	return p.outputChannel
}

func (p *Plan) Start(ctx context.Context) {
	go func() {
		p.logger.Info("Plan started")
		for {
			select {
			case <-ctx.Done():
				close(p.outputChannel)
				return
			case analysis, ok := <-p.inputChannel:
				if !ok {
					return
				}
				p.planBatch(analysis)
			}
		}
	}()
}

func (p *Plan) planBatch(analysis []internal.AnalysisResult) {
	batchPlan := make([]internal.PlanResult, 0, len(analysis))
	for _, item := range analysis {

		lastStatus, known := p.lastStatuses[item.BackendId]

		if known && lastStatus == item.Status {
			// No changes
			continue
		}
		p.lastStatuses[item.BackendId] = item.Status

		var action string
		switch item.Status {
		case "DOWN":
			action = "ATTEMPT_RESTART"
			p.logger.Info(fmt.Sprintf("DECISIÓN: Prender/Reiniciar Backend %d (Causa: %s)", item.BackendId, item.Reason))
		case "DEGRADED":
			action = "THROTTLE_TRAFFIC"
			p.logger.Info(fmt.Sprintf("DECISIÓN: Limitar tráfico al Backend %d (Causa: %s)", item.BackendId, item.Reason))
		case "HEALTHY":
			action = "ENSURE_ACTIVE"
		default:
			action = "NO_OP"
			p.logger.Info(fmt.Sprintf("Estado desconocido para Backend %d", item.BackendId))
			log.Printf("[PLAN] Estado desconocido para Backend %d", item.BackendId)
		}

		batchPlan = append(batchPlan, internal.PlanResult{
			BackendId: item.BackendId,
			Action:    action,
		})
	}

	if len(batchPlan) > 0 {
		select {
		case p.outputChannel <- batchPlan:
		default:
			p.logger.Warn("Warning: Output channel full, dropping plan result")
			// log.Printf("[Plan] Warning: Output channel full, dropping plan result")
		}
	}
}
