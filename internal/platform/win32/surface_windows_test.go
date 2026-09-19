//go:build windows

package win32

import (
	"runtime"
	"testing"
)

func TestSurfaceLifecycleDoesNotGrowGDIHandles(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	process, _, _ := getCurrentProcess.Call()
	baseline, _, _ := getGuiResources.Call(process, 0) // GR_GDIOBJECTS; zero is valid.
	for i := 0; i < 50; i++ {
		s, err := newSurface(320+i*4, 240+i*3)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			openCount, _, _ := getGuiResources.Call(process, 0)
			if openCount < baseline+2 {
				t.Fatalf("GetGuiResources did not count the live DC and bitmap: before=%d, open=%d", baseline, openCount)
			}
		}
		if err := s.close(); err != nil {
			t.Fatal(err)
		}
	}
	after, _, _ := getGuiResources.Call(process, 0)
	if after > baseline+1 {
		t.Fatalf("GDI objects grew from %d to %d", baseline, after)
	}
}
