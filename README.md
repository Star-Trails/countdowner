# Countdowner

A lightweight, single-executable countdown widget that lives on your Windows desktop.

## Features

- **Small download, no setup** — about 5 MB to download and a 12 MB EXE on x64. Extract and run; no installer or extra runtime to install. ARM64 builds are available too.
- **One executable, fully native** — built with Go and [MyGo Native UI](https://github.com/egoist/mygo). No Electron, embedded browser, WebView2, or bundled font files.
- **Desktop, not always-on-top** — stays behind your apps without cluttering the taskbar or Alt+Tab. Return to your countdowns with Win+D.
- **See urgency, not just numbers** — colored day grids turn into progress rings for the final 24 hours. Distant deadlines stay text-only to keep the desktop uncluttered.
- **Your data stays on your PC** — countdowns work offline, with no account or cloud service required. Changing the system time zone does not change a deadline's actual moment.
- **One widget, your layout** — keep multiple deadlines together, resize freely, and drag to reorder or sort by deadline. Position, size, and order are remembered.
- **At home on Windows** — system fonts, light/dark/system themes, and English, Traditional Chinese, Simplified Chinese, Japanese, and Korean interfaces.

Remaining days and minutes are rounded up; expired countdowns remain visible.

## Run

Download the ZIP for your architecture from [Releases](https://github.com/Star-Trails/countdowner/releases/latest):

- Intel/AMD 64-bit: `Countdowner-windows-x64.zip`
- ARM64: `Countdowner-windows-arm64.zip`

Extract the ZIP and run `Countdowner.exe`. Keep the included `licenses/` directory. Releases include a `SHA256SUMS.txt` checksum file.

Settings open on first launch. Closing settings leaves the widget running. Use the top-right gear to open Settings, or the tray icon to open Settings or quit. Drag the widget header to move it, or its edges and corners to resize it. The minimum size is 240 × 160 DIP.

Hold the six-dot handle below an event's trash button for 300 ms, then drag vertically to reorder. Only this handle starts reordering; dragging event text or its visualization does not. Custom order is saved; choose **Deadline, earliest first** in Settings to restore automatic sorting, which is the default for new configurations. The trash button to the right of each event starts in monochrome. Click once to turn it red, then click again to confirm deletion; clicking elsewhere in the widget cancels confirmation. Editing remains available in Settings.

Keep the executable in its permanent location before enabling startup at login. Include the accompanying `licenses/` directory when redistributing it. The executable is unsigned and may trigger a Windows SmartScreen warning.

## Build

Requires Go 1.27.2 or a newer 1.27 release.

```sh
go tool mygo build -platform windows/amd64,windows/arm64
```

## Data

Settings and countdowns are stored in `%APPDATA%\Countdowner\config.json`; logs are in `%APPDATA%\Countdowner\logs\`. Damaged or unsupported configuration files are preserved, and the application switches to read-only mode.

Deadlines retain their absolute time when the system time zone changes. Display and editing use the current local time zone.

## Limitations

Intended for Windows 10/11. Calendar month and weekday names currently appear in English.

Windows system fonts are selected for each interface language, with native fallback for other characters. No font files are bundled.

Dependency license notices are included in [`resources/licenses/`](resources/licenses/).
