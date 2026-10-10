# Countdowner

A Windows desktop countdown widget built with Go and [MyGo Native UI](https://github.com/egoist/mygo). No WebView2 or additional font installation is required.

## Features

- Multiple countdowns in one draggable, resizable desktop widget.
- Adaptive text and spacing, with scrolling when needed.
- Light, dark, and system themes; rounded corners and bold countdown values.
- English, Traditional Chinese, Simplified Chinese, Japanese, and Korean.
- Calendar selection, editable hours and minutes, and a date-only option.
- Manual or deadline sorting, saved position and size, and optional startup at login.

The widget stays on the desktop behind ordinary application windows. Remaining days and minutes are rounded up; expired countdowns remain visible.

## Run

Copy the executable for your architecture to a permanent Windows folder, such as `C:\Apps\Countdowner`, and run it:

- x64: `build/windows-amd64/Countdowner.exe`
- ARM64: `build/windows-arm64/Countdowner.exe`

Settings open on first launch. Closing settings leaves the widget running. Use the tray icon to reopen settings or quit. Drag the widget header to move it, or its edges and corners to resize it. The minimum size is 240 × 160 DIP.

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

Targets Windows 11. Calendar month and weekday names currently appear in English.

Bundled fonts and dependency license notices are included in [`resources/licenses/`](resources/licenses/).
