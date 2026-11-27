# Countdowner

A countdown widget for the desktop, built in Rust with [Native Shell](https://github.com/longbridge/nsh).

## Features

- **Native rendering** on Windows (Win32, Direct2D and DirectWrite), macOS (AppKit), and Linux (GTK 4).
- **Desktop widget and separate settings window.** The desktop environment determines window layering and tray support.
- **See urgency, not just numbers** — colored day grids turn into progress rings for the final 24 hours. Distant deadlines stay text-only to keep the desktop uncluttered.
- **Your data stays on your PC** — countdowns work offline, with no account or cloud service required. Changing the system time zone does not change a deadline's actual moment.
- **One widget, your layout** — keep multiple deadlines together, resize freely, and drag to reorder or sort by deadline. Position, size, and order are remembered.
- **Matched interface typography and system themes**, with English, Traditional Chinese, Simplified Chinese, Japanese and Korean interfaces.

Remaining days and minutes are rounded up; expired countdowns remain visible.

## Run

Download the ZIP for your architecture from [Releases](https://github.com/Star-Trails/countdowner/releases/latest):

- Intel/AMD 64-bit: `Countdowner-windows-x64.zip`
- ARM64: `Countdowner-windows-arm64.zip`

Extract the ZIP and run `Countdowner.exe`. Keep the included `licenses/` directory. Releases include a `SHA256SUMS.txt` checksum file.

Settings open when there are no countdowns. Closing settings leaves the widget running. Use the widget or tray to open Settings, and the tray or Settings to quit. Drag the header to move the widget or its edges and corners to resize it. The minimum size is 240 × 160 points where the window manager honours it.

Set a deadline's hour and minute by typing in the numeric fields or using their minus and plus buttons. **Date only** sets the time to 00:00.

NSH remembers widget bounds on supported desktops. **Reset desktop position** requests a move to the primary display.

Hold the six-dot handle below an event's trash button for 300 ms, then drag vertically to reorder. Only this handle starts reordering; dragging event text or its visualization does not. Custom order is saved; choose **Deadline, earliest first** in Settings to restore automatic sorting, which is the default for new configurations. The trash button to the right of each event starts in monochrome. Click once to turn it red, then click again to confirm deletion; clicking elsewhere in the widget cancels confirmation. Editing remains available in Settings.

Keep the executable in its permanent location before enabling startup at login. Include the accompanying `licenses/` directory when redistributing it. The executable is unsigned and may trigger a Windows SmartScreen warning.

## Development

Use Rust 1.97 or later and the platform build dependencies in the [NSH installation guide](https://nsh.dev/docs/installation). Linux needs GTK 4.14 or later to run its native backend. The headless tests run without GTK or a desktop session.

```sh
cargo test --locked --lib
cargo run --locked --features native
cargo build --locked --release --features native
```

The same Cargo feature selects the host's native backend on Windows, macOS and Linux. With the matching NSH CLI installed, `nsh run --target windows`, `nsh run --target macos` and `nsh run --target linux` mount the same Rust app on their respective hosts. See [the NSH migration memo](NSH-MEMO.md) for desktop backend differences.

Translations live in `locales/*.json`. After adding translated characters, run `uv run scripts/subset-ui-fonts.py` to regenerate the interface font subsets. Release ZIPs and acceptance screenshots are local build artefacts. Rebuild them where needed.

## Data

Windows uses the existing `%APPDATA%\Countdowner\config.json`. On other desktops, the same schema lives in `Countdowner/config.json` under the system's user configuration directory. A damaged or unsupported file is preserved and the app opens in read-only mode.

Deadlines retain their absolute time when the system time zone changes. Display and editing use the current local time zone.

## Limitations

Countdowner bundles small Noto Sans CJK subsets for interface text and uses system fonts for user-entered titles. NSH supplies regional calendar settings. Window placement, layering and tray integration follow each desktop backend. Windows 10 version 1809 or newer and macOS 14 or newer are required by their NSH backends.
