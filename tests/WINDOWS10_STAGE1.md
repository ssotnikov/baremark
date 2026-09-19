# Windows 10 stage 1 acceptance test

Run this test on Windows 10 22H2 build 19045 using the AMD64 release executable produced by `build.ps1`. Repeat it at each display scale: 100%, 125%, 150%, and 200%.

## Before each run

1. Open **Settings → System → Display** and select the required scale.
2. Sign out and back in if Windows requests it.
3. Build or copy the exact EXE being tested. Do not rename or replace it between runs.
4. Close other BareMark instances.

## Command

From the repository root in PowerShell 7:

```powershell
.\tests\windows10-stage1.ps1 `
    -Executable .\dist\baremark-0.1.0-amd64.exe `
    -ExpectedScale 100 `
    -LaunchCount 10
```

Change `-ExpectedScale` to `125`, `150`, and `200` for the other runs. The script fails if the real DPI reported by `GetDpiForWindow` does not equal 96, 120, 144, or 192 respectively.

The script performs the following checks:

- launches and closes the real EXE ten times through the normal main-window close path;
- requires exit code `0` for every process;
- performs 30 rapid resize cycles;
- captures small, normal, and large window images after the resize cycle;
- switches the real window through its theme button and captures the resulting dark client area and native caption;
- records GDI and USER handle counts before and after resizing;
- records the Windows build, EXE SHA-256, file version, actual DPI, timings, and results.

## Visual review

Open the three `resize-*-{dpi}-dpi.png` files and `theme-dark-{dpi}-dpi.png` produced by every run and verify:

- the client area is completely painted, without black/uninitialized regions or trails;
- the theme button is fully visible, readable, and not clipped;
- the title, border, and application icon are rendered correctly;
- text and geometry scale consistently between 100%, 125%, 150%, and 200%.

Also launch BareMark once at each scale and manually drag every window edge and corner rapidly for at least ten seconds. Report any flicker, stale pixels, trails, clipped controls, or delayed repaint with a screenshot or short recording.

If two monitors with different scaling are available, move the same open window between them five times. Confirm that its content, button, title icon, and minimum size update after every move.

## Results to return

Return one folder for each scale, containing:

```text
windows10-stage1-100/
windows10-stage1-125/
windows10-stage1-150/
windows10-stage1-200/
```

Each folder must contain `report.json` and its four PNG files. Include the following manual result alongside them:

```text
Windows edition/build:
GPU and driver version:
Monitor model(s):
Monitor scale(s):

100% rapid resize: PASS / FAIL
125% rapid resize: PASS / FAIL
150% rapid resize: PASS / FAIL
200% rapid resize: PASS / FAIL
Mixed-DPI monitor transfer: PASS / FAIL / NOT AVAILABLE
Light/Dark button using keyboard: PASS / FAIL
Title-bar colors and title icon: PASS / FAIL
Taskbar and Alt+Tab icon: PASS / FAIL

Notes and reproduction steps for every failure:
```

Do not report a visual item as passed solely because the script exited successfully. The PNG review and interactive resize remain required.
