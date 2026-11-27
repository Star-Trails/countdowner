mod model;
mod settings;
mod strings;
mod widget;

use std::{cell::RefCell, path::PathBuf, time::Duration};

use base64::{Engine, engine::general_purpose::STANDARD};
use chrono::{DateTime, Utc};
use nsh_core::{
    desktop::{
        DisplayId, QuitMode, WindowAttributes, WindowFrame, WindowFrameStyle, WindowLevel,
        WindowPosition, Workspaces,
    },
    element::{
        ImageSrc, data_dir,
        desktop::{
            MenuItem, WindowRef, current_window, displays, on_reopen, quit, set_quit_mode,
            set_tray, tray, window,
        },
        next_second_instance, set_color_scheme,
    },
    platform::{Appearance, Locale},
    prelude::*,
};

use model::{Config, Language, Position, Theme};

#[native_module("countdowner/native")]
fn native(registry: &mut Registry) {
    registry.register::<App>();
}

#[derive(Clone, Copy)]
struct State {
    config: Signal<Config>,
    path: Signal<Option<PathBuf>>,
    read_only: bool,
    status: Signal<String>,
    now: Signal<chrono::DateTime<Utc>>,
    settings_open: Signal<bool>,
    settings_window: Signal<Option<WindowRef>>,
    widget_window: Signal<Option<WindowRef>>,
    deleting: Signal<Option<String>>,
}

impl State {
    fn save(self, change: impl FnOnce(&mut Config)) -> bool {
        let mut next = self.config.get_untracked();
        let language = next.language;
        let result = if self.read_only {
            Err(language.tr("readOnlyConfig").to_owned())
        } else {
            change(&mut next);
            if self.config.with_untracked(|current| *current == next) {
                return true;
            }
            self.path.with_untracked(|path| match path {
                Some(path) => next.save(path),
                None => Ok(()),
            })
        };
        match result {
            Ok(()) => {
                self.resize_widget_to_count(&next);
                self.status.set(next.language.tr("saved").to_owned());
                self.config.set(next);
                true
            }
            Err(error) => {
                log::warn!("{error}");
                self.status.set(error);
                false
            }
        }
    }

    fn show_settings(self) {
        self.deleting.set(None);
        self.settings_open.set(true);
        if let Some(window) = self.settings_window.get_untracked() {
            window.focus();
        }
    }

    fn delete(self, id: &str) {
        if self.save(|config| config.countdowns.retain(|item| item.id != id)) {
            self.deleting.set(None);
        }
    }

    fn resize_widget_to_count(self, next: &Config) {
        let old_height = self
            .config
            .with_untracked(|config| widget_height(config.countdowns.len()));
        let new_height = widget_height(next.countdowns.len());
        if old_height == new_height || next.widget_size.width > 0 {
            return;
        }
        if let Some(window) = self.widget_window.get_untracked()
            && let Some(state) = window.state()
            && (state.bounds.size.width.0 - 340.0).abs() < 1.0
            && (state.bounds.size.height.0 - old_height).abs() < 1.0
        {
            let mut frame = WindowFrame::default();
            frame.height = Some(px(new_height));
            window.set_bounds(frame);
        }
    }

    fn frame(self, position: &Position) -> WindowFrame {
        let screens = displays();
        let id = DisplayId::from_name(position.monitor.as_bytes());
        let area = screens
            .iter()
            .find(|screen| screen.id == id)
            .or_else(|| screens.iter().find(|screen| screen.primary))
            .or_else(|| screens.first())
            .map(|screen| screen.visible_bounds);
        let (width, height) = self.config.with_untracked(|config| {
            let saved = config.widget_size;
            if saved.width > 0 {
                (saved.width.max(240) as f32, saved.height.max(160) as f32)
            } else {
                (340.0, widget_height(config.countdowns.len()))
            }
        });
        let (x, y, width, height) = match area {
            Some(area) => {
                let width = width.min(area.size.width.0);
                let height = height.min(area.size.height.0);
                (
                    area.origin.x.0 + (position.x as f32).clamp(0.0, area.size.width.0 - width),
                    area.origin.y.0 + (position.y as f32).clamp(0.0, area.size.height.0 - height),
                    width,
                    height,
                )
            }
            None => (position.x as f32, position.y as f32, width, height),
        };
        let mut frame = WindowFrame::default();
        frame.x = Some(px(x));
        frame.y = Some(px(y));
        frame.width = Some(px(width));
        frame.height = Some(px(height));
        frame
    }
}

fn widget_height(count: usize) -> f32 {
    (84 + count.clamp(1, 6) * 110).min(620) as f32
}

/// Shared Rust root, mounted by the selected NSH backend.
#[component]
pub fn App() -> impl IntoElement {
    strings::register_ui_fonts();

    // NSH's test runner can supply an isolated app directory. The standard
    // desktop config path also locates the previous Windows app's data.
    let path = data_dir()
        .and_then(|dir| {
            if dir
                .file_name()
                .is_some_and(|name| name == "local.countdowner")
            {
                dirs::config_dir().map(|dir| dir.join("Countdowner"))
            } else {
                Some(dir)
            }
        })
        .map(|dir| dir.join("config.json"));
    let default = || {
        let mut config = Config::default();
        config.language = Language::from_locale(locale().as_str());
        config
    };
    let (config, error) = match path.as_deref().map(Config::load).transpose() {
        Ok(config) => (config.flatten().unwrap_or_else(default), None),
        Err(error) => (default(), Some(error)),
    };
    let state = State {
        read_only: error.is_some(),
        settings_open: signal(config.countdowns.is_empty() || error.is_some()),
        settings_window: signal(None),
        widget_window: signal(None),
        config: signal(config),
        path: signal(path),
        status: signal(error.unwrap_or_default()),
        now: signal(clock()),
        deleting: signal(None),
    };

    let language = computed(move || state.config.with(|config| config.language));
    let appearance = computed(move || state.config.with(|config| config.theme));
    effect(move || {
        set_color_scheme(match appearance.get() {
            Theme::System => Appearance::System,
            Theme::Light => Appearance::Light,
            Theme::Dark => Appearance::Dark,
        })
    });
    let icon = ImageSrc::from(format!(
        "data:image/png;base64,{}",
        STANDARD.encode(include_bytes!("../resources/icon.png")),
    ));
    effect(move || {
        let language = language.get();
        set_locale(Locale::parse(language.code()));
        set_tray(
            tray()
                .icon(icon.clone())
                .tooltip("Countdowner")
                .menu([
                    MenuItem::new(language.tr("settings")).on_select(move || state.show_settings()),
                    MenuItem::new(language.tr("quit")).on_select(quit),
                ])
                .on_click(move |_| state.show_settings()),
        );
    });
    set_quit_mode(QuitMode::Explicit);
    on_reopen(move || state.show_settings());
    spawn(async move {
        loop {
            next_second_instance().await;
            state.show_settings();
        }
    });
    let refresh = RefCell::new(None::<Spawned<()>>);
    effect(move || {
        let previous = state.now.get();
        let _ = wake_count();
        let now = clock();
        let delay = state.config.with(|config| config.next_refresh(now));
        if let Some(previous) = refresh.borrow_mut().take() {
            previous.cancel();
        }
        if (now - previous).num_milliseconds().unsigned_abs() > 1000 {
            state.now.set(now);
        } else {
            *refresh.borrow_mut() = Some(set_timeout(
                delay.unwrap_or(Duration::from_secs(60)),
                move || {
                    state.now.set(clock());
                },
            ));
        }
    });

    let initial = state
        .config
        .with_untracked(|config| state.frame(&config.position));
    let mut attributes = WindowAttributes::default();
    attributes.frame = Some(WindowFrameStyle::None);
    attributes.level = Some(WindowLevel::DesktopIcon);
    attributes.workspaces = Some(Workspaces::All);
    attributes.taskbar = Some(false);
    attributes.activate = Some(false);
    attributes.focusable = Some(false);
    attributes.minimizable = Some(false);
    attributes.maximizable = Some(false);
    attributes.closable = Some(false);
    attributes.resizable = Some(true);
    attributes.transparent = Some(true);
    attributes.shadow = Some(false);
    attributes.min_size = Some(Some(size(px(240.), px(160.))));
    view()
        .flex_1()
        .child(
            window()
                .id("widget")
                .title("Countdowner")
                .attributes(attributes)
                .default_size(size(initial.width.unwrap(), initial.height.unwrap()))
                .default_position(WindowPosition::At(point(
                    initial.x.unwrap(),
                    initial.y.unwrap(),
                )))
                .restore_bounds(true)
                .content(move || {
                    state.widget_window.set(Some(current_window()));
                    widget::content(state)
                }),
        )
        .child(
            window()
                .id("settings")
                .title(move || format!("Countdowner — {}", language.get().tr("settings")))
                .default_size(size(px(650.), px(760.)))
                .restore_bounds(true)
                .open(state.settings_open)
                .on_open_change(move |change| {
                    state.settings_open.set(change.open);
                    if !change.open {
                        state.settings_window.set(None);
                    }
                })
                .content(move || {
                    state.settings_window.set(Some(current_window()));
                    settings::content(state)
                }),
        )
}

fn clock() -> DateTime<Utc> {
    DateTime::from_timestamp_millis(nsh_core::time::unix_ms() as i64).expect("valid wall clock")
}

#[cfg(test)]
mod tests {
    use nsh_core::{element::Root, platform::Os};
    use nsh_engine::EngineConfig;
    use nsh_testing::Headless;

    use super::*;

    #[test]
    fn settings_edits_the_widget_and_keeps_the_saved_deadline() {
        let dir = tempfile::tempdir().unwrap();
        let mut config = EngineConfig::new(Root::of::<App>());
        config.environment.os = Os::Windows;
        config.environment.locale = Locale::EN_US;
        config.data_dir = Some(dir.path().to_owned());
        let mut app = Headless::start(config).unwrap();
        app.wait_for("id:draft-title").unwrap();
        assert_eq!(app.window_ids().unwrap(), ["widget", "settings"]);

        app.type_text("id:draft-title", "Release review").unwrap();
        app.press("label:Date").unwrap();
        app.wait_for("text:Mo").unwrap();
        app.press("text:15").unwrap();
        app.press("text:Add countdown").unwrap();
        let path = dir.path().join("config.json");
        let saved = Config::load(&path).unwrap().unwrap();
        assert_eq!(saved.countdowns[0].title, "Release review");
        assert_eq!(
            saved.countdowns[0]
                .target
                .with_timezone(&chrono::Local)
                .format("%d")
                .to_string(),
            "15"
        );
        assert!(app.shows("Release review"));
        app.press("text:Edit").unwrap();
        app.clear("id:draft-title").unwrap();
        app.type_text("id:draft-title", "Updated review").unwrap();
        app.press("text:Save changes").unwrap();
        let edited = Config::load(&path).unwrap().unwrap();
        assert_eq!(edited.countdowns[0].title, "Updated review");
        assert_eq!(edited.countdowns[0].target, saved.countdowns[0].target);
        assert_eq!(edited.countdowns[0].id, saved.countdowns[0].id);

        app.press_window_close("settings").unwrap();
        assert_eq!(app.window_ids().unwrap(), ["widget"]);
        app.press("label:Settings").unwrap();
        app.wait_for("id:draft-title").unwrap();
        assert_eq!(app.window_ids().unwrap(), ["widget", "settings"]);
        app.press_window_close("settings").unwrap();
        app.press("label:Delete").unwrap();
        assert_eq!(Config::load(&path).unwrap().unwrap(), edited);
        app.press("text:Updated review").unwrap();
        app.wait_for_gone("label:Confirm delete", Duration::from_secs(1))
            .unwrap();
        app.press("label:Delete").unwrap();
        app.press("label:Confirm delete").unwrap();
        assert!(Config::load(&path).unwrap().unwrap().countdowns.is_empty());
    }
}

#[cfg(test)]
#[test]
fn widget_requires_a_hold_before_reordering() {
    use std::time::SystemTime;

    use nsh_core::{element::Root, platform::Os, time::Clock};
    use nsh_engine::EngineConfig;
    use nsh_testing::{DragOptions, Headless};

    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("config.json");
    let target = Utc::now().fixed_offset() + chrono::Duration::hours(2);
    let mut saved = Config::default();
    saved.countdowns = [("first", 0), ("second", 1)]
        .map(|(id, days)| model::Countdown {
            id: id.to_owned(),
            title: id.to_owned(),
            target: target + chrono::Duration::days(days),
            time_zone: "Local".to_owned(),
        })
        .to_vec();
    saved.save(&path).unwrap();
    let mut config = EngineConfig::new(Root::of::<App>());
    config.environment.os = Os::Windows;
    config.environment.locale = Locale::EN_US;
    config.data_dir = Some(dir.path().to_owned());
    config.clock = Clock::manual(SystemTime::now());
    let mut app = Headless::start(config).unwrap();
    app.wait_for("text:first").unwrap();
    let grip = "label:Reorder >> nth=0";
    let down = point(px(0.), px(94.));
    app.drag(grip, down, DragOptions::default()).unwrap();
    assert_eq!(Config::load(&path).unwrap().unwrap(), saved);

    let mut hold = DragOptions::default();
    hold.hold = Some(Duration::from_millis(300));
    app.drag(grip, down, hold).unwrap();
    let reordered = Config::load(&path).unwrap().unwrap();
    assert_eq!(reordered.sort, model::Sort::Manual);
    assert_eq!(reordered.countdowns[0].id, "second");
}
