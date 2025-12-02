import { useState, useCallback } from 'react'

export interface MetricDataPoint {
  timestamp: number
  latency: number
  errorRate: number
  alive: boolean
}

export interface BackendMetrics {
  [backendId: number]: MetricDataPoint[]
}

const MAX_DATA_POINTS = 60 // Últimos 60 puntos de datos (aprox 5 minutos con updates cada 5s)

export const useMetricsHistory = () => {
  const [metricsHistory, setMetricsHistory] = useState<BackendMetrics>({})

  const addMetricPoint = useCallback((backendId: number, latency: number, errorRate: number, alive: boolean) => {
    const timestamp = Date.now()
    
    setMetricsHistory(prev => {
      const currentHistory = prev[backendId] || []
      const newDataPoint: MetricDataPoint = {
        timestamp,
        latency,
        errorRate,
        alive
      }
      
      // Agregar nuevo punto y mantener solo los últimos MAX_DATA_POINTS
      const updatedHistory = [...currentHistory, newDataPoint].slice(-MAX_DATA_POINTS)
      
      return {
        ...prev,
        [backendId]: updatedHistory
      }
    })
  }, [])

  const getMetricsForBackend = useCallback((backendId: number): MetricDataPoint[] => {
    return metricsHistory[backendId] || []
  }, [metricsHistory])

  const clearHistory = useCallback(() => {
    setMetricsHistory({})
  }, [])

  return {
    addMetricPoint,
    getMetricsForBackend,
    clearHistory,
    metricsHistory
  }
}
