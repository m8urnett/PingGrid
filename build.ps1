<#
.SYNOPSIS
    Builds the PingGrid CLI (pg) binary for Windows, Linux, or macOS.

.DESCRIPTION
    Compiles PingGrid with version, build date, and git commit metadata injected via ldflags.
    Supports native Windows builds and cross-compilation for Linux and macOS.

.PARAMETER Target
    The target OS/platform to build for: "windows" (default), "linux", "darwin", or "all".

.PARAMETER Verify
    Runs module hygiene, cryptographic verification, race-enabled tests, linting,
    PowerShell analysis, and vulnerability scanning. Missing required tools fail verification.

.EXAMPLE
    .\build.ps1
    Builds bin/pg.exe for Windows.

.EXAMPLE
    .\build.ps1 -Verify
    Checks module tidiness, then runs download, checksum verification, tests, linting,
    PowerShell analysis, and govulncheck without building an application binary.

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
    [string]$Target = 'windows',

    [switch]$Verify,
    [switch]$PackageSource
)

$ErrorActionPreference = 'Stop'

if ($Verify) {
    Write-Host "Running PingGrid Verification & Supply-Chain Pipeline..." -ForegroundColor Cyan

    Write-Host "`n[1/8] Checking module tidiness (go mod tidy -diff)..." -ForegroundColor Yellow
    go mod tidy -diff
    if ($LASTEXITCODE -ne 0) { throw "go.mod/go.sum are not tidy (exit code $LASTEXITCODE)" }

    Write-Host "`n[2/8] Downloading modules (go mod download)..." -ForegroundColor Yellow
    go mod download
    if ($LASTEXITCODE -ne 0) { throw "go mod download failed with exit code $LASTEXITCODE" }

    Write-Host "`n[3/8] Verifying module checksums (go mod verify)..." -ForegroundColor Yellow
    go mod verify
    if ($LASTEXITCODE -ne 0) { throw "go mod verify failed with exit code $LASTEXITCODE" }

    Write-Host "`n[4/8] Running unit test suite (go test ./...)..." -ForegroundColor Yellow
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }

    Write-Host "`n[5/8] Running race detector (go test -race ./...)..." -ForegroundColor Yellow
    go test -race ./...
    if ($LASTEXITCODE -ne 0) { throw "go test -race failed with exit code $LASTEXITCODE" }

    Write-Host "`n[6/8] Running linter (golangci-lint run)..." -ForegroundColor Yellow
    if (Get-Command golangci-lint -ErrorAction SilentlyContinue) {
        golangci-lint run
        if ($LASTEXITCODE -ne 0) { throw "golangci-lint failed with exit code $LASTEXITCODE" }
    } else {
        throw "golangci-lint is required for verification but was not found in PATH"
    }

    Write-Host "`n[7/8] Running PowerShell static analysis..." -ForegroundColor Yellow
    if (Get-Command Invoke-ScriptAnalyzer -ErrorAction SilentlyContinue) {
        $Analysis = Invoke-ScriptAnalyzer -Path $PSCommandPath -Severity Warning,Error
        if ($Analysis) {
            $Analysis | Format-Table -AutoSize | Out-String | Write-Host
            throw "PSScriptAnalyzer reported $($Analysis.Count) warning/error finding(s)"
        }
    } else {
        throw "PSScriptAnalyzer is required for verification but Invoke-ScriptAnalyzer was not found"
    }

    Write-Host "`n[8/8] Scanning known vulnerabilities (govulncheck ./...)..." -ForegroundColor Yellow
    if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
        govulncheck ./...
        if ($LASTEXITCODE -ne 0) { throw "govulncheck failed with exit code $LASTEXITCODE" }
    } else {
        throw "govulncheck is required for verification but was not found in PATH"
    }

    Write-Host "`nVerification checks completed!" -ForegroundColor Green
    return
}

$MainSource = Get-Content -LiteralPath 'main.go' -Raw
$VersionMatch = [regex]::Match($MainSource, 'version\s*=\s*"(1\.[0-9]{2}\.[0-9]{3})"')
if (-not $VersionMatch.Success) {
    throw 'VERSION_INVALID: main.go must declare const version using 1.xx.NNN'
}
$ApplicationVersion = $VersionMatch.Groups[1].Value
$ResourceSource = Get-Content -LiteralPath 'pg_windows_amd64.rc' -Raw
if (-not $ResourceSource.Contains("VALUE `"FileVersion`", `"$ApplicationVersion\0`"")) {
    throw "VERSION_MISMATCH: pg_windows_amd64.rc does not match main.go version $ApplicationVersion"
}
$VersionParts = $ApplicationVersion.Split('.')
$NumericVersion = "1,$([int]$VersionParts[1]),$([int]$VersionParts[2]),0"
if (-not $ResourceSource.Contains("FILEVERSION $NumericVersion") -or
    -not $ResourceSource.Contains("PRODUCTVERSION $NumericVersion")) {
    throw "VERSION_MISMATCH: numeric Windows resource version does not match main.go version $ApplicationVersion"
}

$Commit = $(git rev-parse --short HEAD 2>$null; if (-not $?) { 'dev' })
$GitStatus = git status --porcelain 2>$null
if ($LASTEXITCODE -eq 0 -and $GitStatus) {
    $Commit = "$Commit-dirty"
}
$CommitDate = git show -s --format=%cI HEAD 2>$null
if ($LASTEXITCODE -eq 0 -and $CommitDate) {
    $Date = ([DateTimeOffset]::Parse(($CommitDate | Select-Object -First 1))).UtcDateTime.ToString('yyyy-MM-ddTHH:mm:ssZ')
} else {
    $Date = 'unknown'
}
$LdFlags = "-s -w -X github.com/m8urnett/PingGrid/internal/version.GitCommit=$Commit -X github.com/m8urnett/PingGrid/internal/version.BuildDate=$Date"

function Build-Target {
    param(
        [string]$OsName,
        [string]$Arch,
        [string]$OutputFile,
        [string]$Cgo = "1"
    )

    Write-Host "Building for $OsName/$Arch -> $OutputFile..." -ForegroundColor Cyan

    if ($OsName -eq 'windows' -and $Arch -eq 'amd64') {
        $Windres = Get-Command windres -ErrorAction SilentlyContinue
        if (-not $Windres) {
            throw "windres is required to generate mandatory Windows version resources"
        }
        & $Windres.Source 'pg_windows_amd64.rc' '-O' 'coff' '-o' 'resource_windows_amd64.syso'
        if ($LASTEXITCODE -ne 0) {
            throw "Windows resource generation failed with exit code $LASTEXITCODE"
        }
    }

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

function New-SourceArchive {
    param(
        [string]$VersionString
    )
    $OutDir = 'bin'
    if (-not (Test-Path $OutDir)) {
        New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
    }
    $ZipPath = Join-Path $OutDir "PingGrid-$VersionString-source.zip"
    Write-Host "Creating source code archive -> $ZipPath..." -ForegroundColor Cyan
    git archive --format=zip --prefix="PingGrid-$VersionString/" -o $ZipPath HEAD
    if ($LASTEXITCODE -ne 0) {
        throw "git archive failed with exit code $LASTEXITCODE"
    }
    Write-Host "  -> Successfully archived: $ZipPath" -ForegroundColor Green
}

switch ($Target) {
    'windows' {
        Build-Target -OsName 'windows' -Arch 'amd64' -OutputFile 'bin/pg.exe' -Cgo '1'
    }
    'linux' {
        Build-Target -OsName 'linux' -Arch 'amd64' -OutputFile 'bin/pg-linux-amd64' -Cgo '0'
    }
    'darwin' {
        Build-Target -OsName 'darwin' -Arch 'arm64' -OutputFile 'bin/pg-darwin-arm64' -Cgo '0'
        Build-Target -OsName 'darwin' -Arch 'amd64' -OutputFile 'bin/pg-darwin-amd64' -Cgo '0'
    }
    'all' {
        Build-Target -OsName 'windows' -Arch 'amd64' -OutputFile 'bin/pg.exe' -Cgo '1'
        Build-Target -OsName 'linux' -Arch 'amd64' -OutputFile 'bin/pg-linux-amd64' -Cgo '0'
        Build-Target -OsName 'darwin' -Arch 'arm64' -OutputFile 'bin/pg-darwin-arm64' -Cgo '0'
        Build-Target -OsName 'darwin' -Arch 'amd64' -OutputFile 'bin/pg-darwin-amd64' -Cgo '0'
        New-SourceArchive -VersionString $ApplicationVersion
    }
}

if ($PackageSource -and $Target -ne 'all') {
    New-SourceArchive -VersionString $ApplicationVersion
}
