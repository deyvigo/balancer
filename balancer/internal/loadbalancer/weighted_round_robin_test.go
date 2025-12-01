package loadbalancer

import (
	"testing"

	"github.com/deyvigo/balanceador/balancer/internal"
)

func TestCalculateWeight(t *testing.T) {
	wrr := NewWeightedRoundRobin()

	tests := []struct {
		name       string
		metric     internal.Metrics
		expectZero bool
		expectLow  bool // peso bajo (1-30)
		expectHigh bool // peso alto (>60)
	}{
		{
			name: "Backend saludable - baja latencia, sin errores",
			metric: internal.Metrics{
				Id:           1,
				URL:          "http://localhost:8080",
				Alive:        true,
				EMAMs:        10.0,
				ErrorRate:    0.0,
				CircuitState: internal.StateClosed,
			},
			expectHigh: true,
		},
		{
			name: "Backend lento - alta latencia",
			metric: internal.Metrics{
				Id:           2,
				URL:          "http://localhost:8081",
				Alive:        true,
				EMAMs:        500.0,
				ErrorRate:    0.0,
				CircuitState: internal.StateClosed,
			},
			expectLow: true,
		},
		{
			name: "Backend con muchos errores",
			metric: internal.Metrics{
				Id:           3,
				URL:          "http://localhost:8082",
				Alive:        true,
				EMAMs:        50.0,
				ErrorRate:    0.8,
				CircuitState: internal.StateClosed,
			},
			expectLow: true,
		},
		{
			name: "Backend caído (Alive=false)",
			metric: internal.Metrics{
				Id:           4,
				URL:          "http://localhost:8083",
				Alive:        false,
				EMAMs:        0.0,
				ErrorRate:    1.0,
				CircuitState: internal.StateOpen,
			},
			expectZero: true,
		},
		{
			name: "Backend en HALF_OPEN (probando reconexión)",
			metric: internal.Metrics{
				Id:           5,
				URL:          "http://localhost:8084",
				Alive:        true,
				EMAMs:        20.0,
				ErrorRate:    0.1,
				CircuitState: internal.StateHalfOpen,
			},
			expectLow: true,
		},
		{
			name: "Backend con circuito OPEN",
			metric: internal.Metrics{
				Id:           6,
				URL:          "http://localhost:8085",
				Alive:        true,
				EMAMs:        20.0,
				ErrorRate:    0.0,
				CircuitState: internal.StateOpen,
			},
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := wrr.calculateWeight(tt.metric)

			if tt.expectZero && weight != 0 {
				t.Errorf("Esperaba peso=0, obtuvo %d", weight)
			}

			if !tt.expectZero && weight == 0 {
				t.Errorf("Esperaba peso>0, obtuvo 0")
			}

			if tt.expectLow && weight > 30 {
				t.Errorf("Esperaba peso bajo (1-30), obtuvo %d", weight)
			}

			if tt.expectHigh && weight < 60 {
				t.Errorf("Esperaba peso alto (>60), obtuvo %d", weight)
			}

			t.Logf("✓ %s → Peso: %d", tt.name, weight)
		})
	}
}

func TestWeightedRoundRobin_Distribution(t *testing.T) {
	wrr := NewWeightedRoundRobin()

	// Simular 3 backends con diferentes rendimientos
	metrics := []internal.Metrics{
		{
			Id:           0,
			URL:          "http://backend-rapido:8080",
			Alive:        true,
			EMAMs:        10.0, // MUY RÁPIDO
			ErrorRate:    0.0,  // SIN ERRORES
			CircuitState: internal.StateClosed,
		},
		{
			Id:           1,
			URL:          "http://backend-medio:8081",
			Alive:        true,
			EMAMs:        50.0, // MEDIO
			ErrorRate:    0.2,  // 20% ERRORES
			CircuitState: internal.StateClosed,
		},
		{
			Id:           2,
			URL:          "http://backend-lento:8082",
			Alive:        true,
			EMAMs:        200.0, // LENTO
			ErrorRate:    0.1,   // 10% ERRORES
			CircuitState: internal.StateClosed,
		},
	}

	wrr.UpdateMetrics(metrics)

	// Verificar que se crearon backends
	weights := wrr.GetBackendWeights()
	if len(weights) != 3 {
		t.Fatalf("Esperaba 3 backends, obtuvo %d", len(weights))
	}

	t.Logf("Pesos calculados:")
	for _, w := range weights {
		t.Logf("  %s → Peso: %d", w.URL, w.Weight)
	}

	// El backend más rápido debe tener el mayor peso
	if weights[0].Weight <= weights[1].Weight || weights[0].Weight <= weights[2].Weight {
		t.Errorf("El backend rápido debería tener el mayor peso")
	}

	// Probar distribución de peticiones
	selections := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		backend, ok := wrr.NextBackend()
		if !ok {
			t.Fatal("NextBackend() falló")
		}
		selections[backend]++
	}

	t.Logf("\nDistribución de %d peticiones:", iterations)
	for url, count := range selections {
		percentage := float64(count) / float64(iterations) * 100
		t.Logf("  %s: %d peticiones (%.1f%%)", url, count, percentage)
	}

	// El backend rápido debería recibir MÁS tráfico
	rapidoCount := selections["http://backend-rapido:8080"]
	medioCount := selections["http://backend-medio:8081"]
	lentoCount := selections["http://backend-lento:8082"]

	if rapidoCount <= medioCount || rapidoCount <= lentoCount {
		t.Errorf("El backend rápido debería recibir más peticiones que los demás")
	}
}

func TestNoBackendsAvailable(t *testing.T) {
	wrr := NewWeightedRoundRobin()

	// Todos los backends caídos
	metrics := []internal.Metrics{
		{
			Id:           0,
			URL:          "http://localhost:8080",
			Alive:        false,
			CircuitState: internal.StateOpen,
		},
		{
			Id:           1,
			URL:          "http://localhost:8081",
			Alive:        false,
			CircuitState: internal.StateOpen,
		},
	}

	wrr.UpdateMetrics(metrics)

	_, ok := wrr.NextBackend()
	if ok {
		t.Error("NextBackend() debería retornar false cuando no hay backends disponibles")
	}
}

func TestGCD(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{10, 5, 5},
		{12, 8, 4},
		{7, 13, 1},
		{100, 50, 50},
		{0, 5, 5},
		{17, 19, 1},
	}

	for _, tt := range tests {
		result := gcd(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("gcd(%d, %d) = %d, esperaba %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestUpdateMetrics_EmptyList(t *testing.T) {
	wrr := NewWeightedRoundRobin()
	wrr.UpdateMetrics([]internal.Metrics{})

	_, ok := wrr.NextBackend()
	if ok {
		t.Error("NextBackend() debería retornar false con lista vacía")
	}
}
