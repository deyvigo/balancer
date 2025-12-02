export interface BackendMetrics {
  id: number
  url: string
  ema_ms: number
  error_rate: number
  alive: boolean
  last_checked: string
}

export interface DashboardData {
  backends: BackendMetrics[]
  rateLimitStats: any
  circuitBreakerStats: any
  loadBalancerStats: any
}

export const formatLatency = (ms: number): string => {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`
  if (ms < 1000) return `${ms.toFixed(1)}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

export const formatErrorRate = (rate: number): string => {
  return `${(rate * 100).toFixed(2)}%`
}

export const getLatencyColor = (ms: number): string => {
  if (ms < 50) return 'text-green-600'
  if (ms < 200) return 'text-yellow-600'
  return 'text-red-600'
}

export const getLatencyBg = (ms: number): string => {
  if (ms < 50) return 'bg-green-100'
  if (ms < 200) return 'bg-yellow-100'
  return 'bg-red-100'
}

export const getHealthIcon = (alive: boolean): string => {
  return alive ? '🟢' : '🔴'
}

export const getHealthText = (alive: boolean): string => {
  return alive ? 'Online' : 'Offline'
}

export const getHealthColor = (alive: boolean): string => {
  return alive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
}