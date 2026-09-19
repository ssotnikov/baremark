package render

import "fmt"

const maxSurfaceBytes = 256 << 20

// SurfacePixels validates a 32-bit DIB allocation before native resources are created.
func SurfacePixels(width, height int) (int, error) {
	if width < 0 || height < 0 {
		return 0, fmt.Errorf("negative surface dimensions %dx%d", width, height)
	}
	if width == 0 || height == 0 {
		return 0, nil
	}
	if width > maxSurfaceBytes/4/height {
		return 0, fmt.Errorf("surface %dx%d exceeds the %d-byte limit", width, height, maxSurfaceBytes)
	}
	return width * height, nil
}
