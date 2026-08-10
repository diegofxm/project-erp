# Compila el binario de erp/cmd/server para subir manualmente al selector de archivo de
# "dpanel" (el panel de despliegue de la VPS -- ver docs/deploy/app-api-cofacture.service).
#
# El unit de systemd espera el binario con el nombre exacto "bin"
# (ExecStart=/srv/dpanel/apps/api-cofacture/current/bin), así que el archivo final se llama
# "bin" a propósito -- no le cambies el nombre al subirlo.
#
# Asume VPS linux/amd64 (lo más común); si tu VPS es ARM, cambia GOARCH a "arm64" abajo.
#
# Uso: desde la raíz del repo, `pwsh docs/deploy/build-release.ps1`

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$erpDir = Join-Path $repoRoot "erp"
$outFile = Join-Path $repoRoot "bin"

Push-Location $erpDir
try {
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    go build -o $outFile ./cmd/server
} finally {
    Remove-Item Env:\GOOS, Env:\GOARCH, Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}

Write-Host "Listo: $outFile -- súbelo desde el selector de archivo de dpanel."
