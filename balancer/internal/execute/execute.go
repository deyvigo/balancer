package execute

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deyvigo/balanceador/balancer/internal"
	"github.com/deyvigo/balanceador/balancer/internal/loadbalancer"
	"github.com/deyvigo/balanceador/balancer/internal/monitor"
)

type Execute struct {
	inputChannel <-chan map[string]internal.PlanResult
	logger       *slog.Logger
	monitor      *monitor.MonitorService
	lb           *loadbalancer.WeightedRoundRobin
}

func NewExecute(inputChannel <-chan map[string]internal.PlanResult, logger *slog.Logger, mon *monitor.MonitorService, lb *loadbalancer.WeightedRoundRobin) *Execute {
	return &Execute{
		inputChannel: inputChannel,
		logger:       logger,
		monitor:      mon,
		lb:           lb,
	}
}

func (e *Execute) Start(ctx context.Context) {
	go func() {
		e.logger.Info("Execute started")
		for {
			select {
			case <-ctx.Done():
				return
			case plan, ok := <-e.inputChannel:
				if !ok {
					return
				}
				e.executePlan(plan)
			}
		}
	}()
}

func (e *Execute) executePlan(plan map[string]internal.PlanResult) {
	e.logger.Info(fmt.Sprintf("MAPE-K cycle triggered update. Plan received: %v", plan))

	metricsMap := e.monitor.SnapshotMetrics()
	metricsSlice := make([]internal.Metrics, 0, len(metricsMap))
	for _, m := range metricsMap {
		metricsSlice = append(metricsSlice, m)
	}
	e.lb.UpdateMetrics(metricsSlice)

	weights := e.lb.GetBackendWeights()
	e.logger.Info(fmt.Sprintf("[LoadBalancer] Weights updated by Execute: %+v", weights))
}
