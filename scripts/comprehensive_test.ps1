# Script de Prueba Comprensiva
# Ejecuta todas las pruebas en secuencia para una evaluación completa

Write-Host "🎯 INICIANDO SUITE COMPLETA DE PRUEBAS" -ForegroundColor Magenta
Write-Host "=========================================" -ForegroundColor Magenta

$testResults = @{}
$startTime = Get-Date

# Función para registrar resultados
function Record-TestResult {
    param($TestName, $Success, $Details)
    $testResults[$TestName] = @{
        Success = $Success
        Details = $Details
        Timestamp = Get-Date
    }
}

# Verificar que el balanceador esté funcionando
Write-Host "`n🔍 VERIFICACIÓN INICIAL" -ForegroundColor Cyan
Write-Host "Verificando que el load balancer esté funcionando..." -ForegroundColor Yellow
try {
    $healthCheck = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
    Write-Host "✅ Load balancer respondiendo correctamente" -ForegroundColor Green
    Record-TestResult "HealthCheck" $true "Status: $($healthCheck.StatusCode)"
}
catch {
    Write-Host "❌ Load balancer no está funcionando: $($_.Exception.Message)" -ForegroundColor Red
    Record-TestResult "HealthCheck" $false $_.Exception.Message
    Write-Host "Abortando pruebas..." -ForegroundColor Red
    exit 1
}

# Test 1: Prueba Básica
Write-Host "`n🚀 TEST 1: PRUEBA BÁSICA" -ForegroundColor Cyan
try {
    & "$PSScriptRoot\basic_load_test.ps1"
    Record-TestResult "BasicLoad" $true "Completado"
}
catch {
    Record-TestResult "BasicLoad" $false $_.Exception.Message
    Write-Host "❌ Error en prueba básica: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n⏳ Pausa de 5 segundos entre pruebas..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Test 2: Distribución de Carga
Write-Host "`n⚖️ TEST 2: DISTRIBUCIÓN DE CARGA" -ForegroundColor Cyan
try {
    & "$PSScriptRoot\load_balancing_test.ps1"
    Record-TestResult "LoadBalancing" $true "Completado"
}
catch {
    Record-TestResult "LoadBalancing" $false $_.Exception.Message
    Write-Host "❌ Error en prueba de distribución: $($_.Exception.Message)" -ForegroundColor Red
}

Start-Sleep -Seconds 5

# Test 3: Rate Limiting
Write-Host "`n⚡ TEST 3: RATE LIMITING" -ForegroundColor Cyan
try {
    & "$PSScriptRoot\rate_limit_test.ps1"
    Record-TestResult "RateLimiting" $true "Completado"
}
catch {
    Record-TestResult "RateLimiting" $false $_.Exception.Message
    Write-Host "❌ Error en prueba de rate limiting: $($_.Exception.Message)" -ForegroundColor Red
}

Start-Sleep -Seconds 10  # Más tiempo para reset del rate limiter

# Test 4: Circuit Breaker
Write-Host "`n🛡️ TEST 4: CIRCUIT BREAKER" -ForegroundColor Cyan
try {
    & "$PSScriptRoot\circuit_breaker_test.ps1"
    Record-TestResult "CircuitBreaker" $true "Completado"
}
catch {
    Record-TestResult "CircuitBreaker" $false $_.Exception.Message
    Write-Host "❌ Error en prueba de circuit breaker: $($_.Exception.Message)" -ForegroundColor Red
}

Start-Sleep -Seconds 10  # Tiempo para recovery del circuit breaker

# Test 5: Prueba de Estrés (opcional, comentada por defecto)
# Write-Host "`n🔥 TEST 5: PRUEBA DE ESTRÉS" -ForegroundColor Cyan
# try {
#     & "$PSScriptRoot\stress_test.ps1"
#     Record-TestResult "StressTest" $true "Completado"
# }
# catch {
#     Record-TestResult "StressTest" $false $_.Exception.Message
#     Write-Host "❌ Error en prueba de estrés: $($_.Exception.Message)" -ForegroundColor Red
# }

$endTime = Get-Date
$totalTime = ($endTime - $startTime).TotalMinutes

# Resumen Final
Write-Host "`n🎯 RESUMEN FINAL DE PRUEBAS" -ForegroundColor Magenta
Write-Host "=============================" -ForegroundColor Magenta
Write-Host "Tiempo total de ejecución: $([math]::Round($totalTime, 2)) minutos" -ForegroundColor Yellow
Write-Host "`nResultados por prueba:" -ForegroundColor White

$successCount = 0
$failureCount = 0

foreach ($test in $testResults.GetEnumerator()) {
    if ($test.Value.Success) {
        Write-Host "   ✅ $($test.Key): PASSED" -ForegroundColor Green
        $successCount++
    } else {
        Write-Host "   ❌ $($test.Key): FAILED - $($test.Value.Details)" -ForegroundColor Red
        $failureCount++
    }
}

$totalTests = $successCount + $failureCount
Write-Host "`n📊 ESTADÍSTICAS FINALES:" -ForegroundColor Cyan
Write-Host "   Total de pruebas: $totalTests" -ForegroundColor White
Write-Host "   Exitosas: $successCount" -ForegroundColor Green
Write-Host "   Fallidas: $failureCount" -ForegroundColor Red
Write-Host "   Tasa de éxito: $([math]::Round(($successCount / $totalTests) * 100, 1))%" -ForegroundColor Magenta

if ($failureCount -eq 0) {
    Write-Host "`n🎉 ¡TODAS LAS PRUEBAS PASARON! Sistema funcionando correctamente." -ForegroundColor Green
} else {
    Write-Host "`n⚠️  Algunas pruebas fallaron. Revisar logs para más detalles." -ForegroundColor Yellow
}

Write-Host "`n💡 Para ver métricas en tiempo real, visita: http://localhost:5173" -ForegroundColor Cyan
