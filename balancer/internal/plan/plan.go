package plan

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type Plan struct {
	inputChannel  <-chan map[string]internal.AnalysisResult
	outputChannel chan map[string]internal.PlanResult
	logger        *slog.Logger
	lastStatuses  map[string]string
}

func NewPlan(inputChannel <-chan map[string]internal.AnalysisResult, logger *slog.Logger) *Plan {
	return &Plan{
		inputChannel:  inputChannel,
		outputChannel: make(chan map[string]internal.PlanResult, 10),
		logger:        logger,
		lastStatuses:  make(map[string]string),
	}
}

func (p *Plan) GetUpdatesChannel() <-chan map[string]internal.PlanResult {
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
				p.planSnapshot(analysis)
			}
		}
	}()
}

func (p *Plan) planSnapshot(analysis map[string]internal.AnalysisResult) {
	batchPlan := make(map[string]internal.PlanResult, len(analysis))
	for url, item := range analysis {

		lastStatus, known := p.lastStatuses[url]

		if known && lastStatus == item.Status {
			// No changes
			continue
		}
		p.lastStatuses[url] = item.Status

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

		batchPlan[url] = internal.PlanResult{
			BackendId: item.BackendId,
			Action:    action,
		}
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
