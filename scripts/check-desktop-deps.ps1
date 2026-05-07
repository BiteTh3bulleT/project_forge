$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
& node (Join-Path $RootDir "scripts/check-desktop-deps.mjs") @args
exit $LASTEXITCODE
