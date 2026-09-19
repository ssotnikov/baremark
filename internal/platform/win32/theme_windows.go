//go:build windows

package win32

import (
	"syscall"
	"unsafe"

	"github.com/ssotnikov/baremark/internal/ui/theme"
)

func systemAppsDark() bool {
	path, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)
	name, _ := syscall.UTF16PtrFromString("AppsUseLightTheme")
	var light uint32
	size := uint32(unsafe.Sizeof(light))
	// RRF_RT_REG_DWORD requires the value to be a DWORD. An unavailable setting
	// falls back to Light, as required by the appearance specification.
	status, _, _ := regGetValueW.Call(0x80000001, ptr(path), ptr(name), 0x18, 0, ptr(&light), ptr(&size))
	return status == 0 && size == 4 && light == 0
}

func highContrastEnabled() bool {
	state := highContrast{size: uint32(unsafe.Sizeof(highContrast{}))}
	ok, _, _ := systemParametersInfoW.Call(spiGetHighContrast, uintptr(state.size), ptr(&state), 0)
	return ok != 0 && state.flags&hcfHighContrastOn != 0
}

func paletteFor(appearance theme.Resolved) theme.Palette {
	if appearance.HighContrast {
		return theme.Palette{
			WindowBackground: theme.Color(systemColorRGB(colorWindow)),
			Surface:          theme.Color(systemColorRGB(colorWindow)),
			SurfaceRaised:    theme.Color(systemColorRGB(colorBtnFace)),
			Button:           theme.Color(systemColorRGB(colorBtnFace)),
			ButtonPressed:    theme.Color(systemColorRGB(colorHighlight)),
			Border:           theme.Color(systemColorRGB(colorWindowText)),
			TextPrimary:      theme.Color(systemColorRGB(colorBtnText)),
			TextSecondary:    theme.Color(systemColorRGB(colorWindowText)),
			FocusRing:        theme.Color(systemColorRGB(colorWindowText)),
			Accent:           theme.Color(systemColorRGB(colorHighlight)),
		}
	}
	return theme.PaletteFor(appearance.Dark)
}

func systemColorRGB(index uintptr) uint32 {
	color, _, _ := getSysColor.Call(index)
	return colorRefToRGB(uint32(color))
}

func colorRefToRGB(color uint32) uint32 {
	return (color&0xff)<<16 | color&0xff00 | (color>>16)&0xff
}

func rgbToColorRef(color uint32) uint32 { return colorRefToRGB(color) }

func darkBackground(color uint32) bool {
	r, g, b := (color>>16)&0xff, (color>>8)&0xff, color&0xff
	return 299*r+587*g+114*b < 128000
}

// The asset names describe artwork color: a dark glyph is legible on a light
// caption, and a light glyph on a dark caption. The static Shell icon is fixed.
func iconGroupForCaption(darkCaption bool) uint16 {
	if darkCaption {
		return appLightIconID
	}
	return appDarkIconID
}

func setDwmAttribute(hwnd uintptr, attribute uint32, value *uint32) bool {
	result, _, _ := dwmSetWindowAttribute.Call(hwnd, uintptr(attribute), ptr(value), 4)
	return int32(result) == 0 // S_OK
}

type captionAPI struct {
	setAttribute     func(hwnd uintptr, attribute uint32, value *uint32) bool
	refreshFrame     func(hwnd uintptr)
	activeBackground func() uint32
}

func resetCaptionColors(api captionAPI, hwnd uintptr) {
	value := uint32(dwmColorDefault)
	api.setAttribute(hwnd, dwmCaptionColor, &value)
	api.setAttribute(hwnd, dwmTextColor, &value)
	api.setAttribute(hwnd, dwmBorderColor, &value)
}

func applyCaption(hwnd uintptr, appearance theme.Resolved) bool {
	if dwmSetWindowAttribute.Find() != nil {
		return false
	}
	return applyCaptionWith(hwnd, appearance, captionAPI{
		setAttribute: setDwmAttribute,
		refreshFrame: refreshNonClientFrame,
		activeBackground: func() uint32 {
			return systemColorRGB(colorActiveCaption)
		},
	})
}

// applyCaptionWith keeps the Windows 10-compatible immersive-dark request
// independent from the Windows 11-only explicit caption-color attributes.
// Every attribute is best-effort and the native system caption remains the
// fallback when DWM rejects the requested appearance.
func applyCaptionWith(hwnd uintptr, appearance theme.Resolved, api captionAPI) bool {
	if appearance.HighContrast {
		light := uint32(0)
		api.setAttribute(hwnd, dwmDarkMode, &light)
		resetCaptionColors(api, hwnd)
		api.refreshFrame(hwnd)
		return darkBackground(api.activeBackground())
	}
	dark := uint32(0)
	if appearance.Dark {
		dark = 1
	}
	immersiveOK := api.setAttribute(hwnd, dwmDarkMode, &dark)

	colors := paletteFor(appearance)
	caption, textColor := rgbToColorRef(uint32(colors.WindowBackground)), rgbToColorRef(uint32(colors.TextPrimary))
	captionOK := api.setAttribute(hwnd, dwmCaptionColor, &caption)
	textOK := api.setAttribute(hwnd, dwmTextColor, &textColor)
	if captionOK && textOK {
		border := rgbToColorRef(uint32(colors.Border))
		api.setAttribute(hwnd, dwmBorderColor, &border)
		api.refreshFrame(hwnd)
		return appearance.Dark
	}
	resetCaptionColors(api, hwnd)
	api.refreshFrame(hwnd)
	if immersiveOK {
		return appearance.Dark
	}
	return darkBackground(api.activeBackground())
}

func refreshNonClientFrame(hwnd uintptr) {
	setWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
	active, _, _ := getForegroundWindow.Call()
	sendMessageW.Call(hwnd, wmNCActivate, 0, 0)
	if active == hwnd {
		sendMessageW.Call(hwnd, wmNCActivate, 1, 0)
	}
}

func (a *application) applyAppearance(w *window) error {
	systemDark := systemAppsDark()
	w.appearance = theme.Resolve(w.mode, systemDark, highContrastEnabled())
	w.colors = paletteFor(w.appearance)
	group := iconGroupForCaption(applyCaption(w.hwnd, w.appearance))
	if w.largeIcon == 0 || w.smallIcon == 0 {
		w.titleIconGroup = group
		if err := a.updateIcons(w); err != nil {
			return err
		}
	} else if group != w.titleIconGroup {
		if err := a.updateTitleIcon(w, group); err != nil {
			return err
		}
	}
	w.surface.fillRGB(uint32(w.colors.WindowBackground))
	if err := w.updateButtonLabel(); err != nil {
		return err
	}
	invalidateRect.Call(w.hwnd, 0, 0)
	return nil
}
