package render

import "testing"

func TestSurfacePixels(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		width, height int
		want          int
		wantError     bool
	}{
		{960, 640, 614400, false},
		{0, 640, 0, false},
		{640, 0, 0, false},
		{-1, 640, 0, true},
		{1 << 20, 1 << 20, 0, true},
	} {
		got, err := SurfacePixels(test.width, test.height)
		if (err != nil) != test.wantError || got != test.want {
			t.Errorf("SurfacePixels(%d, %d) = %d, %v; want %d, error=%t", test.width, test.height, got, err, test.want, test.wantError)
		}
	}
}
