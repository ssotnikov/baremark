// Package theme resolves the effective light or dark appearance without Win32 dependencies.
package theme

type Mode uint8

const (
	System Mode = iota
	Light
	Dark
)

type Resolved struct {
	Dark         bool
	HighContrast bool
}

// Color stores an RGB color as 0xRRGGBB. Platform adapters convert it to the
// native representation at the API boundary.
type Color uint32

// Palette names colors by UI role so renderers do not depend on theme-specific
// literals. Light and dark palettes are designed independently, not generated
// by mechanically inverting one another.
type Palette struct {
	WindowBackground Color
	Surface          Color
	SurfaceRaised    Color
	Button           Color
	ButtonPressed    Color
	Border           Color
	TextPrimary      Color
	TextSecondary    Color
	FocusRing        Color
	Accent           Color
}

func PaletteFor(dark bool) Palette {
	if dark {
		return Palette{
			WindowBackground: 0x1c1b19,
			Surface:          0x242320,
			SurfaceRaised:    0x2c2a27,
			Button:           0x34322e,
			ButtonPressed:    0x48453f,
			Border:           0x625e57,
			TextPrimary:      0xf3f1ec,
			TextSecondary:    0xbab6ad,
			FocusRing:        0x55bfc4,
			Accent:           0x55bfc4,
		}
	}
	return Palette{
		WindowBackground: 0xfbfaf7,
		Surface:          0xf7f5f0,
		SurfaceRaised:    0xffffff,
		Button:           0xf0eee9,
		ButtonPressed:    0xe0ddd5,
		Border:           0xb7b3aa,
		TextPrimary:      0x222524,
		TextSecondary:    0x62645f,
		FocusRing:        0x008f98,
		Accent:           0x008f98,
	}
}

// Resolve keeps the user's preference separate from the effective appearance.
// High Contrast always uses system colors, regardless of the selected mode.
func Resolve(mode Mode, systemDark, highContrast bool) Resolved {
	if highContrast {
		return Resolved{HighContrast: true}
	}
	return Resolved{Dark: mode == Dark || mode == System && systemDark}
}

// Toggle selects an explicit mode even when the previous mode was System.
func Toggle(current Resolved) Mode {
	if current.Dark {
		return Light
	}
	return Dark
}
