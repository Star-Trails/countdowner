//go:build windows

// Package desktop embeds an app-owned window into Explorer's desktop icon host.
// Explorer's Progman/WorkerW hierarchy is not a public Windows API.
package desktop

import (
	"errors"
	"fmt"
	"math"
	"syscall"
	"unsafe"

	"countdowner/internal/model"
)

const CornerRadius = 8 // Fluent surface corners, in device-independent pixels.

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	gdi32               = syscall.NewLazyDLL("gdi32.dll")
	comctl32            = syscall.NewLazyDLL("comctl32.dll")
	shcore              = syscall.NewLazyDLL("shcore.dll")
	findWindow          = user32.NewProc("FindWindowW")
	findWindowEx        = user32.NewProc("FindWindowExW")
	enumWindows         = user32.NewProc("EnumWindows")
	getParent           = user32.NewProc("GetParent")
	setParent           = user32.NewProc("SetParent")
	isWindow            = user32.NewProc("IsWindow")
	getWindowLong       = user32.NewProc("GetWindowLongPtrW")
	setWindowLong       = user32.NewProc("SetWindowLongPtrW")
	setWindowPos        = user32.NewProc("SetWindowPos")
	getWindowRect       = user32.NewProc("GetWindowRect")
	screenToClient      = user32.NewProc("ScreenToClient")
	getCursorPos        = user32.NewProc("GetCursorPos")
	setCapture          = user32.NewProc("SetCapture")
	releaseCapture      = user32.NewProc("ReleaseCapture")
	loadCursor          = user32.NewProc("LoadCursorW")
	setCursor           = user32.NewProc("SetCursor")
	showWindow          = user32.NewProc("ShowWindow")
	getDPIForWindow     = user32.NewProc("GetDpiForWindow")
	enumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	getMonitorInfo      = user32.NewProc("GetMonitorInfoW")
	enumDisplayDevices  = user32.NewProc("EnumDisplayDevicesW")
	getDPIForMonitor    = shcore.NewProc("GetDpiForMonitor")
	createRoundRectRgn  = gdi32.NewProc("CreateRoundRectRgn")
	deleteObject        = gdi32.NewProc("DeleteObject")
	setWindowRgn        = user32.NewProc("SetWindowRgn")
	setWindowSubclass   = comctl32.NewProc("SetWindowSubclass")
	defSubclassProc     = comctl32.NewProc("DefSubclassProc")
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type monitorInfo struct {
	Size         uint32
	Bounds, Work rect
	Flags        uint32
	Device       [32]uint16
}
type displayDevice struct {
	Size        uint32
	Name        [32]uint16
	Description [128]uint16
	Flags       uint32
	ID, Key     [128]uint16
}
type monitor struct {
	ID      string
	Work    rect
	DPI     uint32
	Primary bool
}

// Controller is confined to MyGo's main thread, including all callbacks.
// Its placement uses physical screen coordinates at the Win32 boundary and
// monitor-relative DIPs in the saved configuration.
type Controller struct {
	hwnd, surface, host       uintptr
	width, height             int
	position                  model.Position
	monitor                   monitor
	visualScale               float32
	dragging, placing, hidden bool
	dragX, dragY              float64
	resizing                  resizeEdge
	resizeStart               bounds
	cursorStart               point
	OnResize                  func(model.Position, model.Size)
	OnPosition                func(model.Position)
	OnRedraw                  func()
}

type hook struct {
	controller *Controller
	surface    bool
}

var (
	hooks          = map[uintptr]hook{}
	desktopHost    uintptr
	monitorList    []monitor
	windowCallback = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		view, _, _ := findWindowEx.Call(hwnd, 0, uintptr(unsafe.Pointer(wide("SHELLDLL_DefView"))), 0)
		if view != 0 {
			desktopHost = hwnd
			return 0
		}
		return 1
	})
	monitorCallback = syscall.NewCallback(func(hmon, _, _, _ uintptr) uintptr {
		var info monitorInfo
		info.Size = uint32(unsafe.Sizeof(info))
		if ok, _, _ := getMonitorInfo.Call(hmon, uintptr(unsafe.Pointer(&info))); ok == 0 {
			return 1
		}
		var device displayDevice
		device.Size = uint32(unsafe.Sizeof(device))
		id := syscall.UTF16ToString(info.Device[:])
		// EDD_GET_DEVICE_INTERFACE_NAME gives a monitor identity rather than
		// just DISPLAY1, which can change after displays are rearranged.
		if ok, _, _ := enumDisplayDevices.Call(uintptr(unsafe.Pointer(&info.Device[0])), 0, uintptr(unsafe.Pointer(&device)), 1); ok != 0 {
			if hardwareID := syscall.UTF16ToString(device.ID[:]); hardwareID != "" {
				id = hardwareID
			}
		}
		dpi, y := uint32(96), uint32(96)
		if result, _, _ := getDPIForMonitor.Call(hmon, 0, uintptr(unsafe.Pointer(&dpi)), uintptr(unsafe.Pointer(&y))); result != 0 || dpi == 0 {
			dpi = 96
		}
		monitorList = append(monitorList, monitor{ID: id, Work: info.Work, DPI: dpi, Primary: info.Flags&1 != 0})
		return 1
	})
	subclassCallback = syscall.NewCallback(windowProcedure)
)

func wide(value string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(value)
	return p
}

func findHost() uintptr {
	// On current Windows 11 builds DefView can be a child of Progman or
	// a top-level WorkerW. Embed as its sibling ABOVE it, not into the
	// wallpaper WorkerW behind it (which would intercept pointer input).
	progman, _, _ := findWindow.Call(uintptr(unsafe.Pointer(wide("Progman"))), 0)
	if progman != 0 {
		view, _, _ := findWindowEx.Call(progman, 0, uintptr(unsafe.Pointer(wide("SHELLDLL_DefView"))), 0)
		if view != 0 {
			return progman
		}
		for child := uintptr(0); ; {
			child, _, _ = findWindowEx.Call(progman, child, uintptr(unsafe.Pointer(wide("WorkerW"))), 0)
			if child == 0 {
				break
			}
			view, _, _ := findWindowEx.Call(child, 0, uintptr(unsafe.Pointer(wide("SHELLDLL_DefView"))), 0)
			if view != 0 {
				return child
			}
		}
	}
	desktopHost = 0
	enumWindows.Call(windowCallback, 0)
	return desktopHost
}

func Available() bool { return findHost() != 0 }

func monitors() []monitor {
	monitorList = nil
	enumDisplayMonitors.Call(0, 0, monitorCallback, 0)
	return monitorList
}

func (c *Controller) Attach(hwnd uintptr, position model.Position, width, height int) error {
	c.hwnd, c.position, c.width, c.height = hwnd, position, width, height
	c.visualScale = 1
	host := findHost()
	if host == 0 {
		return errors.New("未找到 Explorer 桌面图标宿主；Widget 保持隐藏")
	}
	if err := c.embed(host); err != nil {
		return err
	}
	c.surface, _, _ = findWindowEx.Call(hwnd, 0, uintptr(unsafe.Pointer(wide("MyGoSurface"))), 0)
	if c.surface == 0 {
		return errors.New("未找到 MyGo Native UI surface")
	}
	for _, entry := range []struct {
		hwnd    uintptr
		surface bool
	}{{hwnd, false}, {c.surface, true}} {
		if ok, _, err := setWindowSubclass.Call(entry.hwnd, subclassCallback, 1, 0); ok == 0 {
			return fmt.Errorf("安装桌面窗口事件处理：%w", err)
		}
		hooks[entry.hwnd] = hook{c, entry.surface}
	}
	if err := c.Restore(position); err != nil {
		return err
	}
	showWindow.Call(hwnd, 4) // SW_SHOWNOACTIVATE; never foreground the desktop.
	return nil
}

func (c *Controller) embed(host uintptr) error {
	const (
		gwlStyle       = ^uintptr(15) // -16
		gwlExStyle     = ^uintptr(19) // -20
		wsChild        = 0x40000000
		wsPopup        = 0x80000000
		wsCaption      = 0x00C00000
		wsThickFrame   = 0x00040000
		wsExToolWindow = 0x80
		wsExNoActivate = 0x08000000
		wsExAppWindow  = 0x40000
	)
	style, _, _ := getWindowLong.Call(c.hwnd, gwlStyle)
	ex, _, _ := getWindowLong.Call(c.hwnd, gwlExStyle)
	setWindowLong.Call(c.hwnd, gwlStyle, (style|wsChild)&^(wsPopup|wsCaption|wsThickFrame))
	setWindowLong.Call(c.hwnd, gwlExStyle, (ex|wsExToolWindow|wsExNoActivate)&^wsExAppWindow)
	_, _, err := setParent.Call(c.hwnd, host)
	parent, _, _ := getParent.Call(c.hwnd)
	if parent != host {
		return fmt.Errorf("桌面嵌入失败（可能是 DPI awareness 不兼容）：%w", err)
	}
	c.host = host
	// HWND_TOP is only the top of Explorer's CHILD hierarchy, not TOPMOST.
	setWindowPos.Call(c.hwnd, 0, 0, 0, 0, 0, 0x1|0x2|0x10|0x20)
	return nil
}

// Ensure repairs embedding only when the desktop host changes. It does not
// periodically manipulate z-order or steal focus.
func (c *Controller) Ensure() error {
	if ok, _, _ := isWindow.Call(c.hwnd); ok == 0 {
		return errors.New("桌面窗口已销毁，等待重建")
	}
	host := findHost()
	if host == 0 {
		showWindow.Call(c.hwnd, 0)
		c.hidden = true
		return errors.New("Explorer 桌面尚未就绪")
	}
	parent, _, _ := getParent.Call(c.hwnd)
	if parent != host {
		if err := c.embed(host); err != nil {
			showWindow.Call(c.hwnd, 0)
			c.hidden = true
			return err
		}
		if err := c.Restore(c.position); err != nil {
			return err
		}
		showWindow.Call(c.hwnd, 4)
		c.hidden = false
	}
	if c.hidden {
		showWindow.Call(c.hwnd, 4)
		c.hidden = false
	}
	if !c.dragging && c.resizing == 0 {
		windowDPI, _, _ := getDPIForWindow.Call(c.hwnd)
		for _, m := range monitors() {
			if m.ID == c.monitor.ID && m.DPI == c.monitor.DPI && m.Work == c.monitor.Work &&
				windowDPI != 0 && c.visualScale == float32(m.DPI)/float32(windowDPI) {
				return nil
			}
		}
		return c.Restore(c.position)
	}
	return nil
}

func (c *Controller) Restore(position model.Position) error {
	list := monitors()
	if len(list) == 0 {
		return errors.New("没有可用显示器")
	}
	chosen := list[0]
	for _, m := range list {
		if m.Primary {
			chosen = m
		}
	}
	for _, m := range list {
		if m.ID == position.Monitor {
			chosen = m
			break
		}
	}
	c.place(chosen, float64(chosen.Work.Left)+position.X*float64(chosen.DPI)/96,
		float64(chosen.Work.Top)+position.Y*float64(chosen.DPI)/96)
	c.reportPosition()
	return nil
}

func (c *Controller) SetSize(width, height int) error {
	c.width, c.height = max(MinWidth, width), max(MinHeight, height)
	return c.Restore(c.position)
}

func (c *Controller) Size() model.Size {
	return model.Size{Width: c.width, Height: c.height}
}

// VisualScale compensates for Explorer child windows inheriting their host's
// DPI. Native UI layout and font metrics are scaled before rasterization, not
// by stretching a bitmap. Settings remain a normal per-monitor-v2 window.
func (c *Controller) VisualScale() float32 {
	if c.visualScale <= 0 {
		return 1
	}
	return c.visualScale
}

func (c *Controller) place(m monitor, x, y float64) {
	if c.placing {
		return
	}
	c.placing = true
	defer func() { c.placing = false }()
	dpi := float64(m.DPI) / 96
	w := min(int32(math.Round(float64(c.width)*dpi)), m.Work.Right-m.Work.Left)
	h := min(int32(math.Round(float64(c.height)*dpi)), m.Work.Bottom-m.Work.Top)
	x = math.Max(float64(m.Work.Left), math.Min(x, float64(m.Work.Right-w)))
	y = math.Max(float64(m.Work.Top), math.Min(y, float64(m.Work.Bottom-h)))
	pt := point{int32(math.Round(x)), int32(math.Round(y))}
	screenToClient.Call(c.host, uintptr(unsafe.Pointer(&pt)))
	setWindowPos.Call(c.hwnd, 0, uintptr(pt.X), uintptr(pt.Y), uintptr(w), uintptr(h), 0x4|0x10)
	c.monitor = m
	c.width, c.height = int(math.Round(float64(w)/dpi)), int(math.Round(float64(h)/dpi))
	c.position = model.Position{Monitor: m.ID, X: (x - float64(m.Work.Left)) / dpi, Y: (y - float64(m.Work.Top)) / dpi}
	windowDPI, _, _ := getDPIForWindow.Call(c.hwnd)
	if windowDPI == 0 {
		windowDPI = 96
	}
	c.visualScale = float32(m.DPI) / float32(windowDPI)
	radius := int32(math.Round(CornerRadius * dpi))
	rgn, _, _ := createRoundRectRgn.Call(0, 0, uintptr(w+1), uintptr(h+1), uintptr(2*radius), uintptr(2*radius))
	if rgn != 0 {
		if ok, _, _ := setWindowRgn.Call(c.hwnd, rgn, 1); ok == 0 {
			deleteObject.Call(rgn) // On success Windows owns the region.
		}
	}
	if c.OnRedraw != nil {
		c.OnRedraw()
	}
}

func (c *Controller) reportPosition() {
	if c.OnPosition != nil {
		c.OnPosition(c.position)
	}
}

func (c *Controller) dragMove() {
	var cursor point
	getCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	chosen := c.monitor
	for _, m := range monitors() {
		if cursor.X >= m.Work.Left && cursor.X < m.Work.Right && cursor.Y >= m.Work.Top && cursor.Y < m.Work.Bottom {
			chosen = m
			break
		}
	}
	dpi := float64(chosen.DPI) / 96
	c.place(chosen, float64(cursor.X)-c.dragX*dpi, float64(cursor.Y)-c.dragY*dpi)
}

func (c *Controller) pointerBounds() (point, rect) {
	var cursor point
	var frame rect
	getCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	getWindowRect.Call(c.hwnd, uintptr(unsafe.Pointer(&frame)))
	return cursor, frame
}

func (c *Controller) edgeAtPointer(cursor point, frame rect) resizeEdge {
	dpi := float64(c.monitor.DPI) / 96
	return hitResizeEdge(float64(cursor.X-frame.Left)/dpi, float64(cursor.Y-frame.Top)/dpi,
		float64(frame.Right-frame.Left)/dpi, float64(frame.Bottom-frame.Top)/dpi)
}

func setResizeCursor(edge resizeEdge) {
	var id uintptr
	switch edge {
	case edgeLeft | edgeTop, edgeRight | edgeBottom:
		id = 32642 // IDC_SIZENWSE
	case edgeRight | edgeTop, edgeLeft | edgeBottom:
		id = 32643 // IDC_SIZENESW
	case edgeLeft, edgeRight:
		id = 32644 // IDC_SIZEWE
	case edgeTop, edgeBottom:
		id = 32645 // IDC_SIZENS
	}
	if id != 0 {
		cursor, _, _ := loadCursor.Call(0, id)
		setCursor.Call(cursor)
	}
}

func (c *Controller) resizeMove() {
	cursor, _ := c.pointerBounds()
	work := c.monitor.Work
	area := bounds{float64(work.Left), float64(work.Top), float64(work.Right), float64(work.Bottom)}
	dpi := float64(c.monitor.DPI) / 96
	next := resizedBounds(c.resizeStart, area, c.resizing,
		float64(cursor.X-c.cursorStart.X), float64(cursor.Y-c.cursorStart.Y),
		math.Min(MinWidth*dpi, area.right-area.left), math.Min(MinHeight*dpi, area.bottom-area.top))
	c.width = int(math.Round((next.right - next.left) / dpi))
	c.height = int(math.Round((next.bottom - next.top) / dpi))
	c.place(c.monitor, next.left, next.top)
	setResizeCursor(c.resizing)
}

func windowProcedure(hwnd, message, wp, lp, _, _ uintptr) uintptr {
	entry, found := hooks[hwnd]
	if found {
		c := entry.controller
		switch message {
		case 0x0021: // WM_MOUSEACTIVATE: do not activate Explorer or the app.
			return 3 // MA_NOACTIVATE, not MA_NOACTIVATEANDEAT.
		case 0x0020: // WM_SETCURSOR; border hit zones are six DIPs wide.
			cursor, frame := c.pointerBounds()
			edge := c.resizing
			if edge == 0 {
				edge = c.edgeAtPointer(cursor, frame)
			}
			if edge != 0 {
				setResizeCursor(edge)
				return 1
			}
		case 0x0201: // WM_LBUTTONDOWN; resize takes precedence over header drag.
			cursor, frame := c.pointerBounds()
			if edge := c.edgeAtPointer(cursor, frame); edge != 0 {
				c.resizing, c.cursorStart = edge, cursor
				c.resizeStart = bounds{float64(frame.Left), float64(frame.Top), float64(frame.Right), float64(frame.Bottom)}
				setCapture.Call(hwnd)
				return 0
			}
			if entry.surface {
				dpi := float64(c.monitor.DPI) / 96
				if float64(cursor.Y-frame.Top)/dpi < 58 {
					c.dragging = true
					c.dragX, c.dragY = float64(cursor.X-frame.Left)/dpi, float64(cursor.Y-frame.Top)/dpi
					setCapture.Call(hwnd)
					return 0
				}
			}
		case 0x0200: // WM_MOUSEMOVE
			if c.resizing != 0 {
				c.resizeMove()
				return 0
			}
			if entry.surface && c.dragging {
				c.dragMove()
				return 0
			}
		case 0x0202, 0x0215: // WM_LBUTTONUP / WM_CAPTURECHANGED
			if c.resizing != 0 {
				c.resizing = 0
				releaseCapture.Call()
				if c.OnResize != nil {
					c.OnResize(c.position, c.Size())
				}
				return 0
			}
			if c.dragging {
				c.dragging = false
				releaseCapture.Call()
				c.reportPosition()
				return 0
			}
		case 0x02E0, 0x02E3: // WM_DPICHANGED / WM_DPICHANGED_AFTERPARENT.
			// MyGo's top-level screen rect is wrong for an Explorer child.
			if !entry.surface {
				if !c.placing && !c.dragging && c.resizing == 0 {
					_ = c.Restore(c.position)
				}
				return 0
			}
		case 0x0082: // WM_NCDESTROY; Explorer may destroy the cross-process child.
			delete(hooks, hwnd)
		}
	}
	result, _, _ := defSubclassProc.Call(hwnd, message, wp, lp)
	return result
}
