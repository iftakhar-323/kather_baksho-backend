# Full demo dataset for KatherBox. Idempotent - safe to re-run.
# Usage:  ./seed.ps1  [-Products 120]  [-Users 60]
param(
    [int]$Products = 120,
    [int]$Users = 60
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "==> makeadmin" -ForegroundColor Cyan
go run ./cmd/makeadmin/

Write-Host "==> seedproducts ($Products)" -ForegroundColor Cyan
go run ./cmd/seedproducts/ $Products

Write-Host "==> seedusers ($Users)" -ForegroundColor Cyan
go run ./cmd/seedusers/ $Users

Write-Host "==> seeddummy (orders, reviews, returns, subscriptions, addresses, ...)" -ForegroundColor Cyan
go run ./cmd/seeddummy/

Write-Host "==> seedorders (extra orders for customer@test.com)" -ForegroundColor Cyan
try { go run ./cmd/seedorders/ } catch { Write-Host "  (skipped: $_)" -ForegroundColor Yellow }

Write-Host ""
Write-Host "Done. Admin: admin@katherbox.com / Admin@12345" -ForegroundColor Green
Write-Host "      Seeded users: <name>@katherbox.test / Test@12345" -ForegroundColor Green
