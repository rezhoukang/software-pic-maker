$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
if (!(Test-Path '.tools/wails.exe')) {
    New-Item -ItemType Directory -Force '.tools' | Out-Null
    $previousBin = $env:GOBIN
    try {
        $env:GOBIN = Join-Path (Get-Location) '.tools'
        go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
        if ($LASTEXITCODE -ne 0) { throw 'Wails CLI installation failed' }
    } finally { $env:GOBIN = $previousBin }
}
& '.tools/wails.exe' dev
