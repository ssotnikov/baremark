//go:build windows

package win32

import (
	"fmt"
	"syscall"

	"github.com/ssotnikov/baremark/internal/ui/theme"
)

func (a *application) createThemeButton(w *window) error {
	class, _ := syscall.UTF16PtrFromString("BUTTON")
	label, _ := syscall.UTF16PtrFromString(buttonLabel(w))
	button, _, err := createWindowExW.Call(0, ptr(class), ptr(label), wsChild|wsVisible|wsTabStop|bsOwnerDraw,
		0, 0, 1, 1, w.hwnd, themeButtonID, a.instance, 0)
	if button == 0 {
		return callFailure("CreateWindowExW(theme button)", err)
	}
	w.themeButton = button
	if err := a.updateButtonFont(w); err != nil {
		return err
	}
	return a.layoutButton(w)
}

func (a *application) updateButtonFont(w *window) error {
	if w.themeButton == 0 {
		return nil
	}
	var font logFont
	font.height = -scaleDIP(15, w.dpi)
	font.weight = 400
	font.charSet = 1 // DEFAULT_CHARSET
	font.quality = 5 // CLEARTYPE_QUALITY
	copy(font.face[:], syscall.StringToUTF16("Segoe UI"))
	handle, _, err := createFontIndirectW.Call(ptr(&font))
	if handle == 0 {
		return callFailure("CreateFontIndirectW", err)
	}
	sendMessageW.Call(w.themeButton, wmSetFont, handle, 1)
	if w.buttonFont != 0 {
		deleteObject.Call(w.buttonFont)
	}
	w.buttonFont = handle
	return nil
}

func (a *application) layoutButton(w *window) error {
	if w.themeButton == 0 {
		return nil
	}
	var client rect
	if ok, _, err := getClientRect.Call(w.hwnd, ptr(&client)); ok == 0 {
		return callFailure("GetClientRect(theme button)", err)
	}
	margin, top := scaleDIP(24, w.dpi), scaleDIP(24, w.dpi)
	width, height := scaleDIP(220, w.dpi), scaleDIP(42, w.dpi)
	if available := client.right - 2*margin; width > available {
		width = max(1, available)
	}
	if ok, _, err := moveWindow.Call(w.themeButton, uintptr(margin), uintptr(top), uintptr(width), uintptr(height), 1); ok == 0 {
		return callFailure("MoveWindow(theme button)", err)
	}
	return nil
}

func buttonLabel(w *window) string {
	if w.mode == theme.Dark || w.mode == theme.System && w.appearance.Dark {
		return "Switch to light theme"
	}
	return "Switch to dark theme"
}

func (w *window) updateButtonLabel() error {
	if w.themeButton == 0 {
		return nil
	}
	label, _ := syscall.UTF16PtrFromString(buttonLabel(w))
	if ok, _, err := setWindowTextW.Call(w.themeButton, ptr(label)); ok == 0 {
		return callFailure("SetWindowTextW(theme button)", err)
	}
	invalidateRect.Call(w.themeButton, 0, 1)
	return nil
}

func drawThemeButton(w *window, item *drawItemStruct) error {
	if item.controlType != odtButton || item.hwnd != w.themeButton {
		return nil
	}
	colors := w.colors
	background := colors.Button
	if item.state&odsSelected != 0 {
		background = colors.ButtonPressed
	}
	backgroundBrush, err := solidBrush(background)
	if err != nil {
		return err
	}
	defer deleteObject.Call(backgroundBrush)
	if ok, _, callErr := fillRect.Call(item.hdc, ptr(&item.bounds), backgroundBrush); ok == 0 {
		return callFailure("FillRect(theme button)", callErr)
	}
	borderBrush, err := solidBrush(colors.Border)
	if err != nil {
		return err
	}
	defer deleteObject.Call(borderBrush)
	if ok, _, callErr := frameRect.Call(item.hdc, ptr(&item.bounds), borderBrush); ok == 0 {
		return callFailure("FrameRect(theme button)", callErr)
	}
	previousMode, _, _ := setBkMode.Call(item.hdc, transparent)
	previousColor, _, _ := setTextColor.Call(item.hdc, uintptr(rgbToColorRef(uint32(colors.TextPrimary))))
	defer setBkMode.Call(item.hdc, previousMode)
	defer setTextColor.Call(item.hdc, previousColor)
	label := syscall.StringToUTF16(buttonLabel(w))
	textBounds := item.bounds
	if result, _, callErr := drawTextW.Call(item.hdc, ptr(&label[0]), uintptr(len(label)-1), ptr(&textBounds), dtCenter|dtVCenter|dtSingleLine); result == 0 {
		return callFailure("DrawTextW(theme button)", callErr)
	}
	if item.state&odsFocus != 0 {
		focus := item.bounds
		inset := scaleDIP(4, w.dpi)
		focus.left += inset
		focus.top += inset
		focus.right -= inset
		focus.bottom -= inset
		if ok, _, callErr := drawFocusRect.Call(item.hdc, ptr(&focus)); ok == 0 {
			return callFailure("DrawFocusRect(theme button)", callErr)
		}
	}
	return nil
}

func solidBrush(rgb theme.Color) (uintptr, error) {
	brush, _, err := createSolidBrush.Call(uintptr(rgbToColorRef(uint32(rgb))))
	if brush == 0 {
		return 0, fmt.Errorf("create theme brush: %w", callFailure("CreateSolidBrush", err))
	}
	return brush, nil
}
