import React from 'react'
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer } from 'recharts'
import type { MetricDataPoint } from '../hooks/useMetricsHistory'

interface PerformanceChartProps {
  data: MetricDataPoint[]
  type: 'latency' | 'errorRate'
  height?: number
}

const PerformanceChart: React.FC<PerformanceChartProps> = ({ 
  data, 
  type, 
  height = 60 
}) => {
  // Formatear datos para el gráfico
  const chartData = data.map((point, index) => ({
    index,
    value: type === 'latency' ? point.latency : (point.errorRate * 100), // Error rate como porcentaje
    alive: point.alive
  }))

  // Color basado en el estado y tipo
  const getAreaColor = () => {
    if (type === 'latency') {
      return '#22c55e' // Verde para latencia (como Task Manager)
    } else {
      return '#ef4444' // Rojo para error rate
    }
  }

  const getStrokeColor = () => {
    if (type === 'latency') {
      return '#16a34a' // Verde más oscuro
    } else {
      return '#dc2626' // Rojo más oscuro
    }
  }

  // Si no hay suficientes datos, mostrar línea plana
  const minDataPoints = 2
  const displayData = chartData.length >= minDataPoints 
    ? chartData 
    : Array.from({ length: 20 }, (_, i) => ({ 
        index: i, 
        value: 0, 
        alive: true 
      }))

  return (
    <div className="w-full" style={{ height: `${height}px` }}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={displayData}
          margin={{ top: 2, right: 2, left: 2, bottom: 2 }}
        >
          <Area
            type="monotone"
            dataKey="value"
            stroke={getStrokeColor()}
            strokeWidth={1.5}
            fill={getAreaColor()}
            fillOpacity={0.4}
            dot={false}
            connectNulls={false}
          />
          <XAxis hide />
          <YAxis hide />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}

export default PerformanceChart