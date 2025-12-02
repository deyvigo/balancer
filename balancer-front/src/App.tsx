import { useEffect, useState } from "react"
import { CardReplic, type ReplicInfo } from "./components/CardReplic"
import { connectWebSocket } from "./service/WebSocketService"
import { DashboardHeader } from "./components/DashboardHeader"
import { useMetricsHistory } from "./hooks/useMetricsHistory"

function App() {
  const [replics, setReplics] = useState<ReplicInfo[]>([])
  const [rateLimit, setRateLimit] = useState({ rate: 50, burst: 10 })
  
  // Hook para manejar métricas históricas
  const { addMetricPoint, getMetricsForBackend } = useMetricsHistory()

  useEffect(() => {
    // Conexión WebSocket para métricas en tiempo real
    const wsConnection = connectWebSocket((data) => {
      // data viene como {backends: [...], lb: {...}, rateLimit: {...}}
      if (data.backends && Array.isArray(data.backends)) {
        setReplics(data.backends)
        if (data.rateLimit) {
          setRateLimit(data.rateLimit)
        }
      } else if (Array.isArray(data)) {
        // Compatibilidad con formato antiguo
        setReplics(data)
      }
    })

    return () => {
      wsConnection.close()
    }
  }, [])

  // Calcular estadísticas desde los datos de WebSocket
  const activeBackends = replics.filter(r => r.alive).length
  const avgLatency = replics.length > 0 
    ? replics.reduce((acc, r) => acc + (isNaN(r.ema_ms) ? 0 : r.ema_ms), 0) / replics.length 
    : 0
  const avgErrorRate = replics.length > 0 
    ? replics.reduce((acc, r) => acc + (isNaN(r.error_rate) ? 0 : r.error_rate), 0) / replics.length 
    : 0
  const circuitsClosed = replics.filter(r => r.circuit_state === 'CLOSED').length
  const circuitsOpen = replics.filter(r => r.circuit_state === 'OPEN').length
  const circuitsHalfOpen = replics.filter(r => r.circuit_state === 'HALF_OPEN').length

  return (
    <div className="min-h-screen bg-gray-100 p-6">
      <DashboardHeader />

      {/* Stats Cards calculadas desde WebSocket */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">🖥️ Backends</h3>
          <div className="text-3xl font-bold text-green-600">{activeBackends}/{replics.length}</div>
          <p className="text-sm text-gray-500">Activos</p>
        </div>
        
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">⚡ Latencia</h3>
          <div className="text-3xl font-bold text-blue-600">{avgLatency.toFixed(1)}ms</div>
          <p className="text-sm text-gray-500">Promedio EMA</p>
        </div>
        
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">❌ Error Rate</h3>
          <div className="text-3xl font-bold text-red-600">{(avgErrorRate * 100).toFixed(1)}%</div>
          <p className="text-sm text-gray-500">Promedio</p>
        </div>
        
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">🛡️ Circuit Breakers</h3>
          <div className="flex gap-2 text-sm">
            <span className="px-2 py-1 bg-green-100 text-green-800 rounded">🟢 {circuitsClosed}</span>
            <span className="px-2 py-1 bg-yellow-100 text-yellow-800 rounded">🟡 {circuitsHalfOpen}</span>
            <span className="px-2 py-1 bg-red-100 text-red-800 rounded">🔴 {circuitsOpen}</span>
          </div>
        </div>
      </div>

      {/* Rate Limit Status */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">🚦 Rate Limit</h3>
          <div className="text-3xl font-bold text-purple-600">{rateLimit.rate}</div>
          <p className="text-sm text-gray-500">peticiones/segundo</p>
        </div>
        
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">💥 Burst</h3>
          <div className="text-3xl font-bold text-orange-600">{rateLimit.burst}</div>
          <p className="text-sm text-gray-500">ráfaga máxima</p>
        </div>
        
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">🛡️ Estado</h3>
          <div className="text-3xl font-bold text-green-600">
            {rateLimit.rate <= 30 ? '🟢 Normal' : 
             rateLimit.rate <= 80 ? '🟡 Alta Carga' : '🔴 Ataque'}
          </div>
          <p className="text-sm text-gray-500">modo actual</p>
        </div>
        
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-2">📈 Cambios</h3>
          <div className="text-3xl font-bold text-blue-600">
            {rateLimit.rate === 50 ? '✅' : '🔄'}
          </div>
          <p className="text-sm text-gray-500">última actualización</p>
        </div>
      </div>

      {/* Backend Replicas */}
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-800 mb-4">Backend Services</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {
            replics.map(({ id, ema_ms, error_rate, alive, url, circuit_state }) => 
              <CardReplic 
                key={id} 
                id={id} 
                ema_ms={ema_ms} 
                error_rate={error_rate} 
                alive={alive} 
                url={url}
                circuit_state={circuit_state}
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
