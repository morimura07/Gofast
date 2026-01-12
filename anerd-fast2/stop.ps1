# GoFast Project Stop Script
# Stops all running containers for the project

$ErrorActionPreference = "Stop"

Write-Host "`nStopping GoFast services..." -ForegroundColor Cyan

# Stop and remove containers
docker compose -f docker-compose.yml -f docker-compose.client.yml down

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ All services stopped successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Error stopping services" -ForegroundColor Red
    exit 1
}
