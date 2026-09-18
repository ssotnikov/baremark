[CmdletBinding()]
param(
    [ValidateSet("all", "amd64", "arm64")]
    [string]$Architecture = "all",

    [ValidateSet("release", "debug")]
    [string]$Configuration = "release",

    [switch]$Vulncheck
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = $PSScriptRoot
$resourceDirectory = Join-Path $repoRoot "cmd\baremark"
$toolModule = Join-Path $repoRoot "build\tools"
$distDirectory = Join-Path $repoRoot "dist"

function Invoke-Go {
    param([Parameter(Mandatory)][string[]]$Arguments)

    & go @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "go $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

function Invoke-Checks {
    Invoke-Go -Arguments @("-C", $toolModule, "test", "./...")
    Invoke-Go -Arguments @("-C", $toolModule, "vet", "-unsafeptr=false", "./...")
    Invoke-Go -Arguments @("test", "./...")
    Invoke-Go -Arguments @("vet", "-unsafeptr=false", "./...")
    Write-Host "Passed go vet (-unsafeptr=false)"

    $oldGOOS = $env:GOOS
    try {
        $env:GOOS = "windows"
        Invoke-Go -Arguments @("vet", "-unsafeptr=false", "./...")
        Write-Host "Passed go vet (GOOS=windows, -unsafeptr=false)"
    }
    finally {
        $env:GOOS = $oldGOOS
    }
}

function Invoke-VulnerabilityCheck {
    $command = Get-Command govulncheck -ErrorAction SilentlyContinue
    if ($null -ne $command) {
        $scanner = $command.Source
    }
    else {
        $goBin = (& go env GOBIN).Trim()
        if ([string]::IsNullOrWhiteSpace($goBin)) {
            $goPath = (& go env GOPATH).Split([IO.Path]::PathSeparator)[0]
            $goBin = Join-Path $goPath "bin"
        }
        $scanner = Join-Path $goBin "govulncheck.exe"
        if (-not (Test-Path -LiteralPath $scanner -PathType Leaf)) {
            throw "govulncheck not found; install it with: go install golang.org/x/vuln/cmd/govulncheck@latest"
        }
    }
    & $scanner ./...
    if ($LASTEXITCODE -ne 0) {
        throw "govulncheck ./... failed with exit code $LASTEXITCODE"
    }
    Write-Host "Passed govulncheck ./..."
}

function New-WindowsResource {
    param(
        [Parameter(Mandatory)][string]$TargetArchitecture,
        [Parameter(Mandatory)][string]$Version,
        [Parameter(Mandatory)][string]$OriginalFilename
    )

    $resourceFile = Join-Path $resourceDirectory "resource_windows_$TargetArchitecture.syso"
    Remove-Item -LiteralPath $resourceFile -Force -ErrorAction SilentlyContinue

    $arguments = @(
        "-C", $toolModule,
        "run", "-mod=readonly", "./cmd/resourcegen",
        "-root", $repoRoot,
        "-arch", $TargetArchitecture,
        "-version", $Version,
        "-original-name", $OriginalFilename,
        "-o", $resourceFile
    )
    Invoke-Go -Arguments $arguments

    if (-not (Test-Path -LiteralPath $resourceFile -PathType Leaf)) {
        throw "resource generator did not create $resourceFile"
    }
    Invoke-Go -Arguments @("run", "./cmd/artifactcheck", "-kind", "syso", "-arch", $TargetArchitecture, "-path", $resourceFile) | Out-Host
    return $resourceFile
}

function Build-Architecture {
    param(
        [Parameter(Mandatory)][string]$TargetArchitecture,
        [Parameter(Mandatory)][string]$Version,
        [Parameter(Mandatory)][string]$BuildConfiguration
    )

    $suffix = if ($BuildConfiguration -eq "debug") { "-debug" } else { "" }
    $outputName = "baremark-$Version-$TargetArchitecture$suffix.exe"
    $resourceFile = New-WindowsResource -TargetArchitecture $TargetArchitecture -Version $Version -OriginalFilename $outputName
    $outputFile = Join-Path $distDirectory $outputName
    $temporaryOutputFile = Join-Path $distDirectory ".$outputName.$PID.tmp"
    Remove-Item -LiteralPath $temporaryOutputFile -Force -ErrorAction SilentlyContinue

    $oldGOOS = $env:GOOS
    $oldGOARCH = $env:GOARCH
    $oldCGOEnabled = $env:CGO_ENABLED
    try {
        try {
            $env:GOOS = "windows"
            $env:GOARCH = $TargetArchitecture
            $env:CGO_ENABLED = "0"
            $buildArguments = @(
                "build",
                "-trimpath",
                "-buildvcs=true"
            )
            if ($BuildConfiguration -eq "release") {
                $buildArguments += "-ldflags=-H=windowsgui -s -w"
            }
            else {
                $buildArguments += @("-gcflags=all=-N -l", "-ldflags=-H=windowsgui")
            }
            $buildArguments += @("-o", $temporaryOutputFile, "./cmd/baremark")
            Invoke-Go -Arguments $buildArguments
        }
        finally {
            $env:GOOS = $oldGOOS
            $env:GOARCH = $oldGOARCH
            $env:CGO_ENABLED = $oldCGOEnabled
            Remove-Item -LiteralPath $resourceFile -Force -ErrorAction SilentlyContinue
        }

        if (-not (Test-Path -LiteralPath $temporaryOutputFile -PathType Leaf)) {
            throw "build did not create temporary artifact $temporaryOutputFile"
        }
        Invoke-Go -Arguments @("run", "./cmd/artifactcheck", "-kind", "exe", "-arch", $TargetArchitecture, "-path", $temporaryOutputFile, "-icons-root", (Join-Path $repoRoot "assets\icons"), "-manifest", (Join-Path $repoRoot "build\windows\baremark.manifest"), "-version", $Version, "-original-name", $outputName, "-publish", $outputFile)
        if (-not (Test-Path -LiteralPath $outputFile -PathType Leaf)) {
            throw "build did not publish $outputFile"
        }
        Write-Host "Built $outputFile"
    }
    finally {
        Remove-Item -LiteralPath $temporaryOutputFile -Force -ErrorAction SilentlyContinue
    }
}

Push-Location $repoRoot
try {
    Invoke-Checks
    if ($Vulncheck) {
        Invoke-VulnerabilityCheck
    }

    $versionOutput = & go run ./cmd/resourcecheck -root $repoRoot
    if ($LASTEXITCODE -ne 0) {
        throw "build metadata validation failed with exit code $LASTEXITCODE"
    }
    $version = ($versionOutput | Select-Object -Last 1).Trim()
    if ([string]::IsNullOrWhiteSpace($version)) {
        throw "resource validator returned an empty version"
    }
    Write-Host "Building BareMark version $version"

    New-Item -ItemType Directory -Path $distDirectory -Force | Out-Null
    $architectures = if ($Architecture -eq "all") { @("amd64", "arm64") } else { @($Architecture) }
    foreach ($target in $architectures) {
        Build-Architecture -TargetArchitecture $target -Version $version -BuildConfiguration $Configuration
    }
}
finally {
    Pop-Location
}
