//go:build windows

package desktop

import (
	"runtime"
	"testing"
	"unsafe"

	"countdowner/internal/model"
)

func nativeTestController(t *testing.T) *Controller {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	createWindow := user32.NewProc("CreateWindowExW")
	destroyWindow := user32.NewProc("DestroyWindow")
	class := wide("STATIC")
	host, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(class)), 0, 0x80000000,
		0, 0, 1920, 1080, 0, 0, 0, 0) // Hidden test-owned host; never alter Explorer.
	if host == 0 {
		t.Fatalf("create host: %v", err)
	}
	t.Cleanup(func() { destroyWindow.Call(host) })
	hwnd, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(class)), 0, 0x40000000,
		0, 0, 340, 196, host, 0, 0, 0)
	if hwnd == 0 {
		t.Fatalf("create widget: %v", err)
	}
	return &Controller{hwnd: hwnd, host: host, width: 340, height: 196}
}

func TestMissingMonitorDoesNotReplaceSavedPosition(t *testing.T) {
	c := nativeTestController(t)
	want := model.Position{Monitor: "temporarily-unavailable-monitor", X: 128.5, Y: 76.25}
	saved := want
	c.OnPosition = func(position model.Position) { saved = position }
	if err := c.Restore(want); err != nil {
		t.Fatal(err)
	}
	if saved != want {
		t.Fatalf("temporary fallback replaced saved position: got %+v, want %+v", saved, want)
	}
	if err := c.SetSize(480, 360); err != nil {
		t.Fatal(err)
	}
	if saved != want {
		t.Fatalf("resizing during fallback replaced saved position: %+v", saved)
	}
	var frame rect
	if ok, _, err := getWindowRect.Call(c.hwnd, uintptr(unsafe.Pointer(&frame))); ok == 0 {
		t.Fatal(err)
	}
	t.Logf("missing monitor retained; native fallback window: %+v", frame)
}

func assertNativeFrame(t *testing.T, c *Controller, want rect) {
	t.Helper()
	var got rect
	if ok, _, err := getWindowRect.Call(c.hwnd, uintptr(unsafe.Pointer(&got))); ok == 0 {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("native window bounds = %+v, want %+v", got, want)
	}
}

func TestDisplayReconnectRestoresPreferredBounds(t *testing.T) {
	c := nativeTestController(t)
	// Model a smaller primary and a secondary to its left at 150% scaling.
	// Only enumeration snapshots are synthetic; placement uses real HWNDs.
	primary := monitor{ID: "primary", Work: rect{0, 0, 260, 180}, DPI: 96, Primary: true}
	secondary := monitor{ID: "secondary", Work: rect{-2560, 0, 0, 1440}, DPI: 144}
	want := model.Position{Monitor: secondary.ID, X: 128, Y: 76}
	if err := c.restore(want, []monitor{primary}); err != nil {
		t.Fatal(err)
	}
	assertNativeFrame(t, c, rect{0, 0, 260, 180})
	list := []monitor{primary, secondary}
	if err := c.ensurePlacement(list); err != nil {
		t.Fatal(err)
	}
	assertNativeFrame(t, c, rect{-2368, 114, -1858, 408})
	// Sleep can remove and restore the same display more than once.
	if err := c.ensurePlacement([]monitor{primary}); err != nil {
		t.Fatal(err)
	}
	assertNativeFrame(t, c, rect{0, 0, 260, 180})
	if err := c.ensurePlacement(list); err != nil {
		t.Fatal(err)
	}
	assertNativeFrame(t, c, rect{-2368, 114, -1858, 408})
	t.Log("native window returned to secondary with original coordinates and size after both reconnects")
}

func TestPlacementRepairsExternalWindowMove(t *testing.T) {
	c := nativeTestController(t)
	m := monitor{ID: "primary", Work: rect{0, 0, 1920, 1080}, DPI: 96, Primary: true}
	list := []monitor{m}
	if err := c.restore(model.Position{Monitor: m.ID, X: 128, Y: 76}, list); err != nil {
		t.Fatal(err)
	}
	want := rect{128, 76, 468, 272}
	assertNativeFrame(t, c, want)
	// Simulate Windows moving the child while the display metadata stays the same.
	if ok, _, err := setWindowPos.Call(c.hwnd, 0, 40, 40, 0, 0, 0x1|0x4|0x10); ok == 0 {
		t.Fatal(err)
	}
	assertNativeFrame(t, c, rect{40, 40, 380, 236})
	if err := c.ensurePlacement(list); err != nil {
		t.Fatal(err)
	}
	assertNativeFrame(t, c, want)
}

func TestUserMoveDuringFallbackAdoptsNewDisplay(t *testing.T) {
	c := nativeTestController(t)
	primary := monitor{ID: "primary", Work: rect{0, 0, 1920, 1080}, DPI: 96, Primary: true}
	secondary := monitor{ID: "secondary", Work: rect{-2560, 0, 0, 1440}, DPI: 144}
	if err := c.restore(model.Position{Monitor: secondary.ID, X: 128, Y: 76}, []monitor{primary}); err != nil {
		t.Fatal(err)
	}
	var saved model.Position
	c.OnPosition = func(position model.Position) { saved = position }
	c.place(primary, 240, 160)
	c.dragging = true
	hooks[c.hwnd] = hook{controller: c}
	t.Cleanup(func() { delete(hooks, c.hwnd) })
	windowProcedure(c.hwnd, 0x0202, 0, 0, 0, 0) // Complete the header drag.
	if saved != (model.Position{Monitor: primary.ID, X: 240, Y: 160}) {
		t.Fatalf("user's new placement was not saved: %+v", saved)
	}
	if err := c.ensurePlacement([]monitor{primary, secondary}); err != nil {
		t.Fatal(err)
	}
	assertNativeFrame(t, c, rect{240, 160, 580, 356})
}

func TestResetPlacementSelectsPrimaryDisplay(t *testing.T) {
	c := nativeTestController(t)
	primary := monitor{ID: "primary", Work: rect{0, 0, 1920, 1080}, DPI: 96, Primary: true}
	secondary := monitor{ID: "secondary", Work: rect{-2560, 0, 0, 1440}, DPI: 144}
	list := []monitor{secondary, primary}
	if err := c.restore(model.Position{Monitor: secondary.ID, X: 128, Y: 76}, list); err != nil {
		t.Fatal(err)
	}
	var saved model.Position
	c.OnPosition = func(position model.Position) { saved = position }
	if err := c.restore(model.Position{X: 40, Y: 40}, list); err != nil {
		t.Fatal(err)
	}
	if saved != (model.Position{Monitor: primary.ID, X: 40, Y: 40}) {
		t.Fatalf("reset did not save the primary display: %+v", saved)
	}
	assertNativeFrame(t, c, rect{40, 40, 380, 236})
}
