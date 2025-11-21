package plan

import (
	"context"
	"log"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type Plan struct {
	inputChannel  <-chan []internal.AnalysisResult
	outputChannel chan []internal.PlanResult
}

func NewPlan(inputChannel <-chan []internal.AnalysisResult) *Plan {
	return &Plan{
		inputChannel:  inputChannel,
		outputChannel: make(chan []internal.PlanResult, 10),
	}
}

func (p *Plan) GetUpdatesChannel() <-chan []internal.PlanResult {
	return p.outputChannel
}

func (p *Plan) Start(ctx context.Context) {
	go func() {
		log.Println("[Plan] Plan started")
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
		var action string

		switch item.Status {
		case "DOWN":
			action = "ATTEMPT_RESTART"
			log.Printf("[PLAN] DECISIÓN: Prender/Reiniciar Backend %d (Causa: %s)", item.BackendId, item.Reason)
		case "DEGRADED":
			action = "THROTTLE_TRAFFIC"
			log.Printf("[PLAN] DECISIÓN: Limitar tráfico al Backend %d (Causa: %s)", item.BackendId, item.Reason)
		case "HEALTHY":
			action = "ENSURE_ACTIVE"
		default:
			action = "NO_OP"
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
			log.Printf("[Plan] Warning: Output channel full, dropping plan result")
		}
	}
}
