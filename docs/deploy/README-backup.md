# Backup de la base de datos de producción

## Contexto

La base de datos real corre en **alwaysdata.net**, no en el VPS. alwaysdata ofrece backups desde
su panel, pero son **manuales** — hay que entrar y generarlos a mano, no hay una copia automática
programada por ellos. `docs/deploy/backup-db.ps1` es el mecanismo automático que falta.

Se guarda **fuera del VPS a propósito**: si el VPS se ve comprometido, un atacante con acceso a
él no tiene acceso directo a las copias de seguridad — es una decisión explícita de defensa en
profundidad, no un descuido de no automatizarlo ahí mismo.

## Configuración inicial (una sola vez)

1. Necesitas `pg_dump.exe`/`pg_restore.exe` — ya vienen con la instalación de PostgreSQL 18 en
   `C:\Program Files\PostgreSQL\18\bin\`. Si tu instalación está en otra ruta, ajusta `$pgDump`
   dentro de `backup-db.ps1`.
2. Configura la cadena de conexión real de producción como **variable de entorno de usuario**
   (no de sistema, no la guardes en ningún archivo del repo):

   ```powershell
   [Environment]::SetEnvironmentVariable("PROD_DATABASE_URL", "postgresql://usuario:password@host-alwaysdata:5432/basededatos", "User")
   ```

   Cierra y vuelve a abrir la terminal/sesión para que la variable quede disponible.

## Uso manual

```powershell
pwsh docs/deploy/backup-db.ps1
```

Por defecto guarda en `%USERPROFILE%\erp-backups\` y conserva 30 días de backups (los más viejos
se borran solos). Ambos son parámetros:

```powershell
pwsh docs/deploy/backup-db.ps1 -BackupDir "D:\backups\erp" -RetentionDays 60
```

## Programarlo (Programador de tareas de Windows)

Para que corra solo, sin depender de acordarte:

```powershell
$action = New-ScheduledTaskAction -Execute "pwsh.exe" -Argument "-File `"C:\Users\codev\Development\Go\project-ubl\docs\deploy\backup-db.ps1`""
$trigger = New-ScheduledTaskTrigger -Daily -At 3am
Register-ScheduledTask -TaskName "ERP-Backup-DB" -Action $action -Trigger $trigger -Description "Backup diario de la base de datos de producción (alwaysdata.net)"
```

La tarea corre con tu sesión de usuario, así que hereda la variable de entorno `PROD_DATABASE_URL`
que configuraste arriba. Verificar que quedó registrada: `Get-ScheduledTask -TaskName "ERP-Backup-DB"`.

## Restaurar un backup

Los `.dump` son formato `custom` de `pg_dump` (comprimidos, permiten restaurar objetos
individuales) — se restauran con `pg_restore`, no con `psql`/`\i`:

```powershell
# Restaurar TODO sobre una base nueva/vacía:
& "C:\Program Files\PostgreSQL\18\bin\pg_restore.exe" --dbname="postgresql://usuario:password@host:5432/base_destino" --clean --if-exists "C:\ruta\al\erp_prod_2026-08-10_193013.dump"

# Ver el contenido de un dump sin restaurar nada (para confirmar que no está corrupto):
& "C:\Program Files\PostgreSQL\18\bin\pg_restore.exe" --list "C:\ruta\al\erp_prod_2026-08-10_193013.dump"

# Restaurar solo un esquema específico (ej. solo "electronic", para recuperar un módulo puntual):
& "C:\Program Files\PostgreSQL\18\bin\pg_restore.exe" --dbname="postgresql://..." --schema=electronic "C:\ruta\al\....dump"
```

`--clean --if-exists` hace que `pg_restore` borre los objetos existentes antes de recrearlos —
úsalo cuando restaures sobre una base que ya tiene el esquema viejo, para evitar conflictos de
"ya existe". Sin esa bandera, restaura solo lo que falte (útil para restaurar sobre una base
recién creada y vacía).

## Verificación hecha en esta sesión

El script se probó de punta a punta contra la base de datos local de desarrollo (como sustituto
seguro — nunca se ejecutó contra la base real de producción desde este entorno, para no arriesgar
una conexión no planeada a datos reales). Resultado: `pg_dump` generó un archivo `.dump` válido
(297 KB, formato custom, gzip), y `pg_restore --list` lo leyó correctamente mostrando las 418
entradas del TOC con los 14 esquemas del proyecto — confirma que el mecanismo funciona end to end.
Lo que queda pendiente, y depende exclusivamente del usuario: configurar `PROD_DATABASE_URL` con
la cadena real de alwaysdata.net y decidir si registra la tarea programada o lo corre manualmente
por ahora.
