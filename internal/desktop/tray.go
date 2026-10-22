package desktop

// TrayMenuAction is the command chosen from the notification-area menu.
type TrayMenuAction uint32

const (
	TrayMenuDismissed TrayMenuAction = iota
	TrayMenuSettings
	TrayMenuQuit
)
