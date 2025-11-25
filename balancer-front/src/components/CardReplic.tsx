import { formatLatency, formatErrorRate, getLatencyColor, getHealthIcon, getHealthText, getHealthColor } from '../utils/dashboard'
import PerformanceChart from './PerformanceChart'
import { useMetricsHistory } from '../hooks/useMetricsHistory'
import { useEffect } from 'react'

export interface ReplicInfo {
  id: number,
  url: string,
  ema_ms: number,
  error_rate: number,
  alive: boolean,
  last_checked: Date
}

interface CardReplicProps extends Omit<ReplicInfo, "last_checked"> {
  addMetricPoint: (id: number, latency: number, errorRate: number, alive: boolean) => void
  getMetricsForBackend: (id: number) => Array<{ timestamp: number, latency: number, errorRate: number, alive: boolean }>
}

export const CardReplic = ({ 
  id, 
  url, 
  ema_ms, 
  error_rate, 
  alive, 
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
        <span className={`px-3 py-1 rounded-full text-xs font-medium ${getHealthColor(alive)}`}>
          {getHealthIcon(alive)} {getHealthText(alive)}
        </span>
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