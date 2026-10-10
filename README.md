# Countdowner

A Windows desktop countdown widget built with Go and [MyGo Native UI](https://github.com/egoist/mygo). No WebView2 or additional font installation is required.

## Features

- Multiple countdowns in one draggable, resizable desktop widget.
- Adaptive text and spacing, with scrolling when needed.
- Fixed-size 31-cell day grids and 24-hour urgency rings; events more than 31 days away show text only.
- Light, dark, and system themes; rounded corners and bold countdown values.
- English, Traditional Chinese, Simplified Chinese, Japanese, and Korean.
- Calendar selection, editable hours and minutes, and a date-only option.
- Manual or deadline sorting, saved position and size, and optional startup at login.

The widget stays on the desktop behind ordinary application windows. Remaining days and minutes are rounded up; expired countdowns remain visible.

## Run

Download the ZIP for your architecture from [Releases](https://github.com/Star-Trails/countdowner/releases/latest):

- Intel/AMD 64-bit: `Countdowner-windows-x64.zip`
- ARM64: `Countdowner-windows-arm64.zip`

Extract the ZIP and run `Countdowner.exe`. Keep the included `licenses/` directory. Releases include a `SHA256SUMS.txt` checksum file.

Settings open on first launch. Closing settings leaves the widget running. Use the top-right gear to open Settings, or the tray icon to open Settings or quit. Drag the widget header to move it, or its edges and corners to resize it. The minimum size is 240 × 160 DIP.

Hold the six-dot handle below an event's trash button for 300 ms, then drag vertically to reorder. Only this handle starts reordering; dragging event text or its visualization does not. Custom order is saved; choose **Deadline, earliest first** in Settings to restore automatic sorting, which is the default for new configurations. The trash button to the right of each event starts in monochrome. Click once to turn it red, then click again to confirm deletion; clicking elsewhere in the widget cancels confirmation. Editing remains available in Settings.

Keep the executable in its permanent location before enabling startup at login. Include the accompanying `licenses/` directory when redistributing it. The executable is unsigned and may trigger a Windows SmartScreen warning.

## Build

Requires Go 1.27.1 or a newer 1.27 release.

```sh
go tool mygo build -platform windows/amd64,windows/arm64
```

## Data

Settings and countdowns are stored in `%APPDATA%\Countdowner\config.json`; logs are in `%APPDATA%\Countdowner\logs\`. Damaged or unsupported configuration files are preserved, and the application switches to read-only mode.

Deadlines retain their absolute time when the system time zone changes. Display and editing use the current local time zone.

## Limitations

Intended for Windows 10/11. Calendar month and weekday names currently appear in English.

Bundled fonts and dependency license notices are included in [`resources/licenses/`](resources/licenses/).
