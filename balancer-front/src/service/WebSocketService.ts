export const connectWebSocket = (onMessage: (data: any) => void) => {
  let socket: WebSocket | null = null;
  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;
  const reconnectDelay = 2000;
  let isIntentionallyClosed = false;

  const connect = () => {
    try {
      socket = new WebSocket("ws://localhost:9000/metrics/ws");

      socket.onopen = () => {
        console.log("✅ WebSocket conectado");
        reconnectAttempts = 0;
      };

      socket.onclose = (event) => {
        console.log("🔌 WebSocket cerrado", event.code);
        
        if (!isIntentionallyClosed && reconnectAttempts < maxReconnectAttempts) {
          reconnectAttempts++;
          console.log(`🔄 Reintentando conexión (${reconnectAttempts}/${maxReconnectAttempts})...`);
          setTimeout(connect, reconnectDelay);
        }
      };

      socket.onerror = (err) => console.error("❌ Error WS:", err);

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          // El backend ahora envía: {backends: [...], lb: {...}, rateLimit: {rate, burst}}
          if (data.backends && Array.isArray(data.backends)) {
            // Enviar métricas y parámetros de rate limit
            onMessage({
              backends: data.backends,
              lb: data.lb,
              rateLimit: data.rateLimit,
            });
          } else if (Array.isArray(data)) {
            // Compatibilidad con formato antiguo
            onMessage({ backends: data });
          } else {
            // Formato de mapa antiguo: {url: metrics}
            const metricsArray = Object.values(data);
            onMessage({ backends: metricsArray });
          }
        } catch (e) {
          console.error("❗ Error al parsear mensaje:", e);
          console.error("Datos recibidos:", event.data);
        }
      };
    } catch (error) {
      console.error("❌ Error al crear WebSocket:", error);
    }
  };

  connect();

  return {
    close: () => {
      isIntentionallyClosed = true;
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.close(1000, "Cierre intencional");
      }
    }
  };
}