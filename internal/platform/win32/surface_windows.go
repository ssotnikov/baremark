//go:build windows

package win32

import (
	"errors"
	"unsafe"

	"github.com/ssotnikov/baremark/internal/ui/render"
)

type surface struct {
	dc       uintptr
	bitmap   uintptr
	previous uintptr
	pixels   uintptr
	width    int
	height   int
	stride   int
}

func newSurface(width, height int) (*surface, error) {
	count, err := render.SurfacePixels(width, height)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	dc, _, callErr := createCompatibleDC.Call(0)
	if dc == 0 {
		return nil, callFailure("CreateCompatibleDC", callErr)
	}
	var info bitmapInfo
	info.header.size = uint32(unsafe.Sizeof(info.header))
	info.header.width = int32(width)
	info.header.height = -int32(height)
	info.header.planes = 1
	info.header.bitCount = 32
	info.header.compression = biRGB
	var pixels uintptr
	bitmap, _, callErr := createDIBSection.Call(0, ptr(&info), dibRGBColors, ptr(&pixels), 0, 0)
	if bitmap == 0 || pixels == 0 {
		deleteDC.Call(dc)
		return nil, callFailure("CreateDIBSection", callErr)
	}
	previous, _, callErr := selectObject.Call(dc, bitmap)
	if previous == 0 || previous == ^uintptr(0) {
		deleteObject.Call(bitmap)
		deleteDC.Call(dc)
		return nil, callFailure("SelectObject", callErr)
	}
	s := &surface{dc: dc, bitmap: bitmap, previous: previous, pixels: pixels, width: width, height: height, stride: width * 4}
	s.fillRGB(systemColorRGB(colorWindow))
	return s, nil
}

func (s *surface) fillRGB(color uint32) {
	if s == nil {
		return
	}
	// GDI may still be reading the DIB after an earlier BitBlt.
	gdiFlush.Call()
	data := unsafe.Slice((*uint32)(unsafe.Pointer(s.pixels)), s.width*s.height)
	for i := range data {
		data[i] = color
	}
}

func (s *surface) blit(hdc uintptr, dirty rect) error {
	if s == nil {
		return nil
	}
	x, y := max(0, int(dirty.left)), max(0, int(dirty.top))
	right, bottom := min(s.width, int(dirty.right)), min(s.height, int(dirty.bottom))
	if right <= x || bottom <= y {
		return nil
	}
	ok, _, err := bitBlt.Call(hdc, uintptr(x), uintptr(y), uintptr(right-x), uintptr(bottom-y), s.dc, uintptr(x), uintptr(y), srccopy)
	if ok == 0 {
		return callFailure("BitBlt", err)
	}
	return nil
}

func (s *surface) close() error {
	if s == nil {
		return nil
	}
	gdiFlush.Call()
	previous, _, err := selectObject.Call(s.dc, s.previous)
	if previous == 0 || previous == ^uintptr(0) {
		// Delete the DC first so the bitmap is no longer selected anywhere.
		deleteDC.Call(s.dc)
		deleteObject.Call(s.bitmap)
		s.pixels, s.bitmap, s.dc = 0, 0, 0
		return callFailure("restore selected bitmap", err)
	}
	var cleanupErr error
	if ok, _, err := deleteObject.Call(s.bitmap); ok == 0 {
		cleanupErr = callFailure("DeleteObject(HBITMAP)", err)
	}
	if ok, _, err := deleteDC.Call(s.dc); ok == 0 {
		cleanupErr = errors.Join(cleanupErr, callFailure("DeleteDC", err))
	}
	s.pixels = 0
	s.bitmap = 0
	s.dc = 0
	return cleanupErr
}
