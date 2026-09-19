[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$Executable,

    [ValidateSet(100, 125, 150, 200)]
    [int]$ExpectedScale = 100,

    [ValidateRange(1, 100)]
    [int]$LaunchCount = 10,

    [string]$ReportDirectory = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$executablePath = (Resolve-Path -LiteralPath $Executable).Path
if ([string]::IsNullOrWhiteSpace($ReportDirectory)) {
    $ReportDirectory = Join-Path (Split-Path $PSScriptRoot -Parent) "dist\windows10-stage1-$ExpectedScale"
}
$reportPath = [IO.Path]::GetFullPath($ReportDirectory)
New-Item -ItemType Directory -Path $reportPath -Force | Out-Null

Add-Type -AssemblyName System.Drawing
if ($null -eq ("BareMarkSmokeNative" -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;

[StructLayout(LayoutKind.Sequential)]
public struct BareMarkSmokeRect {
    public int Left;
    public int Top;
    public int Right;
    public int Bottom;
}

public static class BareMarkSmokeNative {
    [DllImport("user32.dll", SetLastError = true)]
    public static extern bool SetWindowPos(IntPtr hwnd, IntPtr insertAfter, int x, int y, int width, int height, uint flags);

    [DllImport("user32.dll", SetLastError = true)]
    public static extern bool GetWindowRect(IntPtr hwnd, out BareMarkSmokeRect bounds);

    [DllImport("user32.dll")]
    public static extern uint GetDpiForWindow(IntPtr hwnd);

    [DllImport("user32.dll")]
    public static extern uint GetGuiResources(IntPtr process, uint flags);

    [DllImport("user32.dll", SetLastError = true)]
    public static extern bool PrintWindow(IntPtr hwnd, IntPtr hdc, uint flags);

    [DllImport("user32.dll", SetLastError = true)]
    public static extern IntPtr GetDlgItem(IntPtr parent, int controlID);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern IntPtr SendMessage(IntPtr hwnd, uint message, IntPtr wParam, IntPtr lParam);
}
'@
}

function Wait-MainWindow {
    param(
        [Parameter(Mandatory)][Diagnostics.Process]$Process,
        [int]$TimeoutMilliseconds = 10000
    )

    $deadline = [DateTime]::UtcNow.AddMilliseconds($TimeoutMilliseconds)
    do {
        $Process.Refresh()
        if ($Process.HasExited) {
            throw "BareMark exited before creating its main window; exit code $($Process.ExitCode)"
        }
        if ($Process.MainWindowHandle -ne [IntPtr]::Zero) {
            return $Process.MainWindowHandle
        }
        Start-Sleep -Milliseconds 25
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "BareMark did not create a main window within $TimeoutMilliseconds ms"
}

function Close-And-Verify {
    param(
        [Parameter(Mandatory)][Diagnostics.Process]$Process,
        [int]$TimeoutMilliseconds = 5000
    )

    if (-not $Process.CloseMainWindow()) {
        throw "CloseMainWindow failed for PID $($Process.Id)"
    }
    if (-not $Process.WaitForExit($TimeoutMilliseconds)) {
        throw "BareMark PID $($Process.Id) did not exit within $TimeoutMilliseconds ms"
    }
    if ($Process.ExitCode -ne 0) {
        throw "BareMark PID $($Process.Id) returned exit code $($Process.ExitCode)"
    }
    return $Process.ExitCode
}

function Save-WindowImage {
    param(
        [Parameter(Mandatory)][IntPtr]$Window,
        [Parameter(Mandatory)][string]$Path
    )

    $bounds = New-Object BareMarkSmokeRect
    if (-not [BareMarkSmokeNative]::GetWindowRect($Window, [ref]$bounds)) {
        throw "GetWindowRect failed with Win32 error $([Runtime.InteropServices.Marshal]::GetLastWin32Error())"
    }
    $width = $bounds.Right - $bounds.Left
    $height = $bounds.Bottom - $bounds.Top
    $bitmap = New-Object Drawing.Bitmap($width, $height, [Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $graphics = [Drawing.Graphics]::FromImage($bitmap)
    try {
        $hdc = $graphics.GetHdc()
        try {
            if (-not [BareMarkSmokeNative]::PrintWindow($Window, $hdc, 2)) {
                throw "PrintWindow failed with Win32 error $([Runtime.InteropServices.Marshal]::GetLastWin32Error())"
            }
        }
        finally {
            $graphics.ReleaseHdc($hdc)
        }
        $bitmap.Save($Path, [Drawing.Imaging.ImageFormat]::Png)
    }
    finally {
        $graphics.Dispose()
        $bitmap.Dispose()
    }
}

$os = Get-CimInstance Win32_OperatingSystem
$file = Get-Item -LiteralPath $executablePath
$hash = Get-FileHash -LiteralPath $executablePath -Algorithm SHA256
$expectedDpi = [uint32](96 * $ExpectedScale / 100)
$launches = [Collections.Generic.List[object]]::new()

Write-Host "Windows: $($os.Caption), version $($os.Version), build $($os.BuildNumber)"
Write-Host "Executable: $executablePath"
Write-Host "Expected scale: $ExpectedScale% ($expectedDpi DPI)"

for ($cycle = 1; $cycle -le $LaunchCount; $cycle++) {
    $timer = [Diagnostics.Stopwatch]::StartNew()
    $process = Start-Process -FilePath $executablePath -PassThru
    $window = Wait-MainWindow -Process $process
    $dpi = [BareMarkSmokeNative]::GetDpiForWindow($window)
    $exitCode = Close-And-Verify -Process $process
    $timer.Stop()
    $launches.Add([ordered]@{
        cycle = $cycle
        pid = $process.Id
        dpi = $dpi
        exitCode = $exitCode
        elapsedMilliseconds = $timer.ElapsedMilliseconds
    })
    Write-Host "Launch $cycle/${LaunchCount}: DPI $dpi, exit code $exitCode, $($timer.ElapsedMilliseconds) ms"
}

$visualProcess = Start-Process -FilePath $executablePath -PassThru
try {
$visualWindow = Wait-MainWindow -Process $visualProcess
$actualDpi = [BareMarkSmokeNative]::GetDpiForWindow($visualWindow)
if ($actualDpi -ne $expectedDpi) {
    $visualProcess.CloseMainWindow() | Out-Null
    $visualProcess.WaitForExit(5000) | Out-Null
    throw "Actual window DPI is $actualDpi; expected $expectedDpi for $ExpectedScale% scaling"
}

$gdiBefore = [BareMarkSmokeNative]::GetGuiResources($visualProcess.Handle, 0)
$userBefore = [BareMarkSmokeNative]::GetGuiResources($visualProcess.Handle, 1)
$scale = $actualDpi / 96.0
$sizes = @(
    @{ name = "small"; width = [int](480 * $scale); height = [int](320 * $scale) },
    @{ name = "normal"; width = [int](960 * $scale); height = [int](640 * $scale) },
    @{ name = "large"; width = [int](1280 * $scale); height = [int](800 * $scale) }
)

for ($cycle = 0; $cycle -lt 30; $cycle++) {
    $size = $sizes[$cycle % $sizes.Count]
    if (-not [BareMarkSmokeNative]::SetWindowPos($visualWindow, [IntPtr]::Zero, 40, 40, $size.width, $size.height, 0x0014)) {
        throw "SetWindowPos failed with Win32 error $([Runtime.InteropServices.Marshal]::GetLastWin32Error())"
    }
    Start-Sleep -Milliseconds 10
}

$screenshots = [Collections.Generic.List[string]]::new()
foreach ($size in $sizes) {
    if (-not [BareMarkSmokeNative]::SetWindowPos($visualWindow, [IntPtr]::Zero, 40, 40, $size.width, $size.height, 0x0014)) {
        throw "SetWindowPos failed with Win32 error $([Runtime.InteropServices.Marshal]::GetLastWin32Error())"
    }
    Start-Sleep -Milliseconds 250
    $imagePath = Join-Path $reportPath "resize-$($size.name)-$actualDpi-dpi.png"
    Save-WindowImage -Window $visualWindow -Path $imagePath
    $screenshots.Add($imagePath)
    Write-Host "Captured $imagePath"
}

$themeButton = [BareMarkSmokeNative]::GetDlgItem($visualWindow, 1001)
if ($themeButton -eq [IntPtr]::Zero) {
    throw "BareMark theme button was not found"
}
[BareMarkSmokeNative]::SendMessage($visualWindow, 0x0111, [IntPtr]1001, $themeButton) | Out-Null
Start-Sleep -Milliseconds 250
$darkImagePath = Join-Path $reportPath "theme-dark-$actualDpi-dpi.png"
Save-WindowImage -Window $visualWindow -Path $darkImagePath
$screenshots.Add($darkImagePath)
Write-Host "Captured $darkImagePath"

$gdiAfter = [BareMarkSmokeNative]::GetGuiResources($visualProcess.Handle, 0)
$userAfter = [BareMarkSmokeNative]::GetGuiResources($visualProcess.Handle, 1)
$visualExitCode = Close-And-Verify -Process $visualProcess
}
finally {
    $visualProcess.Refresh()
    if (-not $visualProcess.HasExited) {
        $visualProcess.CloseMainWindow() | Out-Null
        $visualProcess.WaitForExit(5000) | Out-Null
    }
}

$report = [ordered]@{
    generatedAt = [DateTimeOffset]::Now.ToString("o")
    operatingSystem = [ordered]@{
        caption = $os.Caption
        version = $os.Version
        build = $os.BuildNumber
    }
    executable = [ordered]@{
        path = $executablePath
        length = $file.Length
        sha256 = $hash.Hash
        fileVersion = $file.VersionInfo.FileVersion
    }
    expectedScalePercent = $ExpectedScale
    expectedDpi = $expectedDpi
    actualDpi = $actualDpi
    launches = $launches
    resize = [ordered]@{
        cycles = 30
        screenshots = $screenshots
        gdiHandlesBefore = $gdiBefore
        gdiHandlesAfter = $gdiAfter
        userHandlesBefore = $userBefore
        userHandlesAfter = $userAfter
        exitCode = $visualExitCode
    }
}
$jsonPath = Join-Path $reportPath "report.json"
$report | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $jsonPath -Encoding utf8
Write-Host "PASS: report written to $jsonPath"
