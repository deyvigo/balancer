package execute

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type Execute struct {
	inputChannel <-chan []internal.PlanResult
	logger       *slog.Logger
}

func NewExecute(inputChannel <-chan []internal.PlanResult, logger *slog.Logger) *Execute {
	return &Execute{
		inputChannel: inputChannel,
		logger:       logger,
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
	// log.Printf("[Execute] Executing plan: %v", plan)
}
