//go:build windows

package win32

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/ssotnikov/baremark/internal/ui/theme"
)

func TestThemeToggleWindowResources(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := setDPIAwareness(); err != nil {
		t.Fatal(err)
	}
	process, _, _ := getCurrentProcess.Call()
	beforeGDI, _, _ := getGuiResources.Call(process, 0)
	beforeUSER, _, _ := getGuiResources.Call(process, 1)
	instance, _, callErr := getModuleHandleW.Call(0)
	if instance == 0 {
		t.Fatal(callFailure("GetModuleHandleW", callErr))
	}
	className, _ := syscall.UTF16PtrFromString("BareMark.ThemeIntegrationTest")
	buttonCursor, _, callErr := loadCursorW.Call(0, idcArrow)
	if buttonCursor == 0 {
		t.Fatal(callFailure("LoadCursorW", callErr))
	}
	a := &application{instance: instance, className: className, windows: make(map[uintptr]*window)}
	a.iconLoader = func(_ uint16, _, _ int) (uintptr, error) {
		// go test binaries have no BareMark PE icon resources. Use a real owned
		// HICON to exercise WM_SETICON and DestroyIcon without external files.
		shared, _, err := loadIconW.Call(0, 32512) // IDI_APPLICATION
		if shared == 0 {
			return 0, callFailure("LoadIconW", err)
		}
		owned, _, err := copyIcon.Call(shared)
		if owned == 0 {
			return 0, callFailure("CopyIcon", err)
		}
		return owned, nil
	}
	a.callback = syscall.NewCallback(a.windowProc)
	class := windowClass{size: uint32(unsafe.Sizeof(windowClass{})), wndProc: a.callback, instance: instance, cursor: buttonCursor, className: className}
	if atom, _, err := registerClassExW.Call(ptr(&class)); atom == 0 {
		t.Fatal(callFailure("RegisterClassExW", err))
	}
	defer unregisterClassW.Call(ptr(className), instance)
	w := &window{dpi: 96, mode: theme.Light, appearance: theme.Resolve(theme.Light, false, false), titleIconGroup: appDarkIconID}
	w.colors = paletteFor(w.appearance)
	a.creating = w
	title, _ := syscall.UTF16PtrFromString("BareMark integration test")
	hwnd, _, callErr := createWindowExW.Call(0, ptr(className), ptr(title), wsOverlappedWnd|wsClipChildren,
		0, 0, 640, 480, 0, 0, instance, 0)
	a.creating = nil
	if hwnd == 0 {
		t.Fatal(callFailure("CreateWindowExW", callErr))
	}
	defer func() {
		if a.windows[hwnd] != nil {
			destroyWindow.Call(hwnd)
		}
	}()
	w.dpi = dpiForWindow(hwnd)
	if err := a.applyAppearance(w); err != nil {
		t.Fatal(err)
	}
	if err := a.resize(w); err != nil {
		t.Fatal(err)
	}
	if err := a.createThemeButton(w); err != nil {
		t.Fatal(err)
	}
	if w.themeButton == 0 || w.largeIcon == 0 || w.smallIcon == 0 {
		t.Fatal("theme button and both window icons must be installed")
	}
	for _, dpi := range []uint32{96, 120, 144, 192} {
		suggested := rect{left: -40, top: 50, right: -40 + scaleDIP(640, dpi), bottom: 50 + scaleDIP(480, dpi)}
		sendMessageW.Call(hwnd, wmDPIChanged, uintptr(dpi)|uintptr(dpi)<<16, ptr(&suggested))
		if a.runErr != nil || w.dpi != dpi || w.buttonFont == 0 || w.surface == nil {
			t.Fatalf("WM_DPICHANGED(%d) failed: error=%v, dpi=%d", dpi, a.runErr, w.dpi)
		}
	}
	setupGDI, _, _ := getGuiResources.Call(process, 0)
	setupUSER, _, _ := getGuiResources.Call(process, 1)
	stableLargeIcon := w.largeIcon
	captionChecks := 0
	for cycle := 0; cycle < 20; cycle++ {
		sendMessageW.Call(hwnd, wmCommand, themeButtonID, w.themeButton)
		wantDark := cycle%2 == 0
		if w.appearance.Dark != wantDark {
			t.Fatalf("toggle %d: dark=%t, want %t", cycle, w.appearance.Dark, wantDark)
		}
		if a.runErr != nil {
			t.Fatal(a.runErr)
		}
		if large, _, _ := sendMessageW.Call(hwnd, 0x007F, iconBig, 0); large != w.largeIcon { // WM_GETICON
			t.Fatalf("toggle %d: installed big icon differs from owned icon", cycle)
		}
		if w.largeIcon != stableLargeIcon {
			t.Fatalf("toggle %d: theme switch replaced the stable Shell/Alt+Tab icon", cycle)
		}
		if small, _, _ := sendMessageW.Call(hwnd, 0x007F, iconSmall, 0); small != w.smallIcon { // WM_GETICON
			t.Fatalf("toggle %d: installed title icon differs from owned icon", cycle)
		}
		dark := uint32(0)
		if w.appearance.Dark {
			dark = 1
		}
		if setDwmAttribute(hwnd, dwmDarkMode, &dark) {
			captionChecks++
			if w.titleIconGroup != iconGroupForCaption(w.appearance.Dark) {
				t.Fatalf("toggle %d: title icon group=%d does not match caption contrast", cycle, w.titleIconGroup)
			}
		}
	}
	toggledGDI, _, _ := getGuiResources.Call(process, 0)
	toggledUSER, _, _ := getGuiResources.Call(process, 1)
	if toggledGDI > setupGDI+1 || toggledUSER > setupUSER+1 {
		t.Fatalf("resources grew while toggling: GDI %d -> %d, USER %d -> %d", setupGDI, toggledGDI, setupUSER, toggledUSER)
	}
	if captionChecks == 0 {
		t.Log("DWM immersive-dark attribute unavailable; standard title-bar fallback remains in use")
	}
	if ok, _, err := destroyWindow.Call(hwnd); ok == 0 {
		t.Fatal(callFailure("DestroyWindow", err))
	}
	if a.runErr != nil {
		t.Fatal(a.runErr)
	}
	if len(a.windows) != 0 || w.surface != nil || w.largeIcon != 0 || w.smallIcon != 0 || w.buttonFont != 0 {
		t.Fatal("window resources remain after WM_NCDESTROY")
	}
	// Win32 may retain a few process-wide GDI objects after the first window.
	// Subsequent create/destroy cycles must not grow beyond that warm baseline.
	warmGDI, _, _ := getGuiResources.Call(process, 0)
	warmUSER, _, _ := getGuiResources.Call(process, 1)
	for cycle := 0; cycle < 10; cycle++ {
		next := &window{dpi: 96, mode: theme.Light, appearance: theme.Resolve(theme.Light, false, false), titleIconGroup: appDarkIconID}
		next.colors = paletteFor(next.appearance)
		a.creating = next
		other, _, err := createWindowExW.Call(0, ptr(className), ptr(title), wsOverlappedWnd|wsClipChildren,
			0, 0, 640, 480, 0, 0, instance, 0)
		a.creating = nil
		if other == 0 {
			t.Fatal(callFailure("CreateWindowExW(repeat)", err))
		}
		next.dpi = dpiForWindow(other)
		if err := a.applyAppearance(next); err != nil {
			t.Fatal(err)
		}
		if err := a.createThemeButton(next); err != nil {
			t.Fatal(err)
		}
		if ok, _, err := destroyWindow.Call(other); ok == 0 {
			t.Fatal(callFailure("DestroyWindow(repeat)", err))
		}
	}
	afterGDI, _, _ := getGuiResources.Call(process, 0)
	afterUSER, _, _ := getGuiResources.Call(process, 1)
	if afterGDI > warmGDI+1 || afterUSER > warmUSER+1 {
		t.Fatalf("resources grew across windows: GDI %d -> %d, USER %d -> %d (initial %d/%d)", warmGDI, afterGDI, warmUSER, afterUSER, beforeGDI, beforeUSER)
	}
}
