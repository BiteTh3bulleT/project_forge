$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
& node (Join-Path $RootDir "scripts/check-vsa-files.mjs") @args
exit $LASTEXITCODE
