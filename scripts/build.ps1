param([switch]$SkipTests)
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
if (!$SkipTests) {
    go test ./internal/core ./internal/mcpserver
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
}
& '.tools/wails.exe' build -platform windows/amd64 -o EZ软图-build.exe -trimpath -webview2 error
if ($LASTEXITCODE -ne 0) { throw 'Desktop build failed' }
if (!$SkipTests) {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Desktop and core regression tests failed' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go static checks failed' }
}
$releaseVersion = (Get-Content -LiteralPath 'wails.json' -Raw | ConvertFrom-Json).info.productVersion
if ($releaseVersion -notmatch '^\d+\.\d+\.\d+$') { throw 'Invalid release version' }
$releaseDir = Join-Path 'dist' ('EZ软图-' + $releaseVersion)
$releaseArchive = Join-Path 'dist' ('EZ软图-' + $releaseVersion + '-windows-amd64.zip')
New-Item -ItemType Directory -Force $releaseDir | Out-Null
$desktopExe = Join-Path $releaseDir 'EZ软图.exe'
Copy-Item -LiteralPath 'build/bin/EZ软图-build.exe' -Destination $desktopExe -Force
# Remove only obsolete bundled files, preserving any user-created data beside the EXE.
foreach ($obsoleteName in @('tuxu-mcp.exe', 'diagram.schema.json', '使用说明.md', '图书借阅系统.tuxu.json')) {
    $obsoletePath = Join-Path $releaseDir $obsoleteName
    if (Test-Path -LiteralPath $obsoletePath -PathType Leaf) { Remove-Item -LiteralPath $obsoletePath -Force }
}
# Package explicit distributable files; never include the project's data directory.
Compress-Archive -LiteralPath $desktopExe -DestinationPath $releaseArchive -Force
Get-Item -LiteralPath $desktopExe, $releaseArchive | Select-Object Name, Length
