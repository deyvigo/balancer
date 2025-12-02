package loadbalancer

import (
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal"
	"github.com/deyvigo/balanceador/balancer/internal/config"
)

type LoadBalancer struct {
	backends     []*httputil.ReverseProxy
	targets      []string
	current      uint64
	incomingReqs uint64
	droppedReqs  uint64
	limiter      *TokenBucket
	wrr          *WeightedRoundRobin
}

func NewLoadBalancer(targets []string) *LoadBalancer {
	var proxies []*httputil.ReverseProxy

	transport := &http.Transport{
		MaxIdleConns:        100, // Reducir para no consumir tantos recursos
		MaxIdleConnsPerHost: 10,  // Menos conexiones por host
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false, // Mantener keep-alives para reutilizar
	}

	for _, target := range targets {
		url, _ := url.Parse(target)
		proxy := httputil.NewSingleHostReverseProxy(url)

		proxy.Transport = transport

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			// Si falla el backend, devolvemos 502 pero sin saturar la consola
			// fmt.Printf("DEBUG: Error proxying to %s: %v\n", u, err)
			w.WriteHeader(http.StatusBadGateway)
		}

		proxies = append(proxies, proxy)
	}

	limiterConfig := config.GetRateLimiterConfig()

	return &LoadBalancer{
		backends: proxies,
		targets:  targets,
		limiter:  NewTokenBucket(limiterConfig.NormalRate, limiterConfig.NormalBurst),
		wrr:      NewWeightedRoundRobin(),
	}
}

// usar WeightedRoundRobin para seleccionar el siguiente backend
func (lb *LoadBalancer) NextProxy() *httputil.ReverseProxy {
	if backendURL, ok := lb.wrr.NextBackend(); ok {
		for i, target := range lb.targets {
			if target == backendURL {
				return lb.backends[i]
			}
		}
	}

	// Fallback a round robin simple si WRR no tiene backends disponibles
	next := atomic.AddUint64(&lb.current, 1)
	index := next % uint64(len(lb.backends))
	return lb.backends[index]
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// clientIp := GetClientIP(r)
	atomic.AddUint64(&lb.incomingReqs, 1)

	if !lb.limiter.Allow() {
		atomic.AddUint64(&lb.droppedReqs, 1)
		// fmt.Printf("⛔ Bloqueando IP: %s (Rate Limit Excedido) - Enviando 429\n", clientIp)
		w.Header().Set("Retry-After", "1")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusTooManyRequests) // 429
		w.Write([]byte("Rate limit exceeded"))
		return
	}

	proxy := lb.NextProxy()
	proxy.ServeHTTP(w, r)
}

// método para el monitor
func (lb *LoadBalancer) CollectStats() (incoming, dropped uint64) {
	incoming = atomic.SwapUint64(&lb.incomingReqs, 0)
	dropped = atomic.SwapUint64(&lb.droppedReqs, 0)
	return incoming, dropped
}

// metodo para execute
func (lb *LoadBalancer) UpdateRateLimit(rate, burst float64) {
	lb.limiter.SetParams(rate, burst)
}

// metodo para el monitor (actualizar metricas del weighted round robin)
func (lb *LoadBalancer) UpdateMetrics(metrics *[]internal.Metrics) {
	lb.wrr.UpdateMetrics(*metrics)
}

// implementacion tocken bucket (podria no ir aquí para modular mejor)
type TokenBucket struct {
	rate       float64
	capacity   float64
	tokens     float64
	lastRefill time.Time
	mu         sync.RWMutex
}

func NewTokenBucket(rate float64, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := elapsed * tb.rate

	if tokensToAdd > 0 {
		tb.tokens = math.Min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// SetParams para el modulo execute
func (tb *TokenBucket) SetParams(newRate, newCapacity float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.rate = newRate
	tb.capacity = newCapacity

	if tb.tokens > newCapacity {
		tb.tokens = newCapacity
	}
}

// Helper para obtener la IP real (o la falsa si estamos en modo test)
func GetClientIP(r *http.Request) string {
	// 1. ¿Estamos en modo TEST/ATAQUE?
	// Leemos la variable de entorno del sistema operativo
	if os.Getenv("TRUST_FAKE_IP") == "true" {
		// Leemos la cabecera que manda tu script de ataque
		fakeIP := r.Header.Get("X-Forwarded-For")
		if fakeIP != "" {
			// A veces viene como "10.0.0.1, 192.168.1.1", tomamos la primera
			ips := strings.Split(fakeIP, ",")
			return strings.TrimSpace(ips[0])
		}
	}

	// 2. MODO NORMAL (Producción)
	// Usamos la IP real de la conexión física
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
