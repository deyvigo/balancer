package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TargetURL = "http://localhost:9000" // Tu Balanceador
)

// Contadores atómicos para ver las estadísticas en tiempo real
var (
	reqsSent    uint64
	reqsSuccess uint64 // HTTP 200
	reqsBlocked uint64 // HTTP 429
	reqsFailed  uint64 // Error de conexión
)

func main() {
	fmt.Println("=================================================")
	fmt.Println("   SIMULADOR DE TRÁFICO DISTRIBUIDO (DDoS)   ")
	fmt.Println("=================================================")
	fmt.Printf("Objetivo: %s\n", TargetURL)
	fmt.Println("Recuerda: Tu balanceador debe confiar en X-Forwarded-For")
	fmt.Println("-------------------------------------------------")
	fmt.Println("Selecciona intensidad:")
	fmt.Println("1. Tráfico Normal  (~50 RPS)")
	fmt.Println("2. Carga Alta      (~200 RPS) -> Debería activar HIGH_LOAD")
	fmt.Println("3. ATAQUE TOTAL    (Max Power) -> Debería activar ATTACK y 429s")

	var choice int
	fmt.Print("\nElige (1-3): ")
	fmt.Scan(&choice)

	var workers int
	var sleepTime time.Duration

	switch choice {
	case 1:
		workers = 5
		sleepTime = 100 * time.Millisecond // Lento
	case 2:
		workers = 20
		sleepTime = 10 * time.Millisecond // Rápido
	case 3:
		workers = 100
		sleepTime = 0 // Sin piedad
	default:
		workers = 1
		sleepTime = time.Second
	}

	fmt.Printf("\nLanzando %d atacantes simultáneos...\n", workers)

	// Canal para detener el ataque tras X segundos
	done := make(chan bool)
	go func() {
		time.Sleep(30 * time.Second) // Duración del test: 30 segundos
		close(done)
	}()

	// Iniciar reporte en tiempo real
	go reporter(done)

	// Iniciar workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(i, sleepTime, done, &wg)
	}

	wg.Wait()
	fmt.Println("\n--- TEST FINALIZADO ---")
}

func worker(id int, sleepTime time.Duration, done chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-done:
			return
		default:
			// 1. Generar IP aleatoria (Spoofing)
			fakeIP := fmt.Sprintf("%d.%d.%d.%d",
				rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))

			// 2. Crear petición
			req, _ := http.NewRequest("GET", TargetURL, nil)

			// IMPORTANTE: Cabecera para simular ser usuarios distintos
			req.Header.Set("X-Forwarded-For", fakeIP)
			req.Header.Set("User-Agent", "LoadTest-Bot/1.0")

			// 3. Enviar
			resp, err := client.Do(req)
			atomic.AddUint64(&reqsSent, 1)

			if err != nil {
				if atomic.LoadUint64(&reqsFailed) == 0 {
					fmt.Printf("\n[DEBUG] Error de red real: %v\n", err)
				}
				atomic.AddUint64(&reqsFailed, 1)
			} else {
				if resp.StatusCode == 200 {
					atomic.AddUint64(&reqsSuccess, 1)
				} else if resp.StatusCode == 429 {
					atomic.AddUint64(&reqsBlocked, 1)
				} else {
					// Otros errores (500, etc)
					atomic.AddUint64(&reqsFailed, 1)
				}
				resp.Body.Close()
			}

			// Control de velocidad
			if sleepTime > 0 {
				time.Sleep(sleepTime)
			}
		}
	}
}

func reporter(done chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var prevSent uint64 = 0

	fmt.Println("\nTIEMPO | RPS Real | 200 OK | 429 BLOCK | ERRORES")
	fmt.Println("-------+----------+--------+-----------+--------")

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			currSent := atomic.LoadUint64(&reqsSent)
			currSuccess := atomic.LoadUint64(&reqsSuccess)
			currBlocked := atomic.LoadUint64(&reqsBlocked)
			currFailed := atomic.LoadUint64(&reqsFailed)

			rps := currSent - prevSent
			prevSent = currSent

			fmt.Printf("%s | %6d   | %6d | %9d | %6d\n",
				time.Now().Format("15:04:05"), rps, currSuccess, currBlocked, currFailed)
		}
	}
}
