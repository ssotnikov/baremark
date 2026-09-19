//go:build windows

package win32

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/ssotnikov/baremark/internal/ui/theme"
)

type application struct {
	instance    uintptr
	className   *uint16
	callback    uintptr
	iconLoader  func(group uint16, width, height int) (uintptr, error)
	creating    *window
	windows     map[uintptr]*window // Only the locked UI thread touches this registry.
	quitOnEmpty bool
	runErr      error
}

type window struct {
	hwnd           uintptr
	dpi            uint32
	surface        *surface
	titleIconGroup uint16
	largeIcon      uintptr
	smallIcon      uintptr
	mode           theme.Mode
	appearance     theme.Resolved
	colors         theme.Palette
	themeButton    uintptr
	buttonFont     uintptr
	lastMouse      point
	lastKey        uintptr
}

// Run owns the UI thread and returns only after every window has been destroyed.
func Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := setDPIAwareness(); err != nil {
		return err
	}
	a := &application{windows: make(map[uintptr]*window), quitOnEmpty: true}
	instance, _, err := getModuleHandleW.Call(0)
	if instance == 0 {
		return callFailure("GetModuleHandleW", err)
	}
	a.instance = instance
	a.className, err = syscall.UTF16PtrFromString("BareMark.MainWindow")
	if err != nil {
		return err
	}
	a.callback = syscall.NewCallback(a.windowProc)
	cursor, _, err := loadCursorW.Call(0, idcArrow)
	if cursor == 0 {
		return callFailure("LoadCursorW", err)
	}
	class := windowClass{
		size:      uint32(unsafe.Sizeof(windowClass{})),
		wndProc:   a.callback,
		instance:  instance,
		cursor:    cursor, // Shared system cursor; the application must not destroy it.
		className: a.className,
	}
	if atom, _, err := registerClassExW.Call(ptr(&class)); atom == 0 {
		return callFailure("RegisterClassExW", err)
	}
	defer unregisterClassW.Call(ptr(a.className), instance)

	title, _ := syscall.UTF16PtrFromString("BareMark")
	appearance := theme.Resolve(theme.System, systemAppsDark(), highContrastEnabled())
	w := &window{dpi: 96, mode: theme.System, appearance: appearance, colors: paletteFor(appearance), titleIconGroup: appDarkIconID}
	a.creating = w
	defaultPosition := uintptr(uint32(0x80000000)) // CW_USEDEFAULT
	hwnd, _, err := createWindowExW.Call(
		0, ptr(a.className), ptr(title), wsOverlappedWnd|wsClipChildren,
		defaultPosition, defaultPosition, 960, 640,
		0, 0, instance, 0,
	)
	a.creating = nil
	if hwnd == 0 {
		return callFailure("CreateWindowExW", err)
	}
	defer func() {
		if a.windows[hwnd] != nil {
			destroyWindow.Call(hwnd)
		}
	}()
	if a.runErr != nil {
		return a.runErr
	}
	w.dpi = dpiForWindow(hwnd)
	if err := a.applyAppearance(w); err != nil {
		return err
	}
	if err := a.resize(w); err != nil {
		return err
	}
	if err := a.createThemeButton(w); err != nil {
		return err
	}
	showWindow.Call(hwnd, swShowNormal)
	updateWindow.Call(hwnd)
	setFocus.Call(w.themeButton)

	var msg message
	for {
		result, _, err := getMessageW.Call(ptr(&msg), 0, 0, 0)
		switch int32(result) {
		case -1:
			return callFailure("GetMessageW", err)
		case 0:
			if a.runErr != nil {
				return a.runErr
			}
			return nil
		default:
			translateMessage.Call(ptr(&msg))
			dispatchMessageW.Call(ptr(&msg))
		}
	}
}

// ShowStartupError keeps implementation details out of the user-facing dialog.
func ShowStartupError() {
	title, _ := syscall.UTF16PtrFromString("BareMark")
	message, _ := syscall.UTF16PtrFromString("BareMark could not start.")
	messageBoxW.Call(0, ptr(message), ptr(title), 0x10) // MB_ICONERROR
}

func setDPIAwareness() error {
	// The manifest already requests Per-Monitor V2; ERROR_ACCESS_DENIED means
	// Windows applied that manifest before main and the process context is set.
	if err := setProcessDPIAwareContext.Find(); err != nil {
		return nil // The manifest remains the safe fallback.
	}
	const perMonitorV2 = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
	if ok, _, err := setProcessDPIAwareContext.Call(perMonitorV2); ok == 0 {
		if errors.Is(err, syscall.Errno(5)) {
			return nil
		}
		return callFailure("SetProcessDpiAwarenessContext", err)
	}
	return nil
}

func dpiForWindow(hwnd uintptr) uint32 {
	if getDPIForWindow.Find() == nil {
		if dpi, _, _ := getDPIForWindow.Call(hwnd); dpi != 0 {
			return uint32(dpi)
		}
	}
	if getDPIForSystem.Find() == nil {
		if dpi, _, _ := getDPIForSystem.Call(); dpi != 0 {
			return uint32(dpi)
		}
	}
	return 96
}

func (a *application) windowProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	if msg == wmNCCreate && a.creating != nil {
		a.creating.hwnd = hwnd
		a.windows[hwnd] = a.creating
	}
	w := a.windows[hwnd]
	if w == nil {
		result, _, _ := defWindowProcW.Call(hwnd, msg, wParam, lParam)
		return result
	}
	switch msg {
	case wmPaint:
		if err := a.paint(w); err != nil {
			a.fail(err)
		}
		return 0
	case wmEraseBkgnd:
		if w.surface != nil {
			return 1
		}
	case wmSize:
		if err := a.resize(w); err != nil {
			a.fail(err)
		} else if err := a.layoutButton(w); err != nil {
			a.fail(err)
		} else {
			invalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case wmDPIChanged:
		if err := a.dpiChanged(w, wParam, lParam); err != nil {
			a.fail(err)
		}
		return 0
	case wmGetMinMaxInfo:
		if lParam != 0 {
			limits := (*minMaxInfo)(unsafe.Pointer(lParam))
			dpi := w.dpi
			if dpi == 0 {
				dpi = 96
			}
			limits.minTrackSize = point{scaleDIP(480, dpi), scaleDIP(320, dpi)}
		}
		return 0
	case wmSysColorChange, wmSettingChange, wmThemeChanged:
		if err := a.applyAppearance(w); err != nil {
			a.fail(err)
		}
		return 0
	case wmCommand:
		if uint16(wParam) == themeButtonID && uint16(wParam>>16) == 0 && lParam == w.themeButton {
			w.mode = theme.Toggle(w.appearance)
			if err := a.applyAppearance(w); err != nil {
				a.fail(err)
			}
			return 0
		}
	case wmDrawItem:
		if uint16(wParam) == themeButtonID && lParam != 0 {
			if err := drawThemeButton(w, (*drawItemStruct)(unsafe.Pointer(lParam))); err != nil {
				a.fail(err)
			}
			return 1
		}
	case wmMouseMove:
		w.lastMouse = point{int32(int16(lParam)), int32(int16(lParam >> 16))}
	case wmLButtonDown:
		setFocus.Call(hwnd)
	case wmLButtonUp:
		// The bootstrap window has no interactive control yet.
	case wmKeyDown:
		w.lastKey = wParam
	case wmClose:
		if ok, _, err := destroyWindow.Call(hwnd); ok == 0 {
			a.fail(callFailure("DestroyWindow", err))
		}
		return 0
	case wmNCDestroy:
		result, _, _ := defWindowProcW.Call(hwnd, msg, wParam, lParam)
		if err := w.release(); err != nil && a.runErr == nil {
			a.runErr = err
		}
		delete(a.windows, hwnd)
		if a.quitOnEmpty && len(a.windows) == 0 {
			postQuitMessage.Call(0)
		}
		return result
	}
	result, _, _ := defWindowProcW.Call(hwnd, msg, wParam, lParam)
	return result
}

func scaleDIP(dip int32, dpi uint32) int32 { return (dip*int32(dpi) + 48) / 96 }

func (a *application) fail(err error) {
	if a.runErr == nil {
		a.runErr = err
		postQuitMessage.Call(1)
	}
}

func (a *application) resize(w *window) error {
	var client rect
	if ok, _, err := getClientRect.Call(w.hwnd, ptr(&client)); ok == 0 {
		return callFailure("GetClientRect", err)
	}
	width, height := int(client.right-client.left), int(client.bottom-client.top)
	if w.surface != nil && w.surface.width == width && w.surface.height == height {
		return nil
	}
	next, err := newSurface(width, height)
	if err != nil {
		return fmt.Errorf("resize backbuffer to %dx%d: %w", width, height, err)
	}
	old := w.surface
	w.surface = next
	w.surface.fillRGB(uint32(w.colors.WindowBackground))
	if err := old.close(); err != nil {
		return err
	}
	return nil
}

func (a *application) paint(w *window) error {
	var state paintStruct
	hdc, _, err := beginPaint.Call(w.hwnd, ptr(&state))
	if hdc == 0 {
		return callFailure("BeginPaint", err)
	}
	defer endPaint.Call(w.hwnd, ptr(&state))
	if w.surface == nil {
		if err := a.resize(w); err != nil {
			return err
		}
	}
	return w.surface.blit(hdc, state.paint)
}

func (a *application) dpiChanged(w *window, wParam, lParam uintptr) error {
	newDPI := uint32(wParam & 0xffff)
	if newDPI != 0 {
		w.dpi = newDPI
	}
	if lParam != 0 {
		bounds := (*rect)(unsafe.Pointer(lParam))
		if ok, _, err := setWindowPos.Call(w.hwnd, 0, uintptr(bounds.left), uintptr(bounds.top), uintptr(bounds.right-bounds.left), uintptr(bounds.bottom-bounds.top), swpNoZOrder|swpNoActivate); ok == 0 {
			return callFailure("SetWindowPos", err)
		}
	}
	if err := a.updateButtonFont(w); err != nil {
		return err
	}
	if err := a.layoutButton(w); err != nil {
		return err
	}
	if err := a.updateIcons(w); err != nil {
		return err
	}
	if err := a.resize(w); err != nil {
		return err
	}
	invalidateRect.Call(w.hwnd, 0, 0)
	return nil
}

func metricForDPI(metric int, dpi uint32) int {
	if getSystemMetricsForDPI.Find() == nil {
		if value, _, _ := getSystemMetricsForDPI.Call(uintptr(metric), uintptr(dpi)); value != 0 {
			return int(value)
		}
	}
	value, _, _ := getSystemMetrics.Call(uintptr(metric))
	return int(value)
}

func (a *application) updateIcons(w *window) error {
	large, err := a.loadIcon(appDarkIconID, metricForDPI(smCXIcon, w.dpi), metricForDPI(smCYIcon, w.dpi))
	if err != nil {
		return err
	}
	small, err := a.loadIcon(w.titleIconGroup, metricForDPI(smCXSmIcon, w.dpi), metricForDPI(smCYSmIcon, w.dpi))
	if err != nil {
		destroyIcon.Call(large)
		return err
	}
	sendMessageW.Call(w.hwnd, wmSetIcon, iconBig, large)
	sendMessageW.Call(w.hwnd, wmSetIcon, iconSmall, small)
	oldLarge, oldSmall := w.largeIcon, w.smallIcon
	w.largeIcon, w.smallIcon = large, small
	if oldLarge != 0 {
		destroyIcon.Call(oldLarge)
	}
	if oldSmall != 0 {
		destroyIcon.Call(oldSmall)
	}
	return nil
}

func (a *application) updateTitleIcon(w *window, group uint16) error {
	small, err := a.loadIcon(group, metricForDPI(smCXSmIcon, w.dpi), metricForDPI(smCYSmIcon, w.dpi))
	if err != nil {
		return err
	}
	sendMessageW.Call(w.hwnd, wmSetIcon, iconSmall, small)
	oldSmall := w.smallIcon
	w.smallIcon = small
	w.titleIconGroup = group
	if oldSmall != 0 {
		destroyIcon.Call(oldSmall)
	}
	return nil
}

func (a *application) loadIcon(group uint16, width, height int) (uintptr, error) {
	if a.iconLoader != nil {
		return a.iconLoader(group, width, height)
	}
	icon, _, err := loadImageW.Call(a.instance, uintptr(group), imageIcon, uintptr(width), uintptr(height), 0)
	if icon == 0 {
		return 0, callFailure("LoadImageW(HICON)", err)
	}
	return icon, nil
}

func (w *window) release() error {
	if w.buttonFont != 0 {
		deleteObject.Call(w.buttonFont)
		w.buttonFont = 0
	}
	if w.largeIcon != 0 {
		destroyIcon.Call(w.largeIcon)
		w.largeIcon = 0
	}
	if w.smallIcon != 0 {
		destroyIcon.Call(w.smallIcon)
		w.smallIcon = 0
	}
	err := w.surface.close()
	w.surface = nil
	return err
}
