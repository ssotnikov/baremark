//go:build windows

package win32

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	wmNCCreate         = 0x0081
	wmNCDestroy        = 0x0082
	wmNCActivate       = 0x0086
	wmPaint            = 0x000F
	wmEraseBkgnd       = 0x0014
	wmSize             = 0x0005
	wmDPIChanged       = 0x02E0
	wmGetMinMaxInfo    = 0x0024
	wmClose            = 0x0010
	wmKeyDown          = 0x0100
	wmMouseMove        = 0x0200
	wmLButtonDown      = 0x0201
	wmLButtonUp        = 0x0202
	wmSysColorChange   = 0x0015
	wmSettingChange    = 0x001A
	wmThemeChanged     = 0x031A
	wmCommand          = 0x0111
	wmDrawItem         = 0x002B
	wmSetFont          = 0x0030
	wmSetIcon          = 0x0080
	iconSmall          = 0
	iconBig            = 1
	wsOverlappedWnd    = 0x00CF0000
	wsClipChildren     = 0x02000000
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabStop          = 0x00010000
	bsOwnerDraw        = 0x0000000B
	themeButtonID      = 1001
	odtButton          = 4
	odsSelected        = 0x0001
	odsFocus           = 0x0010
	transparent        = 1
	dtCenter           = 0x00000001
	dtVCenter          = 0x00000004
	dtSingleLine       = 0x00000020
	spiGetHighContrast = 0x0042
	hcfHighContrastOn  = 0x00000001
	colorWindowText    = 8
	colorBtnFace       = 15
	colorBtnText       = 18
	colorHighlight     = 13
	dwmDarkMode        = 20
	dwmBorderColor     = 34
	dwmCaptionColor    = 35
	dwmTextColor       = 36
	dwmColorDefault    = 0xffffffff
	swShowNormal       = 1
	swpNoSize          = 0x0001
	swpNoMove          = 0x0002
	swpNoZOrder        = 0x0004
	swpNoActivate      = 0x0010
	swpFrameChanged    = 0x0020
	imageIcon          = 1
	idcArrow           = 32512
	colorWindow        = 5
	colorActiveCaption = 2
	smCXIcon           = 11
	smCYIcon           = 12
	smCXSmIcon         = 49
	smCYSmIcon         = 50
	srccopy            = 0x00CC0020
	dibRGBColors       = 0
	biRGB              = 0
	appDarkIconID      = 1
	appLightIconID     = 101
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	registerClassExW          = user32.NewProc("RegisterClassExW")
	unregisterClassW          = user32.NewProc("UnregisterClassW")
	createWindowExW           = user32.NewProc("CreateWindowExW")
	defWindowProcW            = user32.NewProc("DefWindowProcW")
	destroyWindow             = user32.NewProc("DestroyWindow")
	showWindow                = user32.NewProc("ShowWindow")
	updateWindow              = user32.NewProc("UpdateWindow")
	getMessageW               = user32.NewProc("GetMessageW")
	translateMessage          = user32.NewProc("TranslateMessage")
	dispatchMessageW          = user32.NewProc("DispatchMessageW")
	postQuitMessage           = user32.NewProc("PostQuitMessage")
	beginPaint                = user32.NewProc("BeginPaint")
	endPaint                  = user32.NewProc("EndPaint")
	getClientRect             = user32.NewProc("GetClientRect")
	invalidateRect            = user32.NewProc("InvalidateRect")
	setWindowPos              = user32.NewProc("SetWindowPos")
	loadCursorW               = user32.NewProc("LoadCursorW")
	loadImageW                = user32.NewProc("LoadImageW")
	loadIconW                 = user32.NewProc("LoadIconW")
	copyIcon                  = user32.NewProc("CopyIcon")
	destroyIcon               = user32.NewProc("DestroyIcon")
	sendMessageW              = user32.NewProc("SendMessageW")
	setFocus                  = user32.NewProc("SetFocus")
	setWindowTextW            = user32.NewProc("SetWindowTextW")
	moveWindow                = user32.NewProc("MoveWindow")
	fillRect                  = user32.NewProc("FillRect")
	frameRect                 = user32.NewProc("FrameRect")
	drawTextW                 = user32.NewProc("DrawTextW")
	drawFocusRect             = user32.NewProc("DrawFocusRect")
	systemParametersInfoW     = user32.NewProc("SystemParametersInfoW")
	getDPIForWindow           = user32.NewProc("GetDpiForWindow")
	getDPIForSystem           = user32.NewProc("GetDpiForSystem")
	setProcessDPIAwareContext = user32.NewProc("SetProcessDpiAwarenessContext")
	getSystemMetricsForDPI    = user32.NewProc("GetSystemMetricsForDpi")
	getSystemMetrics          = user32.NewProc("GetSystemMetrics")
	getSysColor               = user32.NewProc("GetSysColor")
	getGuiResources           = user32.NewProc("GetGuiResources")
	getForegroundWindow       = user32.NewProc("GetForegroundWindow")
	messageBoxW               = user32.NewProc("MessageBoxW")
	getModuleHandleW          = kernel32.NewProc("GetModuleHandleW")
	getCurrentProcess         = kernel32.NewProc("GetCurrentProcess")
	regGetValueW              = advapi32.NewProc("RegGetValueW")
	dwmSetWindowAttribute     = dwmapi.NewProc("DwmSetWindowAttribute")
	createCompatibleDC        = gdi32.NewProc("CreateCompatibleDC")
	deleteDC                  = gdi32.NewProc("DeleteDC")
	createDIBSection          = gdi32.NewProc("CreateDIBSection")
	selectObject              = gdi32.NewProc("SelectObject")
	deleteObject              = gdi32.NewProc("DeleteObject")
	bitBlt                    = gdi32.NewProc("BitBlt")
	gdiFlush                  = gdi32.NewProc("GdiFlush")
	createSolidBrush          = gdi32.NewProc("CreateSolidBrush")
	createFontIndirectW       = gdi32.NewProc("CreateFontIndirectW")
	setTextColor              = gdi32.NewProc("SetTextColor")
	setBkMode                 = gdi32.NewProc("SetBkMode")
)

type point struct{ x, y int32 }
type rect struct{ left, top, right, bottom int32 }

type paintStruct struct {
	hdc       uintptr
	erase     int32
	paint     rect
	restore   int32
	incUpdate int32
	reserved  [32]byte
}

type message struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

type minMaxInfo struct {
	reserved     point
	maxSize      point
	maxPosition  point
	minTrackSize point
	maxTrackSize point
}

type windowClass struct {
	size        uint32
	style       uint32
	wndProc     uintptr
	classExtra  int32
	windowExtra int32
	instance    uintptr
	icon        uintptr
	cursor      uintptr
	background  uintptr
	menuName    *uint16
	className   *uint16
	smallIcon   uintptr
}

type bitmapInfo struct {
	header struct {
		size            uint32
		width           int32
		height          int32
		planes          uint16
		bitCount        uint16
		compression     uint32
		imageSize       uint32
		xPixelsPerM     int32
		yPixelsPerM     int32
		colorsUsed      uint32
		colorsImportant uint32
	}
	color [4]byte
}

type drawItemStruct struct {
	controlType uint32
	controlID   uint32
	itemID      uint32
	action      uint32
	state       uint32
	hwnd        uintptr
	hdc         uintptr
	bounds      rect
	itemData    uintptr
}

type highContrast struct {
	size          uint32
	flags         uint32
	defaultScheme *uint16
}

type logFont struct {
	height         int32
	width          int32
	escapement     int32
	orientation    int32
	weight         int32
	italic         byte
	underline      byte
	strikeOut      byte
	charSet        byte
	outPrecision   byte
	clipPrecision  byte
	quality        byte
	pitchAndFamily byte
	face           [32]uint16
}

func callFailure(name string, err error) error {
	if err == nil || err == syscall.Errno(0) {
		return fmt.Errorf("%s failed", name)
	}
	return fmt.Errorf("%s: %w", name, err)
}

func ptr[T any](value *T) uintptr { return uintptr(unsafe.Pointer(value)) }
