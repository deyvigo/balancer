package monitor

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal"
)

type SafeBackend struct {
	data internal.Backend
	mu   sync.RWMutex
}

func (b *SafeBackend) update(latMs float64, isErr bool, alpha float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.data.EMAms == 0 {
		b.data.EMAms = latMs
	} else {
		b.data.EMAms = alpha*latMs + (1-alpha)*b.data.EMAms
	}

	var e float64
	if isErr {
		e = 1.0
	} else {
		e = 0.0
	}

	if b.data.ErrorRate == 0 {
		b.data.ErrorRate = e
	} else {
		b.data.ErrorRate = alpha*e + (1-alpha)*b.data.ErrorRate
	}

	b.data.CheckedAt = time.Now()
}

func (b *SafeBackend) setAlive(a bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data.Alive = a
	b.data.CheckedAt = time.Now()
}

func (b *SafeBackend) snapshot() (alive bool, ema float64, errRate float64, last time.Time, rawURL string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data.Alive, b.data.EMAms, b.data.ErrorRate, b.data.CheckedAt, b.data.URL.String()
}

type MonitorService struct {
	backends []*SafeBackend
	client   *http.Client
	alpha    float64
	period   time.Duration
	mu       sync.RWMutex
}

func NewMonitor(backends []string, period time.Duration, alpha float64, timeout time.Duration) *MonitorService {
	bs := make([]*SafeBackend, 0, len(backends))

	for _, s := range backends {
		if strings.TrimSpace(s) == "" {
			continue
		}
		u, err := url.Parse(s)
		if err != nil {
			log.Printf("ignoring invalid backend url %q: %v", s, err)
			continue
		}
		bs = append(bs, &SafeBackend{
			data: internal.Backend{
				URL:   u,
				Alive: false,
			},
		})
	}

	return &MonitorService{
		backends: bs,
		client: &http.Client{
			Timeout: timeout,
		},
		alpha:  alpha,
		period: period,
	}
}

func (m *MonitorService) StartPolling(ctx context.Context) {
	t := time.NewTicker(m.period)
	go func() {
		defer t.Stop()

		m.checkAll()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.checkAll()
			}
		}
	}()
}

func (m *MonitorService) checkBackend(b *SafeBackend) {
	u := *b.data.URL

	if u.Path == "" || u.Path == "/" {
		u.Path = "/health"
	}

	start := time.Now()
	resp, err := m.client.Get(u.String())
	latMs := float64(time.Since(start).Milliseconds())
	isErr := false
	if err != nil {
		isErr = true
		b.setAlive(false)
		log.Printf("[monitor] %s error: %v", b.data.URL.String(), err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			isErr = true
			b.setAlive(false)
			log.Printf("[monitor] %s returned status %d", b.data.URL.String(), resp.StatusCode)
		} else {
			// healthy
			b.setAlive(true)
		}
	}

	b.update(latMs, isErr, m.alpha)
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

func (m *MonitorService) SnapshotMetrics() []internal.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]internal.Metrics, 0, len(m.backends))
	for i, b := range m.backends {
		alive, ema, er, last, u := b.snapshot()
		res = append(res, internal.Metrics{
			Id:          i,
			URL:         u,
			Alive:       alive,
			EMAMs:       ema,
			ErrorRate:   er,
			LastChecked: last.Format(time.RFC3339),
		})
	}
	return res
}

// GetAliveBackends devuelve una lista de URLs de backends que están vivos
func (m *MonitorService) GetAliveBackends() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var alive []string
	for _, b := range m.backends {
		isAlive, _, _, _, url := b.snapshot()
		if isAlive {
			alive = append(alive, url)
		}
	}
	return alive
}

// GetBackendMetrics devuelve las métricas de un backend específico por URL
func (m *MonitorService) GetBackendMetrics(targetURL string) (alive bool, emaMs float64, errorRate float64, found bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, b := range m.backends {
		isAlive, ema, er, _, url := b.snapshot()
		if url == targetURL {
			return isAlive, ema, er, true
		}
	}
	return false, 0, 0, false
}
