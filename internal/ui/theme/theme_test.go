package theme

import (
	"math"
	"testing"
)

func TestResolve(t *testing.T) {
	for _, test := range []struct {
		name         string
		mode         Mode
		systemDark   bool
		highContrast bool
		want         Resolved
	}{
		{"system light", System, false, false, Resolved{}},
		{"system dark", System, true, false, Resolved{Dark: true}},
		{"forced light", Light, true, false, Resolved{}},
		{"forced dark", Dark, false, false, Resolved{Dark: true}},
		{"high contrast overrides dark", Dark, false, true, Resolved{HighContrast: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Resolve(test.mode, test.systemDark, test.highContrast); got != test.want {
				t.Fatalf("Resolve() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestToggle(t *testing.T) {
	if got := Toggle(Resolved{}); got != Dark {
		t.Fatalf("Toggle(light) = %v, want Dark", got)
	}
	if got := Toggle(Resolved{Dark: true}); got != Light {
		t.Fatalf("Toggle(dark) = %v, want Light", got)
	}
}

func TestPalettesAreIndependentAndReadable(t *testing.T) {
	light, dark := PaletteFor(false), PaletteFor(true)
	if light == dark {
		t.Fatal("light and dark themes must use independent palettes")
	}
	if light != PaletteFor(false) || dark != PaletteFor(true) {
		t.Fatal("PaletteFor must be deterministic")
	}
	for _, test := range []struct {
		name       string
		foreground Color
		background Color
		minimum    float64
	}{
		{"light primary text", light.TextPrimary, light.WindowBackground, 7},
		{"dark primary text", dark.TextPrimary, dark.WindowBackground, 7},
		{"light button text", light.TextPrimary, light.Button, 4.5},
		{"dark button text", dark.TextPrimary, dark.Button, 4.5},
		{"light secondary text", light.TextSecondary, light.WindowBackground, 4.5},
		{"dark secondary text", dark.TextSecondary, dark.WindowBackground, 4.5},
	} {
		if ratio := contrastRatio(test.foreground, test.background); ratio < test.minimum {
			t.Errorf("%s contrast = %.2f, want at least %.2f", test.name, ratio, test.minimum)
		}
	}
}

func contrastRatio(foreground, background Color) float64 {
	first, second := relativeLuminance(foreground), relativeLuminance(background)
	if first < second {
		first, second = second, first
	}
	return (first + 0.05) / (second + 0.05)
}

func relativeLuminance(color Color) float64 {
	channel := func(value uint32) float64 {
		component := float64(value) / 255
		if component <= 0.04045 {
			return component / 12.92
		}
		return math.Pow((component+0.055)/1.055, 2.4)
	}
	value := uint32(color)
	return 0.2126*channel(value>>16&0xff) + 0.7152*channel(value>>8&0xff) + 0.0722*channel(value&0xff)
}
