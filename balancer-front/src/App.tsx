import { useEffect, useState } from "react"
import { CardReplic, type ReplicInfo } from "./components/CardReplic"
import { connectWebSocket } from "./service/WebSocketService"
import { RateLimitDashboard, type RateLimitData } from "./components/RateLimitDashboard"
import { CircuitBreakerDashboard, type CircuitBreakerData } from "./components/CircuitBreakerDashboard"
import { LoadBalancerStats, type LoadBalancerStatsData } from "./components/LoadBalancerStats"
import { DashboardHeader } from "./components/DashboardHeader"
import { useMetricsHistory } from "./hooks/useMetricsHistory"

function App() {
  const [replics, setReplics] = useState<ReplicInfo[]>([])
  const [rateLimitStats, setRateLimitStats] = useState<RateLimitData | null>(null)
  const [circuitBreakerStats, setCircuitBreakerStats] = useState<CircuitBreakerData | null>(null)
  const [loadBalancerStats, setLoadBalancerStats] = useState<LoadBalancerStatsData | null>(null)
  
  // Hook para manejar métricas históricas
  const { addMetricPoint, getMetricsForBackend } = useMetricsHistory()

  // Función para obtener datos de rate limiting
  const fetchRateLimitStats = async () => {
    try {
      const response = await fetch('http://localhost:9000/api/rate-limit')
      const data = await response.json()
      setRateLimitStats(data.data)
    } catch (error) {
      console.error('Error fetching rate limit stats:', error)
    }
  }

  // Función para obtener datos de circuit breakers
  const fetchCircuitBreakerStats = async () => {
    try {
      const response = await fetch('http://localhost:9000/api/circuit-breaker')
      const data = await response.json()
      setCircuitBreakerStats(data.data)
    } catch (error) {
      console.error('Error fetching circuit breaker stats:', error)
    }
  }

  // Función para obtener métricas generales
  const fetchLoadBalancerStats = async () => {
    try {
      const response = await fetch('http://localhost:9000/api/metrics')
      const data = await response.json()
      setLoadBalancerStats(data.data)
    } catch (error) {
      console.error('Error fetching load balancer stats:', error)
    }
  }

  useEffect(() => {
    // Conexión WebSocket para métricas en tiempo real
    const socket = connectWebSocket((data) => {
      if (Array.isArray(data)) {
        setReplics(data)
      } else {
        setReplics((prev) => {
          const idx = prev.findIndex((r) => r.id === data.id)
          if (idx !== -1) {
            const updated = [...prev]
            updated[idx] = data
            return updated
          }
          return [...prev, data]
        })
      }
    })

    // Fetch inicial de datos
    fetchRateLimitStats()
    fetchCircuitBreakerStats()
    fetchLoadBalancerStats()

    // Actualizar datos cada 5 segundos
    const interval = setInterval(() => {
      fetchRateLimitStats()
      fetchCircuitBreakerStats()
      fetchLoadBalancerStats()
    }, 5000)

    return () => {
      socket.close()
      clearInterval(interval)
    }
  }, [])
  return (
    <div className="min-h-screen bg-gray-100 p-6">
      <DashboardHeader />

      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6 mb-8">
        {/* Stats Cards */}
        <LoadBalancerStats data={loadBalancerStats} />
        <RateLimitDashboard data={rateLimitStats} />
        <CircuitBreakerDashboard data={circuitBreakerStats} />
      </div>

      {/* Backend Replicas */}
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-800 mb-4">Backend Services</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {
            replics.map(({ id, ema_ms, error_rate, alive, url }) => 
              <CardReplic 
                key={id} 
                id={id} 
                ema_ms={ema_ms} 
                error_rate={error_rate} 
                alive={alive} 
                url={url}
                addMetricPoint={addMetricPoint}
                getMetricsForBackend={getMetricsForBackend}
              />
            )
          }
        </div>
        {replics.length === 0 && (
          <div className="text-center py-8">
            <p className="text-gray-500">Conectando con los backends...</p>
            <div className="mt-4">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default App
