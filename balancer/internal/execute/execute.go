package execute

import (
	"context"
	"log"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type Execute struct {
	inputChannel <-chan []internal.PlanResult
}

func NewExecute(inputChannel <-chan []internal.PlanResult) *Execute {
	return &Execute{
		inputChannel: inputChannel,
	}
}

func (e *Execute) Start(ctx context.Context) {
	go func() {
		log.Println("[Execute] Execute started")
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
	log.Printf("[Execute] Executing plan: %v", plan)
}
