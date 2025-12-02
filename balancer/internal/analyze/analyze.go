package analyze

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deyvigo/balanceador/balancer/internal"
	"github.com/deyvigo/balanceador/balancer/internal/config"
)

// mover a config.json
const (
	HighTrafficThreshold = 30.0 // RPS (Peticiones por segundo)
	AttackThreshold      = 80.0 // RPS considerado ataque
)

type Analyzer struct {
	inputChannel  <-chan internal.SystemStatus
	outputChannel chan []internal.AnalysisResult
	logger        *slog.Logger
}

func NewAnalyzer(inputChannel <-chan internal.SystemStatus, logger *slog.Logger) *Analyzer {
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

func (a *Analyzer) analyzeBatch(systemStatus internal.SystemStatus) {
	results := make([]internal.AnalysisResult, 0, len(systemStatus.Backends)+1)
	for _, m := range systemStatus.Backends {
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
				a.logger.Info(fmt.Sprintf("Backend %d is down (ErrorRate: %.2f, EMAMs: %.1f)", m.Id, m.ErrorRate, m.EMAMs))
			} else if m.ErrorRate > 0.5 {
				status = "DEGRADED"
				reason = "Error rate is high (>50%)"
				a.logger.Info(fmt.Sprintf("Backend %d is degraded (ErrorRate: %.2f, EMAMs: %.1f)", m.Id, m.ErrorRate, m.EMAMs))
			} else {
				status = "HEALTHY"
				reason = "Everything is ok"
				// a.logger.Info(fmt.Sprintf("Backend %d is healthy", m.Id))
			}
		}

		results = append(results, internal.AnalysisResult{
			BackendId:    m.Id,
			Status:       status,
			Reason:       reason,
			CircuitState: m.CircuitState,
		})

	}

	lbAnalysis := internal.AnalysisResult{
		BackendId: -1,
		Status:    "NORMAL",
		Reason:    "Traffic normal",
	}

	lb := systemStatus.LB
	conf := config.GetAnalyzerConfig()

	if lb.RPS > conf.AttackThreshold {
		lbAnalysis.Status = "ATTACK"
		lbAnalysis.Reason = fmt.Sprintf("Critical RPS detected: %.2f", lb.RPS)
		a.logger.Error("Analyzer detected ATTACK conditions!", "rps", lb.RPS)
	} else if lb.RPS > conf.HighTrafficThreshold {
		lbAnalysis.Status = "HIGH_LOAD"
		lbAnalysis.Reason = fmt.Sprintf("High traffic: %.2f RPS", lb.RPS)
		a.logger.Info("Analyzer detected High Load", "rps", lb.RPS)
	} else if lb.BlockedReqs > 0 {
		// Si el tráfico es bajo pero hay bloqueos, el rate limiter está trabajando (quizás mitigando un ataque lento)
		lbAnalysis.Status = "MITIGATING"
		lbAnalysis.Reason = fmt.Sprintf("Blocking requests: %d dropped", lb.BlockedReqs)
	}

	results = append(results, lbAnalysis)

	if len(results) > 0 {
		select {
		case a.outputChannel <- results:
		default:
			a.logger.Warn("Warning: Output channel full, dropping analysis result")
		}
	}
}
