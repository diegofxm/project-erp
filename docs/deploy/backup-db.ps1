# Backup manual/programado de la base de datos real (alwaysdata.net) hacia esta máquina --
# deliberadamente separado del VPS: si el VPS se ve comprometido, el atacante no tiene acceso
# directo a las copias (defensa en profundidad, decisión explícita del responsable del proyecto).
#
# alwaysdata SÍ ofrece backups desde su panel, pero son MANUALES (el usuario tiene que entrar y
# generarlos a mano) -- no hay backup automático real hoy. Este script es el mecanismo automático
# que falta, corriendo desde donde tú decidas (esta máquina, programado con el Programador de
# tareas de Windows -- ver instrucciones abajo).
#
# Formato: -Fc (custom, comprimido) -- se restaura con pg_restore, no con psql/`\i`. Permite
# restaurar tablas individuales sin tener que aplicar el dump completo.
#
# Requisito: la variable de entorno PROD_DATABASE_URL con la cadena de conexión REAL de
# producción (postgresql://usuario:password@host:puerto/basededatos). A propósito NO va
# hardcodeada aquí ni en ningún archivo versionado -- configúrala como variable de entorno de
# usuario en Windows (no de sistema, no la compartas), o pásala inline al invocar el script:
#   $env:PROD_DATABASE_URL = "postgresql://..."; pwsh docs/deploy/backup-db.ps1
#
# Uso manual: desde la raíz del repo, `pwsh docs/deploy/backup-db.ps1`
# Uso programado: ver docs/deploy/README-backup.md para registrarlo en el Programador de tareas.

param(
    # Carpeta donde quedan los .dump -- fuera del repo a propósito (no se versiona un backup).
    [string]$BackupDir = (Join-Path $env:USERPROFILE "erp-backups"),
    # Cuántos días de backups conservar -- los más viejos se borran automáticamente para no
    # crecer sin límite. 30 días cubre restaurar "el mes pasado" sin acumular años de dumps.
    [int]$RetentionDays = 30
)

$ErrorActionPreference = "Stop"

if (-not $env:PROD_DATABASE_URL) {
    Write-Error "Falta la variable de entorno PROD_DATABASE_URL con la cadena de conexión real de producción. Ver el encabezado de este script."
    exit 1
}

$pgDump = "C:\Program Files\PostgreSQL\18\bin\pg_dump.exe"
if (-not (Test-Path $pgDump)) {
    Write-Error "No se encontró pg_dump.exe en $pgDump -- ajusta la ruta si tu instalación de PostgreSQL client tools está en otro lado."
    exit 1
}

if (-not (Test-Path $BackupDir)) {
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
}

$timestamp = Get-Date -Format "yyyy-MM-dd_HHmmss"
$outFile = Join-Path $BackupDir "erp_prod_$timestamp.dump"

Write-Host "Generando backup -> $outFile"
& $pgDump --format=custom --file="$outFile" --dbname="$env:PROD_DATABASE_URL"
if ($LASTEXITCODE -ne 0) {
    Write-Error "pg_dump terminó con código $LASTEXITCODE -- revisa la conexión/credenciales."
    exit $LASTEXITCODE
}

$sizeKB = [math]::Round((Get-Item $outFile).Length / 1KB, 1)
Write-Host "Backup completado: $outFile ($sizeKB KB)"

# Retención -- borra dumps más viejos que $RetentionDays.
$cutoff = (Get-Date).AddDays(-$RetentionDays)
$old = Get-ChildItem -Path $BackupDir -Filter "erp_prod_*.dump" | Where-Object { $_.LastWriteTime -lt $cutoff }
foreach ($f in $old) {
    Write-Host "Eliminando backup vencido ($RetentionDays+ días): $($f.Name)"
    Remove-Item $f.FullName -Force
}

Write-Host "Backups vigentes en $BackupDir : $((Get-ChildItem -Path $BackupDir -Filter 'erp_prod_*.dump').Count)"
