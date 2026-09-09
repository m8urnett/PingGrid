<#
.SYNOPSIS
    Builds the PingGrid CLI (pg) binary for Windows, Linux, or macOS.

.DESCRIPTION
    Compiles PingGrid with version, build date, and git commit metadata injected via ldflags.
    Supports native Windows builds and cross-compilation for Linux and macOS.

.PARAMETER Target
    The target OS/platform to build for: "windows" (default), "linux", "darwin", or "all".

.EXAMPLE
    .\build.ps1
    Builds pg.exe for Windows.

.EXAMPLE
    .\build.ps1 -Target linux
    Cross-compiles pure Go binary bin/pg-linux-amd64 with CGO disabled.

.EXAMPLE
    .\build.ps1 -Target all
    Builds Windows, Linux (amd64), and macOS (arm64 + amd64) binaries.
#>
[CmdletBinding()]
param(
    [ValidateSet('windows', 'linux', 'darwin', 'all')]
    [string]$Target = 'windows'
)

$ErrorActionPreference = 'Stop'

$Commit = $(git rev-parse --short HEAD 2>$null; if (-not $?) { 'dev' })
$Date = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$LdFlags = "-s -w -X github.com/m8urnett/PingGrid/internal/version.GitCommit=$Commit -X github.com/m8urnett/PingGrid/internal/version.BuildDate=$Date"

function Build-Target {
    param(
        [string]$OsName,
        [string]$Arch,
        [string]$OutputFile,
        [string]$Cgo = "1"
    )

    Write-Host "Building for $OsName/$Arch -> $OutputFile..." -ForegroundColor Cyan

    $outDir = Split-Path $OutputFile -Parent
    if ($outDir -and -not (Test-Path $outDir)) {
        New-Item -ItemType Directory -Path $outDir -Force | Out-Null
    }

    $env:GOOS = $OsName
    $env:GOARCH = $Arch
    $env:CGO_ENABLED = $Cgo

    try {
        go build -trimpath -ldflags $LdFlags -o $OutputFile .
        if ($LASTEXITCODE -ne 0) {
            throw "Compilation failed for $OsName/$Arch with exit code $LASTEXITCODE"
        }
        Write-Host "  -> Successfully built: $OutputFile" -ForegroundColor Green
    }
    finally {
        # Restore environment
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
    }
}

switch ($Target) {
    'windows' {
        Build-Target -OsName 'windows' -Arch 'amd64' -OutputFile 'pg.exe' -Cgo '1'
    }
    'linux' {
        Build-Target -OsName 'linux' -Arch 'amd64' -OutputFile 'bin/pg-linux-amd64' -Cgo '0'
    }
    'darwin' {
        Build-Target -OsName 'darwin' -Arch 'arm64' -OutputFile 'bin/pg-darwin-arm64' -Cgo '0'
        Build-Target -OsName 'darwin' -Arch 'amd64' -OutputFile 'bin/pg-darwin-amd64' -Cgo '0'
    }
    'all' {
        Build-Target -OsName 'windows' -Arch 'amd64' -OutputFile 'pg.exe' -Cgo '1'
        Build-Target -OsName 'linux' -Arch 'amd64' -OutputFile 'bin/pg-linux-amd64' -Cgo '0'
        Build-Target -OsName 'darwin' -Arch 'arm64' -OutputFile 'bin/pg-darwin-arm64' -Cgo '0'
        Build-Target -OsName 'darwin' -Arch 'amd64' -OutputFile 'bin/pg-darwin-amd64' -Cgo '0'
    }
}
