# Script de Prueba de Estrés
# Envía múltiples requests concurrentes para probar límites del sistema

Write-Host "🔥 Iniciando prueba de estrés..." -ForegroundColor Red
Write-Host "Enviando 100 requests concurrentes" -ForegroundColor Yellow

$jobs = @()
$startTime = Get-Date

# Crear trabajos concurrentes
for ($i = 1; $i -le 100; $i++) {
    $job = Start-Job -ScriptBlock {
        param($requestId)
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
            return @{
                Id = $requestId
                StatusCode = $response.StatusCode
                Success = $true
                Error = $null
            }
        }
        catch {
            return @{
                Id = $requestId
                StatusCode = 0
                Success = $false
                Error = $_.Exception.Message
            }
        }
    } -ArgumentList $i
    
    $jobs += $job
    Write-Host "🚀 Lanzando request $i" -ForegroundColor Gray
}

Write-Host "`n⏳ Esperando completación de todos los jobs..." -ForegroundColor Yellow

# Esperar y recopilar resultados
$results = @()
foreach ($job in $jobs) {
    $result = Receive-Job -Job $job -Wait
    $results += $result
    Remove-Job -Job $job
}

$endTime = Get-Date
$totalTime = ($endTime - $startTime).TotalSeconds

$successCount = ($results | Where-Object { $_.Success -eq $true }).Count
$errorCount = ($results | Where-Object { $_.Success -eq $false }).Count

Write-Host "`n🔥 Resultados de la Prueba de Estrés:" -ForegroundColor Cyan
Write-Host "   ✅ Exitosos: $successCount" -ForegroundColor Green
Write-Host "   ❌ Errores: $errorCount" -ForegroundColor Red
Write-Host "   ⏱️  Tiempo total: $($totalTime.ToString('F2')) segundos" -ForegroundColor Yellow
Write-Host "   📈 Throughput: $((100 / $totalTime).ToString('F2')) req/s" -ForegroundColor Magenta

if ($errorCount -gt 0) {
    Write-Host "`n❌ Errores detectados:" -ForegroundColor Red
    $errorResults = $results | Where-Object { $_.Success -eq $false }
    foreach ($error in $errorResults) {
        Write-Host "   Request $($error.Id): $($error.Error)" -ForegroundColor Red
    }
}
