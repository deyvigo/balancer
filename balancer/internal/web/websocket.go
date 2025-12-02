package web

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal/monitor"
	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	Monitor *monitor.MonitorService
}

func (ws *WebSocketServer) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[websocket] error al abrir conexión: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[websocket] cliente conectado al /metrics/ws")

	// Configurar ping/pong para detectar desconexiones
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		return nil
	})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Println("[websocket] cliente desconectado (context done)")
			return
		case <-pingTicker.C:
			// Enviar ping para mantener conexión viva
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("[websocket] error al enviar ping:", err)
				return
			}
		case <-ticker.C:
			metrics := ws.Monitor.SnapshotMetrics()
			data, err := json.Marshal(metrics)
			if err != nil {
				log.Printf("[websocket] error al serializar métricas: %v", err)
				continue
			}

			// Extender deadline antes de escribir
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				// No loguear broken pipe, es normal cuando el cliente se desconecta
				if !isConnectionError(err) {
					log.Println("[websocket] error al escribir mensaje:", err)
				}
				return
			}
		}
	}
}

// isConnectionError verifica si el error es una desconexión normal
func isConnectionError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "broken pipe") ||
		contains(errStr, "connection reset by peer") ||
		contains(errStr, "use of closed network connection")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
