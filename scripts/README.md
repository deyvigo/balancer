# Scripts de Pruebas de Carga - Load Balancer

## 📋 Descripción

Esta carpeta contiene scripts de PowerShell para probar diferentes aspectos del sistema de load balancer. Cada script está diseñado para evaluar características específicas del balanceador.

## 🚀 Scripts Disponibles

### 1. `basic_load_test.ps1` - Prueba Básica
- **Propósito**: Prueba básica de funcionamiento
- **Requests**: 50 secuenciales con 0.5s de intervalo
- **Evalúa**: Conectividad básica y tiempos de respuesta

```powershell
.\basic_load_test.ps1
```

### 2. `stress_test.ps1` - Prueba de Estrés
- **Propósito**: Evaluar el rendimiento bajo carga concurrente
- **Requests**: 100 concurrentes simultáneos
- **Evalúa**: Capacidad de manejo de carga alta

```powershell
.\stress_test.ps1
```

### 3. `rate_limit_test.ps1` - Prueba de Rate Limiting
- **Propósito**: Verificar el funcionamiento del rate limiter
- **Requests**: 30 rápidos + 10 con delay
- **Evalúa**: Activación y recuperación del rate limiting

```powershell
.\rate_limit_test.ps1
```

### 4. `circuit_breaker_test.ps1` - Prueba de Circuit Breaker
- **Propósito**: Probar la activación de circuit breakers
- **Simula**: Fallos 404 para activar protecciones
- **Evalúa**: Detección de fallos y recuperación automática

```powershell
.\circuit_breaker_test.ps1
```

### 5. `load_balancing_test.ps1` - Prueba de Distribución
- **Propósito**: Analizar distribución de carga entre backends
- **Requests**: 60 para análisis estadístico
- **Evalúa**: Balanceo de carga y algoritmos de distribución

```powershell
.\load_balancing_test.ps1
```

### 6. `comprehensive_test.ps1` - Suite Completa
- **Propósito**: Ejecuta todas las pruebas en secuencia
- **Incluye**: Todos los tests anteriores con pausas
- **Evalúa**: Funcionamiento integral del sistema

```powershell
.\comprehensive_test.ps1
```

## ⚙️ Prerequisitos

1. **Load Balancer ejecutándose**:
   ```powershell
   # En el directorio raíz del proyecto
   go run balancer/cmd/main.go
   ```

2. **Backends disponibles**:
   ```powershell
   docker-compose up -d
   ```

3. **Frontend (opcional para visualización)**:
   ```powershell
   cd balancer-front
   pnpm dev
   ```

## 📊 Interpretación de Resultados

### ✅ Indicadores de Éxito
- **Requests exitosos**: Status 200 OK
- **Balance adecuado**: Coeficiente de variación < 20%
- **Rate limiting**: Activación y recuperación correcta
- **Circuit breaker**: Detección de fallos y recuperación

### ⚠️ Indicadores de Problemas
- **Errores de conexión**: Backends no disponibles
- **Timeouts**: Sobrecarga del sistema
- **Rate limiting no funciona**: Configuración incorrecta
- **Balance desigual**: Problemas en algoritmo de distribución

## 🔧 Configuración de Puertos

- **Load Balancer (Proxy)**: http://localhost:8089
- **Admin API**: http://localhost:9000
- **Frontend Dashboard**: http://localhost:5173
- **Backends**: http://localhost:8080-8082

## 📝 Personalización

Puedes modificar los scripts para:
- Cambiar número de requests
- Ajustar intervalos de tiempo
- Modificar endpoints de prueba
- Agregar nuevos tipos de tests

## 🎯 Casos de Uso

### Desarrollo
```powershell
# Prueba rápida durante desarrollo
.\basic_load_test.ps1
```

### Testing
```powershell
# Suite completa antes de deploy
.\comprehensive_test.ps1
```

### Debugging
```powershell
# Probar característica específica
.\circuit_breaker_test.ps1
```

### Performance
```powershell
# Evaluar rendimiento
.\stress_test.ps1
.\load_balancing_test.ps1
```

## 🚨 Notas Importantes

- Los scripts incluyen pausas entre pruebas para evitar interferencias
- La prueba de estrés está comentada en el script comprensivo por defecto
- Asegúrate de que todos los servicios estén ejecutándose antes de las pruebas
- Los resultados se muestran en tiempo real con códigos de color

¡Usa estos scripts para validar que tu load balancer funciona correctamente! 🎉
