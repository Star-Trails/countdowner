//go:build windows

package desktop

import (
	"fmt"
	"unsafe"
)

var (
	createTrayMenu     = user32.NewProc("CreatePopupMenu")
	appendTrayItem     = user32.NewProc("AppendMenuW")
	destroyTrayMenu    = user32.NewProc("DestroyMenu")
	trackTrayMenu      = user32.NewProc("TrackPopupMenuEx")
	foregroundTrayMenu = user32.NewProc("SetForegroundWindow")
	postTrayMessage    = user32.NewProc("PostMessageW")
)

// PopupTrayMenu uses Windows' own menu layout and font metrics. The caller
// owns the top-level window and dispatches the returned command.
// Call it on MyGo's main thread, as with the other desktop operations.
func PopupTrayMenu(owner uintptr, settings, quit string) (TrayMenuAction, error) {
	menu, _, err := createTrayMenu.Call()
	if menu == 0 {
		return TrayMenuDismissed, fmt.Errorf("CreatePopupMenu: %w", err)
	}
	defer destroyTrayMenu.Call(menu)
	if ok, _, err := appendTrayItem.Call(menu, 0, uintptr(TrayMenuSettings), uintptr(unsafe.Pointer(wide(settings)))); ok == 0 {
		return TrayMenuDismissed, fmt.Errorf("AppendMenuW: %w", err)
	}
	if ok, _, err := appendTrayItem.Call(menu, 0x0800, 0, 0); ok == 0 { // MF_SEPARATOR
		return TrayMenuDismissed, fmt.Errorf("AppendMenuW: %w", err)
	}
	if ok, _, err := appendTrayItem.Call(menu, 0, uintptr(TrayMenuQuit), uintptr(unsafe.Pointer(wide(quit)))); ok == 0 {
		return TrayMenuDismissed, fmt.Errorf("AppendMenuW: %w", err)
	}
	var cursor point
	if ok, _, err := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
		return TrayMenuDismissed, fmt.Errorf("GetCursorPos: %w", err)
	}
	// A foreground owner and the final WM_NULL make clicking away dismiss a
	// notification-area menu reliably, as required by TrackPopupMenuEx.
	foregroundTrayMenu.Call(owner)
	command, _, _ := trackTrayMenu.Call(menu, 0x0102, uintptr(cursor.X), uintptr(cursor.Y), owner, 0) // TPM_RETURNCMD | TPM_RIGHTBUTTON
	postTrayMessage.Call(owner, 0, 0, 0)
	return TrayMenuAction(command), nil
}
