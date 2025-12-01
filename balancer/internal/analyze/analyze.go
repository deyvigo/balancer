package analyze

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type Analyzer struct {
	inputChannel  <-chan []internal.Metrics
	outputChannel chan []internal.AnalysisResult
	logger        *slog.Logger
}

func NewAnalyzer(inputChannel <-chan []internal.Metrics, logger *slog.Logger) *Analyzer {
	return &Analyzer{
		inputChannel:  inputChannel,
		outputChannel: make(chan []internal.AnalysisResult, 10),
		logger:        logger,
	}
}

func (a *Analyzer) GetUpdatesChannel() <-chan []internal.AnalysisResult {
	return a.outputChannel
}

func (a *Analyzer) Start(ctx context.Context) {
	go func() {
		a.logger.Info("Analyzer started")
		for {
			select {
			case <-ctx.Done():
				close(a.outputChannel)
				return
			case metrics, ok := <-a.inputChannel:
				if !ok {
					return
				}
				a.analyzeBatch(metrics)
			}
		}
	}()
}

func (a *Analyzer) analyzeBatch(metrics []internal.Metrics) {
	results := make([]internal.AnalysisResult, 0, len(metrics))
	for _, m := range metrics {
		var status, reason string

		switch m.CircuitState {
		case internal.StateOpen:
			status = "DOWN"
			reason = "Circuit is open"
			a.logger.Info(fmt.Sprintf("Backend %d is down (circuit open)", m.Id))
		case internal.StateHalfOpen:
			status = "DEGRADED"
			reason = "Circuit is half-open, testing connection"
			a.logger.Info(fmt.Sprintf("Backend %d is degraded (circuit half-open)", m.Id))
		case internal.StateClosed:
			if !m.Alive {
				status = "DOWN"
				reason = "Connection refused / timeout"
				a.logger.Info(fmt.Sprintf("Backend %d is down", m.Id))
			} else if m.ErrorRate > 0.5 {
				status = "DEGRADED"
				reason = "Error rate is high (>50%)"
				a.logger.Info(fmt.Sprintf("Backend %d is degraded", m.Id))
			} else {
				status = "HEALTHY"
				reason = "Everything is ok"
				a.logger.Info(fmt.Sprintf("Backend %d is healthy", m.Id))
			}
		}

		results = append(results, internal.AnalysisResult{
			BackendId:    m.Id,
			Status:       status,
			Reason:       reason,
			CircuitState: m.CircuitState,
		})

	}
	if len(results) > 0 {
		select {
		case a.outputChannel <- results:
		default:
			a.logger.Warn("Warning: Output channel full, dropping analysis result")
		}
	}
}
