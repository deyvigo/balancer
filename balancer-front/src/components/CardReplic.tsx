import { formatLatency, formatErrorRate, getLatencyColor, getHealthIcon, getHealthText, getHealthColor } from '../utils/dashboard'
import PerformanceChart from './PerformanceChart'
import { useEffect } from 'react'

export type CircuitState = 'CLOSED' | 'OPEN' | 'HALF_OPEN';

export interface ReplicInfo {
  id: number,
  url: string,
  ema_ms: number,
  error_rate: number,
  alive: boolean,
  last_checked: string,
  circuit_state: CircuitState
}

interface CardReplicProps extends Omit<ReplicInfo, "last_checked"> {
  addMetricPoint: (id: number, latency: number, errorRate: number, alive: boolean) => void
  getMetricsForBackend: (id: number) => Array<{ timestamp: number, latency: number, errorRate: number, alive: boolean }>
}

const getCircuitStateColor = (state: CircuitState) => {
  switch (state) {
    case 'CLOSED': return 'bg-green-100 text-green-800'
    case 'OPEN': return 'bg-red-100 text-red-800'
    case 'HALF_OPEN': return 'bg-yellow-100 text-yellow-800'
    default: return 'bg-gray-100 text-gray-800'
  }
}

const getCircuitStateIcon = (state: CircuitState) => {
  switch (state) {
    case 'CLOSED': return '🟢'
    case 'OPEN': return '🔴'
    case 'HALF_OPEN': return '🟡'
    default: return '⚪'
  }
}

export const CardReplic = ({ 
  id, 
  url, 
  ema_ms, 
  error_rate, 
  alive,
  circuit_state,
  addMetricPoint, 
  getMetricsForBackend 
}: CardReplicProps) => {
  // Agregar punto de métrica cuando los datos cambien
  useEffect(() => {
    addMetricPoint(id, ema_ms, error_rate, alive)
  }, [id, ema_ms, error_rate, alive, addMetricPoint])

  const metricsData = getMetricsForBackend(id)
  return (
    <div className="bg-white rounded-lg shadow-md p-4 sm:p-6 border-l-4 border-l-blue-500">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold text-gray-800">
          Backend #{id}
        </h3>
        <div className="flex flex-col items-end gap-1">
          <span className={`px-3 py-1 rounded-full text-xs font-medium ${getHealthColor(alive)}`}>
            {getHealthIcon(alive)} {getHealthText(alive)}
          </span>
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${getCircuitStateColor(circuit_state)}`}>
            {getCircuitStateIcon(circuit_state)} {circuit_state.replace('_', ' ')}
          </span>
        </div>
      </div>
      
      <div className="space-y-4">
        <div>
          <p className="text-sm text-gray-600">URL:</p>
          <p className="font-mono text-sm text-gray-800 bg-gray-50 px-2 py-1 rounded break-all">{url}</p>
        </div>
        
        {/* Métricas principales */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <p className="text-sm text-gray-600 mb-1">Latencia EMA:</p>
            <p className={`text-lg font-bold mb-2 ${getLatencyColor(ema_ms)}`}>
              {formatLatency(ema_ms)}
            </p>
            {/* Gráfico de latencia */}
            <div className="bg-gray-50 rounded p-2">
              <PerformanceChart 
                data={metricsData} 
                type="latency" 
                height={50} 
              />
            </div>
          </div>
          <div>
            <p className="text-sm text-gray-600 mb-1">Error Rate:</p>
            <p className="text-lg font-bold text-red-500 mb-2">{formatErrorRate(error_rate)}</p>
            {/* Gráfico de error rate */}
            <div className="bg-gray-50 rounded p-2">
              <PerformanceChart 
                data={metricsData} 
                type="errorRate" 
                height={50} 
              />
            </div>
          </div>
        </div>
        
        {/* Performance indicator */}
        <div className="pt-2 border-t">
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-600">Rendimiento:</span>
            <span className={`text-sm font-medium ${getLatencyColor(ema_ms)}`}>
              {ema_ms < 50 ? '⚡ Excelente' : ema_ms < 200 ? '⚠️ Bueno' : '🐌 Lento'}
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}