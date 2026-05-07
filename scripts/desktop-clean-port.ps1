$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
& node (Join-Path $RootDir "scripts/desktop-clean-port.mjs") @args
exit $LASTEXITCODE
