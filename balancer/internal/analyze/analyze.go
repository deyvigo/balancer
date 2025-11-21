package analyze

import (
	"context"
	"log"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type Analyzer struct {
	inputChannel <-chan []internal.Metrics
	// Here we can add other channel to comunicate with Plan module
	outputChannel chan []internal.AnalysisResult
}

func NewAnalyzer(inputChannel <-chan []internal.Metrics) *Analyzer {
	return &Analyzer{
		inputChannel:  inputChannel,
		outputChannel: make(chan []internal.AnalysisResult, 10),
	}
}

func (a *Analyzer) GetUpdatesChannel() <-chan []internal.AnalysisResult {
	return a.outputChannel
}

func (a *Analyzer) Start(ctx context.Context) {
	go func() {
		log.Println("[Analize] Analyzer started")
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

		if !m.Alive {
			status = "DOWN"
			reason = "Connection refused / timeout"
			log.Printf("[Analize] Backend %d is down", m.Id)
		} else if m.ErrorRate > 0.5 {
			status = "DEGRADED"
			reason = "Error rate is high (>50%)"
			log.Printf("[Analize] Backend %d is degraded", m.Id)
		} else {
			status = "HEALTHY"
			reason = "Everything is ok"
			log.Printf("[Analize] Backend %d is healthy", m.Id)
		}
		results = append(results, internal.AnalysisResult{
			BackendId: m.Id,
			Status:    status,
			Reason:    reason,
		})

	}
	if len(results) > 0 {
		select {
		case a.outputChannel <- results:
		default:
			log.Printf("[Analize] Warning: Output channel full, dropping analysis result")
		}
	}
}
