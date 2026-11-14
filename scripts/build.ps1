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
    go test ./internal/core ./cmd/mcp
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
$mcpExe = Join-Path $releaseDir 'tuxu-mcp.exe'
Copy-Item -LiteralPath 'build/bin/EZ软图-build.exe' -Destination $desktopExe -Force
go build -trimpath -ldflags '-s -w' -o $mcpExe ./cmd/mcp
if ($LASTEXITCODE -ne 0) { throw 'MCP build failed' }
Copy-Item -LiteralPath 'docs/使用说明.md' -Destination (Join-Path $releaseDir '使用说明.md') -Force
Copy-Item -LiteralPath 'shared/diagram.schema.json' -Destination (Join-Path $releaseDir 'diagram.schema.json') -Force
$library = Get-Content -LiteralPath 'shared/library.json' -Raw -Encoding utf8 | ConvertFrom-Json
$example = @{ schema_version = '1.0'; name = '图书借阅系统'; diagrams = @($library) } | ConvertTo-Json -Depth 50
$examplePath = Join-Path $releaseDir '图书借阅系统.tuxu.json'
[IO.File]::WriteAllText((Join-Path (Get-Location) $examplePath), $example, [Text.UTF8Encoding]::new($false))
$archiveFiles = @($desktopExe, $mcpExe, (Join-Path $releaseDir '使用说明.md'), (Join-Path $releaseDir 'diagram.schema.json'), $examplePath)
# Package explicit distributable files; never include the project's data directory.
Compress-Archive -LiteralPath $archiveFiles -DestinationPath $releaseArchive -Force
Get-Item -LiteralPath $desktopExe, $mcpExe, $releaseArchive | Select-Object Name, Length
