# Countdowner

A lightweight, single-executable countdown widget that lives on your Windows desktop.

## Features

- **Small download, no setup** — about 4.3 MB to download and a 10 MB EXE on x64. Extract and run; no installer or extra runtime to install. ARM64 builds are available too.
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

The tray icon follows the taskbar's light or dark appearance independently of the app theme. Its stable identifier lets Windows remember whether it belongs on the taskbar or in the notification-area overflow. Windows measures and draws the tray menu using the system menu font.

Set a deadline's hour and minute by typing in the numeric fields or using their minus and plus buttons. **Date only** sets the time to 00:00.

If a saved display is temporarily unavailable at login or after waking from sleep, the widget appears on the primary display without overwriting its saved position or size. It returns automatically when the saved display becomes available. Dragging or resizing the widget during this fallback saves its new placement; **Reset desktop position** moves it to the primary display.

Hold the six-dot handle below an event's trash button for 300 ms, then drag vertically to reorder. Only this handle starts reordering; dragging event text or its visualization does not. Custom order is saved; choose **Deadline, earliest first** in Settings to restore automatic sorting, which is the default for new configurations. The trash button to the right of each event starts in monochrome. Click once to turn it red, then click again to confirm deletion; clicking elsewhere in the widget cancels confirmation. Editing remains available in Settings.

Keep the executable in its permanent location before enabling startup at login. Include the accompanying `licenses/` directory when redistributing it. The executable is unsigned and may trigger a Windows SmartScreen warning.

## Build

Requires Go 1.27.3 or a newer 1.27 release.

The MyGo runtime and build tool use the stable release v0.6.2, which includes partial redraws, incremental layout, and Windows focus fixes for native UI. Native library calls use purego v0.11.2. Versions are pinned in `go.mod`, with checksums recorded in `go.sum`.

```sh
go tool mygo build -platform windows/amd64,windows/arm64
```

Translations live in `locales/*.json` and use ICU MessageFormat. After editing them, run `go tool mygo generate` and commit `locales/catalog_gen.go` with the catalogs. Run `go tool mygo i18n check` to validate messages and their arguments.

UI timing tests use MyGo's virtual clock (`Tester.Now` and `Tester.Advance`) to exercise scheduled redraws and the 300 ms reorder threshold without wall-clock waits.

## Data

Settings and countdowns are stored in `%APPDATA%\Countdowner\config.json`; logs are in `%APPDATA%\Countdowner\logs\`. Damaged or unsupported configuration files are preserved, and the application switches to read-only mode.

Deadlines retain their absolute time when the system time zone changes. Display and editing use the current local time zone.

## Limitations

Intended for Windows 10/11. Calendar month and weekday names follow the interface language; date formats and the first day of the week follow Windows regional settings.

The widget requires Windows DirectComposition to blend its rounded edges smoothly with the desktop.

Segoe UI is the primary font in every interface language. Chinese, Japanese, and Korean characters use a fixed system-font fallback order, so switching languages does not change the font of Latin text or digits. No font files are bundled.

Dependency license notices are included in [`resources/licenses/`](resources/licenses/).
