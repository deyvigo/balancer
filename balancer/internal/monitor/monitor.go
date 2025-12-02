package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal"
	"github.com/deyvigo/balanceador/balancer/internal/loadbalancer"
)

type SafeBackend struct {
	data internal.Backend
	mu   sync.RWMutex
}

func (b *SafeBackend) snapshot() (alive bool, ema float64, errRate float64, last time.Time, rawURL string, state internal.CircuitState) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data.Alive, b.data.EMAms, b.data.ErrorRate, b.data.CheckedAt, b.data.URL.String(), b.data.CircuitState
}

type MonitorService struct {
	backends         []*SafeBackend
	client           *http.Client
	alpha            float64
	period           time.Duration
	mu               sync.RWMutex
	updatesChannel   chan internal.SystemStatus
	logger           *slog.Logger
	failureThreshold int
	openStateTimeout time.Duration
	loadbalancer     *loadbalancer.LoadBalancer
}

func NewMonitor(
	backends []string,
	period time.Duration,
	alpha float64,
	timeout time.Duration,
	logger *slog.Logger,
	failureThreshold int,
	openStateTimeout time.Duration,
	lb *loadbalancer.LoadBalancer,
) *MonitorService {
	bs := make([]*SafeBackend, 0, len(backends))

	for _, s := range backends {
		if strings.TrimSpace(s) == "" {
			continue
		}
		u, err := url.Parse(s)
		if err != nil {
			logger.Info(fmt.Sprintf("ignoring invalid backend url %q: %v", s, err))
			continue
		}
		bs = append(bs, &SafeBackend{
			data: internal.Backend{
				URL:             u,
				Alive:           false,
				CircuitState:    internal.StateClosed,
				LastStateChange: time.Now(),
			},
		})
	}

	// Compartir el mismo transport para el health check
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 3,
		IdleConnTimeout:     10 * time.Second,
		DisableKeepAlives:   false,
	}

	return &MonitorService{
		backends: bs,
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		alpha:            alpha,
		period:           period,
		updatesChannel:   make(chan internal.SystemStatus, 10),
		logger:           logger,
		failureThreshold: failureThreshold,
		openStateTimeout: openStateTimeout,
		loadbalancer:     lb,
	}
}

func (m *MonitorService) GetUpdatesChannel() <-chan internal.SystemStatus {
	return m.updatesChannel
}

func (m *MonitorService) checkAndNotify() {
	m.checkAll()

	fullStatus := m.SnapshotMetrics()

	// Actualizar metricas del weighted round robin
	m.loadbalancer.UpdateMetrics(&fullStatus.Backends)
	select {
	case m.updatesChannel <- fullStatus:
	default:
		m.logger.Warn("Warning: Updates channel full, dropping metric snapshot")
	}
}

func (m *MonitorService) StartPolling(ctx context.Context) {
	t := time.NewTicker(m.period)
	go func() {
		m.logger.Info("Monitor started")
		defer t.Stop()

		m.checkAndNotify()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.checkAndNotify()
			}
		}
	}()
}

// performHealthCheck conducts a health check on a backend and updates its metrics.
// It returns true if an error occurred.
// This function assumes the lock on the SafeBackend is already held.
func (m *MonitorService) performHealthCheck(b *SafeBackend) (isErr bool) {
	u := *b.data.URL
	if u.Path == "" || u.Path == "/" {
		u.Path = "/health"
	}

	start := time.Now()
	resp, err := m.client.Get(u.String())
	latMs := float64(time.Since(start).Milliseconds())

	if err != nil {
		isErr = true
		m.logger.Error(fmt.Sprintf("%s health check failed: %v (latency: %.1fms)", b.data.URL.String(), err, latMs))
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			isErr = true
			m.logger.Error(fmt.Sprintf("%s returned status %d (latency: %.1fms)", b.data.URL.String(), resp.StatusCode, latMs))
		} else {
			// Solo loguear si la latencia es alta
			if latMs > 1000 {
				m.logger.Info(fmt.Sprintf("%s health check OK but slow: %.1fms", b.data.URL.String(), latMs))
			}
		}
	}

	// Update metrics
	b.data.Alive = !isErr
	b.data.CheckedAt = time.Now()

	// Update EMA latency
	if b.data.EMAms == 0 {
		b.data.EMAms = latMs
	} else {
		b.data.EMAms = m.alpha*latMs + (1-m.alpha)*b.data.EMAms
	}

	// Update EMA error rate
	var e float64 = 0.0
	if isErr {
		e = 1.0
	}
	if b.data.ErrorRate == 0 {
		b.data.ErrorRate = e
	} else {
		b.data.ErrorRate = m.alpha*e + (1-m.alpha)*b.data.ErrorRate
	}

	return isErr
}

func (m *MonitorService) checkBackend(b *SafeBackend) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.data.CircuitState {
	case internal.StateClosed:
		if m.performHealthCheck(b) { // Health check failed
			b.data.Failures++
			if b.data.Failures >= m.failureThreshold {
				b.data.CircuitState = internal.StateOpen
				b.data.LastStateChange = time.Now()
				m.logger.Info(fmt.Sprintf("Backend %s circuit state changed to %s", b.data.URL, internal.StateOpen))
			}
		} else { // Health check succeeded
			b.data.Failures = 0
		}

	case internal.StateOpen:
		if time.Since(b.data.LastStateChange) > m.openStateTimeout {
			b.data.CircuitState = internal.StateHalfOpen
			b.data.LastStateChange = time.Now()
			m.logger.Info(fmt.Sprintf("Backend %s circuit state changed to %s", b.data.URL, internal.StateHalfOpen))
		}

	case internal.StateHalfOpen:
		if m.performHealthCheck(b) { // Probe failed
			b.data.CircuitState = internal.StateOpen
			b.data.LastStateChange = time.Now()
			m.logger.Info(fmt.Sprintf("Backend %s circuit state changed to %s (from half-open)", b.data.URL, internal.StateOpen))
		} else { // Probe succeeded
			b.data.Failures = 0
			b.data.CircuitState = internal.StateClosed
			b.data.LastStateChange = time.Now()
			m.logger.Info(fmt.Sprintf("Backend %s circuit state changed to %s", b.data.URL, internal.StateClosed))
		}
	}
}

func (m *MonitorService) checkAll() {
	m.mu.Lock()
	backends := make([]*SafeBackend, len(m.backends))
	copy(backends, m.backends)
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, b := range backends {
		wg.Add(1)
		go func(bb *SafeBackend) {
			defer wg.Done()
			m.checkBackend(bb)
		}(b)
	}
	wg.Wait()
}

func (m *MonitorService) SnapshotMetrics() internal.SystemStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	backendMetrics := make([]internal.Metrics, 0, len(m.backends))
	for i, b := range m.backends {
		alive, ema, er, last, u, state := b.snapshot()
		backendMetrics = append(backendMetrics, internal.Metrics{
			Id:           i,
			URL:          u,
			Alive:        alive,
			EMAMs:        ema,
			ErrorRate:    er,
			LastChecked:  last.Format(time.RFC3339),
			CircuitState: state,
		})
	}

	incoming, dropped := m.loadbalancer.CollectStats()
	rps := float64(incoming) / m.period.Seconds()

	lbStats := internal.BalancerStats{
		TotalReqs:   incoming,
		BlockedReqs: dropped,
		RPS:         rps,
	}

	return internal.SystemStatus{
		Backends: backendMetrics,
		LB:       lbStats,
	}
}
