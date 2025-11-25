# 🚀 Enterprise Load Balancer

Un **Load Balancer empresarial** completo desarrollado en **Go** con características avanzadas para distribución de carga, rate limiting, circuit breakers y monitoreo en tiempo real.

## 📑 Tabla de Contenidos

- [🎯 Características Principales](#-características-principales)
- [🏗️ Arquitectura del Sistema](#️-arquitectura-del-sistema)
- [📊 Componentes](#-componentes)
- [🚀 Inicio Rápido](#-inicio-rápido)
- [⚙️ Configuración](#️-configuración)
- [🧪 Scripts de Pruebas](#-scripts-de-pruebas)
- [📈 Dashboard Web](#-dashboard-web)
- [🔧 API Endpoints](#-api-endpoints)
- [📖 Documentación Técnica](#-documentación-técnica)
- [🛠️ Desarrollo](#️-desarrollo)

## 🎯 Características Principales

### ⚖️ **Algoritmos de Load Balancing**
- **Round Robin**: Distribución secuencial entre backends
- **Weighted Round Robin**: Distribución basada en pesos configurables
- **Least Connections**: Redirige al backend con menos conexiones activas
- **Adaptive Weights**: Optimización automática de pesos basada en rendimiento

### 🛡️ **Protecciones y Límites**
- **Circuit Breaker**: Protección automática contra backends fallidos
- **Rate Limiting**: Control de tráfico con Token Bucket y Sliding Window
- **Health Checks**: Monitoreo continuo de estado de backends
- **Timeout Management**: Gestión inteligente de timeouts

### 📊 **Monitoreo y Observabilidad**
- **WebSocket en tiempo real**: Métricas actualizadas cada 5 segundos
- **Dashboard web interactivo**: Visualización con React + TailwindCSS
- **APIs de métricas**: Endpoints RESTful para integración
- **Gráficos de rendimiento**: Estilo Windows Task Manager

### 🔧 **Características Técnicas**
- **Configuración JSON**: Archivo de configuración flexible
- **Hot Reloading**: Recarga de configuración sin reiniciar
- **Logging estructurado**: Logs detallados para debugging
- **Docker Ready**: Containerización completa

## 🏗️ Arquitectura del Sistema

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client        │    │   Load Balancer  │    │   Backend       │
│   Requests      │───▶│   (Port 8089)    │───▶│   Services      │
│                 │    │                  │    │   8080-8082     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   Admin API      │
                       │   (Port 9000)    │
                       │   Metrics &      │
                       │   Management     │
                       └──────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   Web Dashboard  │
                       │   (Port 5173)    │
                       │   React Frontend │
                       └──────────────────┘
```

### 🔄 **Flujo de Datos**

1. **Request Handling**: Cliente → Load Balancer → Backend
2. **Health Monitoring**: Monitor → Backends (cada 5s)
3. **Metrics Collection**: Backends → Monitor → WebSocket
4. **Weight Optimization**: Analyzer → Weight Calculator → Config Update
5. **Circuit Breaker**: Failure Detection → State Management → Recovery

## 📊 Componentes

### 🎯 **Core Components**

#### **1. Load Balancer Core** (`balancer/cmd/main.go`)
- Punto de entrada principal
- Inicialización de todos los componentes
- Gestión del ciclo de vida de la aplicación

#### **2. Proxy Engine** (`balancer/internal/proxy/`)
- Motor de proxying HTTP
- Implementación de algoritmos de balanceo
- Gestión de conexiones y timeouts

#### **3. Health Monitor** (`balancer/internal/monitor/`)
- Monitoreo continuo de backends
- Cálculo de métricas EMA (Exponential Moving Average)
- Detección automática de fallos

#### **4. Circuit Breaker** (`balancer/internal/breaker/`)
- Implementación del patrón Circuit Breaker
- Estados: CLOSED → OPEN → HALF_OPEN
- Recuperación automática basada en thresholds

#### **5. Rate Limiter** (`balancer/internal/ratelimiter/`)
- **Token Bucket**: Para ráfagas controladas
- **Sliding Window**: Para límites temporales
- Control global y por IP

#### **6. Weight Optimizer** (`balancer/internal/optimizer/`)
- Optimización automática de pesos
- Basado en latencia y error rate
- Algoritmo de adaptación gradual

### 🌐 **Frontend Components**

#### **React Dashboard** (`balancer-front/`)
- **Dashboard principal**: Métricas en tiempo real
- **Cards de backends**: Con gráficos de rendimiento
- **Rate Limit Monitor**: Estado del rate limiter
- **Circuit Breaker Status**: Estado de protecciones
- **Responsive Design**: Adaptable a móviles

### 🧪 **Testing & Scripts**

#### **Scripts de Pruebas** (`scripts/`)
- Suite completa de testing automatizado
- Pruebas de carga, estrés y funcionales
- Monitoreo en tiempo real
- Análisis de distribución

## 🚀 Inicio Rápido

### 📋 **Prerrequisitos**

- **Go 1.21+** - [Instalar Go](https://golang.org/dl/)
- **Docker & Docker Compose** - [Instalar Docker](https://docs.docker.com/get-docker/)
- **Node.js & PNPM** (para frontend) - [Instalar Node](https://nodejs.org/)

### ⚡ **Preparación de entorno**

```powershell
# Clona el repositorio
git clone https://github.com/deyvigo/balancer.git
cd balancer

# Instala dependencias Go
go mod tidy
```

###  **Inicio de entorno**

#### **1. Iniciar Backends**
```powershell
# Construir e iniciar servicios Docker
docker-compose up --build --scale go-service=3 -d

# Verificar que están funcionando
docker-compose ps
```

#### **2. Iniciar Load Balancer**
```powershell
# Compilar y ejecutar
go run ./balancer/cmd/main.go

# O compilar primero
go build -o balancer.exe ./balancer/cmd
./balancer.exe
```

#### **3. Iniciar Frontend (Opcional)**
```powershell
cd balancer-front
pnpm install
pnpm dev
```

### ✅ **Verificación**

```powershell
# Probar el load balancer
curl http://localhost:8089/api/hello

# Verificar métricas
curl http://localhost:9000/api/metrics

# Acceder al dashboard
# Navegador: http://localhost:5173
```

## ⚙️ Configuración

### 📄 **Archivo de Configuración** (`config.json`)

```json
{
  "backends": [
    {
      "url": "http://localhost:8080",
      "weight": 1.0,
      "enabled": true
    }
  ],
  "proxy": {
    "algorithm": "round_robin",        // round_robin, weighted_round_robin, least_connections
    "retry_attempts": 2,
    "retry_delay_ms": 100,
    "timeout_ms": 10000,
    "port": 8089
  },
  "monitor": {
    "alpha": 0.2,                     // Factor de suavizado EMA
    "period_s": 5,                    // Intervalo de health checks
    "timeout_s": 30                   // Timeout para health checks
  },
  "circuit_breaker": {
    "enabled": true,
    "failure_threshold": 5,           // Fallos consecutivos para abrir
    "error_rate_threshold": 0.5,      // Tasa de error para abrir (50%)
    "open_timeout_s": 30,             // Tiempo en estado abierto
    "half_open_max_calls": 3,         // Requests de prueba en half-open
    "min_request_count": 5            // Mínimo de requests para evaluar
  },
  "rate_limit": {
    "enabled": true,
    "type": "token_bucket",           // token_bucket, sliding_window
    "global_limit": 1000,             // Requests por minuto globalmente
    "per_ip_limit": 100,              // Requests por IP por minuto
    "refill_rate": 10,                // Tokens por segundo
    "whitelist": ["127.0.0.1"]       // IPs sin límites
  },
  "weight_optimization": {
    "enabled": true,
    "latency_weight": 0.6,            // Peso de latencia en optimización
    "error_rate_weight": 0.4,         // Peso de error rate en optimización
    "adaptation_speed": 0.1,          // Velocidad de adaptación (0.0-1.0)
    "update_interval_s": 10           // Intervalo de actualización
  }
}
```

### 🎛️ **Algoritmos Disponibles**

| Algoritmo | Descripción | Uso Recomendado |
|-----------|-------------|-----------------|
| `round_robin` | Distribución secuencial | Backends similares |
| `weighted_round_robin` | Basado en pesos | Backends con diferente capacidad |
| `least_connections` | Menor carga activa | Conexiones de larga duración |

### 🔧 **Parámetros Críticos**

- **`alpha`**: Factor de suavizado para EMA (0.1-0.3 recomendado)
- **`failure_threshold`**: Fallos para activar circuit breaker
- **`error_rate_threshold`**: Porcentaje de errores límite
- **`adaptation_speed`**: Velocidad de optimización de pesos

## 🧪 Scripts de Pruebas

### 📊 **Scripts Disponibles**

| Script | Propósito | Duración Aprox. |
|--------|-----------|----------------|
| `basic_load_test.ps1` | Prueba básica de conectividad | 30 segundos |
| `stress_test.ps1` | Carga concurrente intensa | 1-2 minutos |
| `rate_limit_test.ps1` | Verificar rate limiting | 45 segundos |
| `circuit_breaker_test.ps1` | Probar circuit breakers | 1 minuto |
| `load_balancing_test.ps1` | Análisis de distribución | 1 minuto |
| `monitor_realtime.ps1` | Monitoreo continuo | ∞ (hasta Ctrl+C) |
| `comprehensive_test.ps1` | Suite completa | 5-7 minutos |

### 🚀 **Ejecución de Pruebas**

```powershell
# Prueba rápida
.\scripts\basic_load_test.ps1

# Ver sistema en acción
.\scripts\monitor_realtime.ps1

# Suite completa de pruebas
.\scripts\comprehensive_test.ps1

# Probar característica específica
.\scripts\circuit_breaker_test.ps1
```

### 📈 **Interpretación de Resultados**

#### **✅ Indicadores Positivos**
- Tasa de éxito > 95%
- Coeficiente de variación < 20% (balance)
- Circuit breakers funcionando correctamente
- Rate limiting activándose según configuración

#### **⚠️ Señales de Alerta**
- Errores de timeout > 5%
- Distribución desigual entre backends
- Circuit breakers constantemente abiertos
- Rate limiting no funcionando

## 📈 Dashboard Web

### 🎨 **Interfaz Principal**

El dashboard web proporciona una vista en tiempo real del estado del sistema:

#### **📊 Secciones del Dashboard**

1. **Header**: Información general y controles
2. **Stats Cards**: Métricas principales del load balancer
3. **Rate Limit Monitor**: Estado del rate limiter en tiempo real
4. **Circuit Breaker Status**: Estado de cada circuit breaker
5. **Backend Cards**: Métricas individuales por backend

#### **📈 Gráficos de Rendimiento**

Cada backend muestra gráficos estilo Windows Task Manager:
- **Gráfico verde**: Latencia en tiempo real
- **Gráfico rojo**: Error rate histórico
- **Responsive**: Se adapta a cualquier pantalla

#### **🔄 Actualización en Tiempo Real**

- **WebSocket**: Conexión persistente para métricas
- **Intervalo**: Actualización cada 5 segundos
- **REST APIs**: Endpoints complementarios para datos específicos

### 🌐 **URLs del Sistema**

| Servicio | URL | Descripción |
|----------|-----|-------------|
| Load Balancer | http://localhost:8089 | Proxy principal |
| Admin API | http://localhost:9000 | APIs de gestión |
| Dashboard | http://localhost:5173 | Interfaz web |
| Backends | http://localhost:8080-8082 | Servicios de prueba |

## 🔧 API Endpoints

### 📊 **Métricas y Estado**

#### **GET** `/api/metrics`
```json
{
  "success": true,
  "data": {
    "algorithm": "round_robin",
    "total_requests": 1250,
    "active_backends": 3,
    "avg_response_time": 45.6,
    "requests_per_minute": 125.3
  }
}
```

#### **GET** `/api/backends`
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "url": "http://localhost:8080",
      "ema_ms": 42.5,
      "error_rate": 0.02,
      "alive": true,
      "weight": 1.2,
      "connections": 5
    }
  ]
}
```

#### **GET** `/api/rate-limit`
```json
{
  "success": true,
  "data": {
    "enabled": true,
    "type": "token_bucket",
    "global_limit": 1000,
    "per_ip_limit": 100,
    "active_ips": 15,
    "global_tokens": 856
  }
}
```

#### **GET** `/api/circuit-breaker`
```json
{
  "success": true,
  "data": {
    "backend-1": {
      "state": "CLOSED",
      "failure_count": 0,
      "error_rate": 0.01,
      "last_failure_time": null,
      "next_attempt": null
    }
  }
}
```

### 🔄 **WebSocket** `/ws/metrics`

**Conexión**: `ws://localhost:9000/ws/metrics`

**Datos enviados cada 5s**:
```json
[
  {
    "id": 1,
    "url": "http://localhost:8080",
    "ema_ms": 45.2,
    "error_rate": 0.01,
    "alive": true,
    "last_checked": "2025-11-25T10:30:15Z"
  }
]
```

### ⚙️ **Configuración** (Futuras versiones)

#### **POST** `/api/config/reload`
Recarga la configuración sin reiniciar el servicio.

#### **PUT** `/api/backends/{id}/enable`
Habilita/deshabilita un backend específico.

## 📖 Documentación Técnica

### 🏗️ **Patrones de Diseño Implementados**

#### **1. Circuit Breaker Pattern**
- **Estados**: CLOSED → OPEN → HALF_OPEN
- **Métricas**: Failure count, error rate, temporal windows
- **Recuperación**: Automática basada en timeouts

#### **2. Health Check Pattern**
- **Estrategia**: Polling activo cada 5 segundos
- **Métricas**: EMA para latencia, contadores de errores
- **Failover**: Automático cuando un backend falla

#### **3. Observer Pattern**
- **WebSocket**: Para notificaciones en tiempo real
- **Event-driven**: Actualizaciones basadas en eventos
- **Decoupling**: Separación entre lógica y presentación

#### **4. Strategy Pattern**
- **Algoritmos**: Intercambiables de load balancing
- **Rate Limiters**: Múltiples estrategias disponibles
- **Extensibilidad**: Fácil agregar nuevos algoritmos

### 🔧 **Estructura de Código**

```
balancer/
├── cmd/main.go                 # Punto de entrada
├── internal/
│   ├── api/                    # Endpoints REST y WebSocket
│   ├── breaker/                # Circuit Breaker implementation
│   ├── config/                 # Gestión de configuración
│   ├── monitor/                # Health monitoring
│   ├── optimizer/              # Weight optimization
│   ├── proxy/                  # Load balancing core
│   ├── ratelimiter/            # Rate limiting strategies
│   ├── web/                    # WebSocket management
│   └── types.go                # Tipos compartidos
```

### 📊 **Algoritmos de Optimización**

#### **Weight Adaptation Algorithm**
```go
newWeight = currentWeight + adaptationSpeed * (targetWeight - currentWeight)

targetWeight = baseWeight * latencyFactor * errorRateFactor

latencyFactor = max(0.1, min(5.0, targetLatency / actualLatency))
errorRateFactor = max(0.1, 1.0 - (errorRate / maxErrorRate))
```

#### **EMA Calculation**
```go
newEMA = alpha * currentValue + (1 - alpha) * previousEMA
```

### 🛡️ **Consideraciones de Seguridad**

- **Rate Limiting**: Protección contra DDoS básicos
- **Input Validation**: Validación en todos los endpoints
- **Timeout Management**: Prevención de resource exhaustion
- **Health Checks**: Detección temprana de problemas

### ⚡ **Optimización de Performance**

- **Connection Pooling**: Reutilización de conexiones HTTP
- **Goroutine Management**: Pool de workers para requests
- **Memory Management**: Buffers reutilizables
- **Async Operations**: Operaciones no bloqueantes

## 🛠️ Desarrollo

### 🔄 **Workflow de Desarrollo**

#### **1. Setup del Entorno**
```powershell
git clone https://github.com/deyvigo/balancer.git
cd balancer
go mod tidy
```

#### **2. Desarrollo Local**
```powershell
# Terminal 1: Backends
docker-compose up -d

# Terminal 2: Load Balancer
go run ./balancer/cmd/main.go

# Terminal 3: Frontend (opcional)
cd balancer-front && pnpm dev

# Terminal 4: Pruebas
.\scripts\monitor_realtime.ps1
```

#### **3. Testing**
```powershell
# Unit tests
go test ./...

# Integration tests
.\scripts\comprehensive_test.ps1

# Load testing
.\scripts\stress_test.ps1
```

### 📦 **Build y Deploy**

#### **Compilación**
```powershell
# Build local
go build -o balancer.exe ./balancer/cmd

# Build para diferentes plataformas
GOOS=linux GOARCH=amd64 go build -o balancer-linux ./balancer/cmd
GOOS=windows GOARCH=amd64 go build -o balancer.exe ./balancer/cmd
```

#### **Docker Build**
```powershell
# Construir imagen
docker build -t load-balancer .

# Ejecutar con Docker
docker run -p 8089:8089 -p 9000:9000 load-balancer
```

### 🐛 **Debugging**

#### **Logs Estructurados**
El sistema usa logs estructurados con diferentes niveles:
```
INFO: Operaciones normales
WARN: Situaciones que requieren atención
ERROR: Errores que afectan funcionalidad
DEBUG: Información detallada para desarrollo
```

#### **Health Check Debugging**
```powershell
# Verificar estado de backends
curl http://localhost:9000/api/backends

# Verificar métricas
curl http://localhost:9000/api/metrics

# Verificar circuit breakers
curl http://localhost:9000/api/circuit-breaker
```

### 🚀 **Contribución**

#### **Guidelines**
1. Fork del repositorio
2. Crear rama feature (`git checkout -b feature/nueva-caracteristica`)
3. Commit de cambios (`git commit -am 'Add nueva caracteristica'`)
4. Push a la rama (`git push origin feature/nueva-caracteristica`)
5. Crear Pull Request

#### **Estándares de Código**
- **Go**: Seguir `gofmt` y `golint`
- **JavaScript/TypeScript**: Usar ESLint + Prettier
- **Testing**: Cobertura mínima 80%
- **Documentación**: Comentarios claros en código complejo

### 📋 **Roadmap**

#### **🎯 Próximas Características**
- [ ] **Configuración dinámica**: Hot reload completo
- [ ] **Múltiples algoritmos**: Consistented hashing, geolocation-based
- [ ] **Métricas avanzadas**: Prometheus integration
- [ ] **SSL/TLS**: Terminación SSL en el load balancer
- [ ] **Service Discovery**: Consul/Etcd integration
- [ ] **Logging**: ELK stack integration

#### **🔧 Mejoras Técnicas**
- [ ] **gRPC Support**: Load balancing para gRPC
- [ ] **HTTP/2**: Soporte completo para HTTP/2
- [ ] **Kubernetes**: Helm charts y operators
- [ ] **Monitoring**: Grafana dashboards
- [ ] **Testing**: Chaos engineering tests

---

## 🎉 ¡Listo para Usar!

Este load balancer está diseñado para ser:
- **🚀 Rápido de configurar**: Un script y está funcionando
- **🔧 Fácil de extender**: Arquitectura modular y bien documentada
- **📊 Observable**: Métricas completas y dashboard interactivo
- **🛡️ Robusto**: Circuit breakers, rate limiting y health checks
- **🧪 Testeable**: Suite completa de pruebas automatizadas

**¿Preguntas o problemas?** Abre un issue en el repositorio o contribuye con mejoras.

¡Happy Load Balancing! 🎯