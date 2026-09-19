//go:build windows

package win32

import (
	"testing"
	"unsafe"

	"github.com/ssotnikov/baremark/internal/ui/theme"
)

func TestWindowsMessageStructLayout(t *testing.T) {
	for _, test := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"WNDCLASSEXW", unsafe.Sizeof(windowClass{}), 80},
		{"MSG", unsafe.Sizeof(message{}), 48},
		{"PAINTSTRUCT", unsafe.Sizeof(paintStruct{}), 72},
		{"BITMAPINFO", unsafe.Sizeof(bitmapInfo{}), 44},
		{"DRAWITEMSTRUCT", unsafe.Sizeof(drawItemStruct{}), 64},
		{"HIGHCONTRASTW", unsafe.Sizeof(highContrast{}), 16},
		{"LOGFONTW", unsafe.Sizeof(logFont{}), 92},
	} {
		if test.got != test.want {
			t.Errorf("%s size = %d, want %d", test.name, test.got, test.want)
		}
	}
}

func TestCaptionIconContrast(t *testing.T) {
	if got := iconGroupForCaption(false); got != appDarkIconID {
		t.Fatalf("light caption icon group = %d, want %d", got, appDarkIconID)
	}
	if got := iconGroupForCaption(true); got != appLightIconID {
		t.Fatalf("dark caption icon group = %d, want %d", got, appLightIconID)
	}
}

func TestThemeButtonLabel(t *testing.T) {
	for _, test := range []struct {
		mode       theme.Mode
		appearance theme.Resolved
		want       string
	}{
		{theme.System, theme.Resolved{}, "Switch to dark theme"},
		{theme.System, theme.Resolved{Dark: true}, "Switch to light theme"},
		{theme.Dark, theme.Resolved{}, "Switch to light theme"},
		{theme.Light, theme.Resolved{Dark: true}, "Switch to dark theme"},
	} {
		w := &window{mode: test.mode, appearance: test.appearance}
		if got := buttonLabel(w); got != test.want {
			t.Errorf("buttonLabel(%v, %+v) = %q, want %q", test.mode, test.appearance, got, test.want)
		}
	}
}

func TestColorRefRoundTrip(t *testing.T) {
	if got := colorRefToRGB(rgbToColorRef(0x123456)); got != 0x123456 {
		t.Fatalf("color conversion = %#x, want %#x", got, 0x123456)
	}
}

func TestCaptionBackgroundContrast(t *testing.T) {
	if !darkBackground(0x1c1b19) || darkBackground(0xfbfaf7) {
		t.Fatal("caption background classification would select an illegible icon")
	}
}

func TestCaptionUsesImmersiveDarkModeWhenExplicitColorsAreUnavailable(t *testing.T) {
	type call struct {
		attribute uint32
		value     uint32
	}
	var calls []call
	refreshes := 0
	api := captionAPI{
		setAttribute: func(_ uintptr, attribute uint32, value *uint32) bool {
			calls = append(calls, call{attribute, *value})
			return attribute == dwmDarkMode
		},
		refreshFrame: func(uintptr) { refreshes++ },
		activeBackground: func() uint32 {
			return 0xffffff
		},
	}
	if dark := applyCaptionWith(1, theme.Resolved{Dark: true}, api); !dark {
		t.Fatal("successful immersive-dark request must select the light caption artwork")
	}
	if len(calls) == 0 || calls[0] != (call{dwmDarkMode, 1}) {
		t.Fatalf("first DWM call = %+v, want immersive dark mode enabled", calls)
	}
	if refreshes != 1 {
		t.Fatalf("non-client refreshes = %d, want 1", refreshes)
	}
}

func TestCaptionFallsBackToSystemBackgroundWhenDWMRejectsTheme(t *testing.T) {
	api := captionAPI{
		setAttribute: func(uintptr, uint32, *uint32) bool { return false },
		refreshFrame: func(uintptr) {},
		activeBackground: func() uint32 {
			return 0xffffff
		},
	}
	if dark := applyCaptionWith(1, theme.Resolved{Dark: true}, api); dark {
		t.Fatal("rejected DWM attributes must use the actual light system caption")
	}
}

func TestHighContrastDisablesImmersiveTheme(t *testing.T) {
	var firstAttribute, firstValue uint32
	api := captionAPI{
		setAttribute: func(_ uintptr, attribute uint32, value *uint32) bool {
			if firstAttribute == 0 {
				firstAttribute, firstValue = attribute, *value
			}
			return true
		},
		refreshFrame: func(uintptr) {},
		activeBackground: func() uint32 {
			return 0x000000
		},
	}
	if dark := applyCaptionWith(1, theme.Resolved{HighContrast: true}, api); !dark {
		t.Fatal("dark High Contrast caption must select contrasting light artwork")
	}
	if firstAttribute != dwmDarkMode || firstValue != 0 {
		t.Fatalf("first DWM call = attribute %d value %d, want immersive mode disabled", firstAttribute, firstValue)
	}
}

func TestScaleDIP(t *testing.T) {
	for _, test := range []struct {
		dpi  uint32
		want int32
	}{
		{96, 480},
		{120, 600},
		{144, 720},
		{192, 960},
	} {
		if got := scaleDIP(480, test.dpi); got != test.want {
			t.Errorf("scaleDIP(480, %d) = %d, want %d", test.dpi, got, test.want)
		}
	}
}
