use chrono::{Duration, Local, NaiveDate, NaiveDateTime, Timelike};
use nsh_base::{
    CalendarEntry, CalendarItemKind, DatePickerValue, PickedDate, button, checkbox, date_picker,
    input, number_input, theme,
};
use nsh_core::{
    element::{
        desktop::{launch_at_login, quit, set_launch_at_login},
        open_url,
    },
    prelude::*,
};

use crate::{
    State,
    model::{Countdown, DraftError, Language, Position, Sort, Theme, remaining},
    strings::LANGUAGES,
};

#[derive(Clone, Copy)]
struct Draft {
    id: Signal<Option<String>>,
    title: Signal<String>,
    date: Signal<NaiveDate>,
    hour: Signal<Option<f64>>,
    minute: Signal<Option<f64>>,
    date_only: Signal<bool>,
}

impl Draft {
    fn new() -> Self {
        let next = next_hour();
        Self {
            id: signal(None),
            title: signal(String::new()),
            date: signal(next.date()),
            hour: signal(Some(f64::from(next.hour()))),
            minute: signal(Some(0.)),
            date_only: signal(false),
        }
    }

    fn reset(self) {
        let next = next_hour();
        self.id.set(None);
        self.title.set(String::new());
        self.date.set(next.date());
        self.hour.set(Some(f64::from(next.hour())));
        self.minute.set(Some(0.));
        self.date_only.set(false);
    }

    fn edit(self, item: &Countdown) {
        let time = item.target.with_timezone(&Local);
        self.id.set(Some(item.id.clone()));
        self.title.set(item.title.clone());
        self.date.set(time.date_naive());
        self.hour.set(Some(f64::from(time.hour())));
        self.minute.set(Some(f64::from(time.minute())));
        self.date_only.set(time.time() == chrono::NaiveTime::MIN);
    }

    fn save(self, state: State, language: Language) {
        let title = self.title.get_untracked();
        let edited = self.id.get_untracked();
        let to_unit = |value: Option<f64>, max: f64| {
            value
                .filter(|v| v.fract() == 0. && (0. ..=max).contains(v))
                .map(|v| v as u32)
        };
        let date = self.date.get_untracked();
        let target = if self.date_only.get_untracked() {
            date.and_hms_opt(0, 0, 0)
        } else {
            to_unit(self.hour.get_untracked(), 23.)
                .zip(to_unit(self.minute.get_untracked(), 59.))
                .and_then(|(hour, minute)| date.and_hms_opt(hour, minute, 0))
        };
        let result = target.ok_or(DraftError::Date).and_then(|target| {
            state.config.with_untracked(|config| {
                let original = edited
                    .as_ref()
                    .and_then(|id| config.countdowns.iter().find(|item| &item.id == id));
                Countdown::from_draft(&title, target, original)
            })
        });
        let item = match result {
            Ok(item) => item,
            Err(error) => {
                state.status.set(
                    language
                        .tr(match error {
                            DraftError::Title => "enterATitle",
                            DraftError::Date => "invalidDateTime",
                        })
                        .to_owned(),
                );
                return;
            }
        };
        if state.save(|config| {
            if let Some(id) = edited {
                if let Some(original) = config.countdowns.iter_mut().find(|item| item.id == id) {
                    *original = item;
                }
            } else {
                config.countdowns.push(item);
            }
        }) {
            self.reset();
        }
    }
}

fn next_hour() -> NaiveDateTime {
    let now = Local::now().naive_local();
    now.date().and_hms_opt(now.hour(), 0, 0).unwrap() + Duration::hours(1)
}

fn command(
    label: impl IntoReactive<SharedString>,
    disabled: bool,
    action: impl Fn() + 'static,
) -> impl IntoElement {
    button()
        .flex_shrink_0()
        .rounded(px(5.))
        .px(px(9.))
        .py(px(5.))
        .bg(move || theme().muted)
        .hover(|style| style.bg(move || theme().accent))
        .border_1()
        .border_color(hsla(0., 0., 0., 0.))
        .focus_visible(|style| style.border_color(move || theme().ring))
        .disabled(disabled)
        .on_press(action)
        .child(text(label).text_sm().text_color(move || theme().foreground))
}

fn heading(label: impl IntoReactive<SharedString>) -> impl IntoElement {
    text(label)
        .text_size(px(16.))
        .font_weight(FontWeight::SEMIBOLD)
        .text_color(move || theme().foreground)
}

fn field_label(label: &'static str) -> impl IntoElement {
    text(label)
        .text_sm()
        .font_weight(FontWeight::MEDIUM)
        .text_color(move || theme().foreground)
}

fn caption(label: impl IntoReactive<SharedString>) -> impl IntoElement {
    text(label)
        .text_size(px(13.))
        .text_color(move || theme().muted_foreground)
}

fn divider() -> impl IntoElement {
    view().h(px(1.)).flex_shrink_0().bg(move || theme().border)
}

pub(super) fn content(state: State) -> impl IntoElement {
    let draft = Draft::new();
    let title = element_ref();
    let delete = signal(None::<String>);
    let login = signal(false);
    let login_busy = signal(true);
    spawn(async move {
        login.set(launch_at_login().await);
        login_busy.set(false);
    });
    let language = computed(move || state.config.with(|config| config.language));
    view()
        .flex_1()
        .min_h(px(0.))
        .font_family(move || language.get().ui_font())
        .bg(move || theme().background)
        .child(scroll().flex_1().min_h(px(0.)).child(dynamic(move || {
            let language = language.get();
            view()
                .gap(px(18.))
                .p(px(22.))
                .pb(px(26.))
                .child(
                    view()
                        .gap(px(4.))
                        .child(
                            text("Countdowner")
                                .font_family(Language::En.ui_font())
                                .text_size(px(24.))
                                .font_weight(FontWeight::BOLD)
                                .text_color(move || theme().foreground),
                        )
                        .child(caption(language.tr("tagline"))),
                )
                .child(preferences(state, language, login, login_busy))
                .child(divider())
                .child(editor(state, language, draft, &title))
                .child(divider())
                .child(countdowns(state, language, draft, delete, title.clone()))
                .child(divider())
                .child(caption(move || {
                    state.path.with(|path| {
                        path.as_ref()
                            .and_then(|path| path.parent())
                            .map(|path| {
                                format!("{}: {}", language.tr("dataFolder"), path.display())
                            })
                            .unwrap_or_default()
                    })
                }))
                .child(
                    view()
                        .flex_row()
                        .flex_wrap()
                        .gap(px(8.))
                        .child(command(
                            language.tr("resetPosition"),
                            state.read_only,
                            move || {
                                let position = Position::default();
                                if state.save(|config| config.position = position.clone()) {
                                    if let Some(widget) = state.widget_window.get_untracked() {
                                        let mut frame = state.frame(&position);
                                        frame.width = None;
                                        frame.height = None;
                                        widget.set_bounds(frame);
                                    }
                                }
                            },
                        ))
                        .child(command(language.tr("quit"), false, quit)),
                )
                .child(divider())
                .child(heading(language.tr("about")))
                .child(
                    view()
                        .flex_row()
                        .gap(px(8.))
                        .child(command(
                            concat!("Countdowner ", env!("CARGO_PKG_VERSION")),
                            false,
                            || {
                                let _ = open_url("https://github.com/Star-Trails/countdowner");
                            },
                        ))
                        .child(command("Native Shell 0.7.2", false, || {
                            let _ = open_url("https://nsh.dev");
                        })),
                )
        })))
        .child(
            view()
                .px(px(22.))
                .py(px(10.))
                .gap(px(2.))
                .border_t_1()
                .border_color(move || theme().border)
                .bg(move || theme().muted)
                .child(dynamic(move || {
                    let status = state.status.get();
                    (!status.is_empty()).then(|| caption(status))
                }))
                .child(caption(move || language.get().tr("autosaveHint"))),
        )
}

fn preferences(
    state: State,
    language: Language,
    login: Signal<bool>,
    login_busy: Signal<bool>,
) -> impl IntoElement {
    let theme_choice = computed(move || state.config.with(|config| config.theme));
    view()
        .gap(px(8.))
        .child(heading(language.tr("appearance")))
        .child(field_label(language.tr("language")))
        .child(
            view()
                .flex_row()
                .flex_wrap()
                .gap(px(6.))
                .child(LANGUAGES.map(|(choice, label)| {
                    option(
                        language_label(choice, label),
                        move || language == choice,
                        state.read_only,
                        move || {
                            state.save(|config| config.language = choice);
                        },
                    )
                })),
        )
        .child(field_label(language.tr("theme")))
        .child(
            view().flex_row().flex_wrap().gap(px(6.)).child(
                [
                    (Theme::System, "followSystem"),
                    (Theme::Light, "light"),
                    (Theme::Dark, "dark"),
                ]
                .map(|(choice, key)| {
                    option(
                        text(language.tr(key)).font_weight(FontWeight::MEDIUM),
                        move || theme_choice.get() == choice,
                        state.read_only,
                        move || {
                            state.save(|config| config.theme = choice);
                        },
                    )
                }),
            ),
        )
        .child(check(
            language.tr("startAtLogin"),
            login,
            move || state.read_only || login_busy.get(),
            move |enabled| {
                login_busy.set(true);
                spawn(async move {
                    match set_launch_at_login(enabled).await {
                        Ok(()) => {
                            login.set(launch_at_login().await);
                            state.save(|config| config.autostart = login.get_untracked());
                        }
                        Err(error) => state.status.set(error.to_string()),
                    }
                    login_busy.set(false);
                });
            },
        ))
}

fn option(
    label: Text,
    selected: impl Fn() -> bool + 'static,
    disabled: bool,
    action: impl Fn() + 'static,
) -> impl IntoElement {
    button()
        .flex_shrink_0()
        .rounded(px(5.))
        .px(px(9.))
        .py(px(5.))
        .border_1()
        .border_color(move || theme().border)
        .bg(move || {
            if selected() {
                theme().accent
            } else {
                hsla(0., 0., 0., 0.).into()
            }
        })
        .hover(|style| style.bg(move || theme().accent))
        .focus_visible(|style| style.border_color(move || theme().ring))
        .disabled(disabled)
        .on_press(action)
        .child(label.text_sm().text_color(move || theme().foreground))
}

fn language_label(language: Language, label: &'static str) -> Text {
    text(label)
        .font_family(language.ui_font())
        .font_weight(FontWeight::NORMAL)
}

fn check(
    label: &'static str,
    checked: Signal<bool>,
    disabled: impl Fn() -> bool + 'static,
    action: impl Fn(bool) + 'static,
) -> impl IntoElement {
    checkbox()
        .checked(checked)
        .disabled(disabled)
        .app_class("flex-row items-center gap-2")
        .indicator_class(
            "size-5 rounded border border-input items-center justify-center checked:bg-primary",
        )
        .indicator(
            text(move || if checked.get() { "✓" } else { "" })
                .text_color(move || theme().primary_foreground),
        )
        .on_change(move |value: &bool| action(*value))
        .child(field_label(label))
}

fn editor(
    state: State,
    language: Language,
    draft: Draft,
    title: &ElementRef,
) -> impl IntoElement + use<> {
    let editing = computed(move || draft.id.with(Option::is_some));
    view().gap(px(8.))
        .child(heading(move || language.tr(if editing.get() { "editCountdown" } else { "newCountdown" })))
        .child(input().value(draft.title).placeholder(language.tr("title"))
            .label(language.tr("title")).test_id("draft-title")
            .font_family(language.title_input_font())
            .element_ref(title).disabled(state.read_only)
            .rounded(px(5.)).border_1().border_color(move || theme().border)
            .h(px(34.)).px(px(8.)).text_color(move || theme().foreground)
            .on_change(move |value: &str| draft.title.set(value.to_owned()))
            .on_submit(move |_: &str| draft.save(state, language)))
        .child(field_label(language.tr("targetDateAndTime")))
        .child(view().flex_row().flex_wrap().items_center().gap(px(8.))
            .child(date_picker()
                .value(move || DatePickerValue::from(draft.date.get()))
                .first_day_of_week(move || u32::from(locale_settings().first_day_of_week.number() % 7))
                .accessibility_label(language.tr("date"))
                .disabled(state.read_only)
                .offset(px(6.))
                .popup_class("p-3 rounded-lg border border-border bg-popover shadow-md")
                .header_class("flex-row items-center justify-between gap-2")
                .item_class("min-w-8 h-8 px-1 items-center justify-center rounded text-sm whitespace-nowrap text-foreground hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring selected:bg-primary selected:text-primary-foreground data-outside:text-muted-foreground")
                .render_item(move |entry: &CalendarEntry| calendar_item(language, entry))
                .trigger(view().px(px(9.)).py(px(5.)).rounded(px(5.))
                    .bg(move || theme().muted).child(text(move || format!("{} ▾", draft.date.get()))
                        .text_sm().text_color(move || theme().foreground)))
                .on_change(move |picked: &PickedDate| {
                    if let Some(date) = picked.value.start() { draft.date.set(date.date()); }
                }))
            .child(command(language.tr("today"), state.read_only, move || draft.date.set(Local::now().date_naive())))
            .child(command(language.tr("tomorrow"), state.read_only, move || {
                draft.date.set(Local::now().date_naive().succ_opt().unwrap());
            })))
        .child(check(language.tr("dateOnly"), draft.date_only, move || state.read_only, move |value| draft.date_only.set(value)))
        .child(dynamic(move || (!draft.date_only.get()).then(|| view().flex_row().flex_wrap().items_center().gap(px(8.))
            .child(field_label(language.tr("hour")))
            .child(clock_input(draft.hour, 23., language.tr("hourInput"), state.read_only))
            .child(field_label(language.tr("minute")))
            .child(clock_input(draft.minute, 59., language.tr("minuteInput"), state.read_only)))))
        .child(caption(language.tr("timeHelp")))
        .child(view().flex_row().gap(px(8.))
            .child(button().rounded(px(5.)).px(px(10.)).py(px(6.))
                .border_1().border_color(hsla(0., 0., 0., 0.))
                .bg(move || theme().primary)
                .hover(|style| style.opacity(0.88))
                .focus_visible(|style| style.border_color(move || theme().ring))
                .disabled(state.read_only)
                .on_press(move || draft.save(state, language))
                .child(text(move || language.tr(if editing.get() { "saveChanges" } else { "addCountdown" }))
                    .text_sm().font_weight(FontWeight::SEMIBOLD).text_color(move || theme().primary_foreground)))
            .child(dynamic(move || editing.get().then(|| command(language.tr("cancel"), false, move || draft.reset())))))
}

fn calendar_item(language: Language, entry: &CalendarEntry) -> impl IntoElement + use<> {
    use CalendarItemKind::*;

    let label: SharedString = match entry.kind {
        Previous => "‹".into(),
        Next => "›".into(),
        Month | MonthToggle => match language {
            Language::En => [
                "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
            ][entry.month as usize - 1]
                .into(),
            Language::Ko => format!("{}월", entry.month).into(),
            _ => format!("{}月", entry.month).into(),
        },
        Weekday => (match language {
            Language::En => ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"],
            Language::ZhHant | Language::ZhHans => ["日", "一", "二", "三", "四", "五", "六"],
            Language::Ja => ["日", "月", "火", "水", "木", "金", "土"],
            Language::Ko => ["일", "월", "화", "수", "목", "금", "토"],
        })[entry.value as usize]
            .into(),
        _ => entry.value.to_string().into(),
    };
    text(label).whitespace_nowrap().tabular_nums()
}

fn clock_input(
    value: Signal<Option<f64>>,
    max: f64,
    label: &'static str,
    disabled: bool,
) -> impl IntoElement {
    number_input()
        .value(value)
        .min(Some(0.))
        .max(Some(max))
        .disabled(disabled)
        .app_class("w-28 flex-row items-center rounded border border-input")
        .input_class("h-8 flex-1 text-center text-foreground")
        .decrement_button_class("size-8 items-center justify-center bg-muted text-foreground")
        .decrement_button(text("−").text_sm().text_color(move || theme().foreground))
        .increment_button_class("size-8 items-center justify-center bg-muted text-foreground")
        .increment_button(text("+").text_sm().text_color(move || theme().foreground))
        .accessibility_label(label)
        .on_change(move |next: &Option<f64>| value.set(*next))
}

fn countdowns(
    state: State,
    language: Language,
    draft: Draft,
    delete: Signal<Option<String>>,
    title: ElementRef,
) -> impl IntoElement {
    let sort = computed(move || state.config.with(|config| config.sort));
    view()
        .gap(px(8.))
        .child(heading(language.tr("countdowns")))
        .child(
            view().flex_row().flex_wrap().gap(px(6.)).child(
                [
                    (Sort::Manual, "sortManual"),
                    (Sort::Deadline, "sortDeadline"),
                ]
                .map(|(choice, key)| {
                    option(
                        text(language.tr(key)).font_weight(FontWeight::MEDIUM),
                        move || sort.get() == choice,
                        state.read_only,
                        move || {
                            state.save(|config| config.sort = choice);
                        },
                    )
                }),
            ),
        )
        .child(dynamic(move || {
            state.config.with(|config| {
                let items = config.ordered();
                if items.is_empty() {
                    return caption(language.tr("emptyCountdowns")).into_element();
                }
                let rows = items
                    .iter()
                    .enumerate()
                    .map(|(index, item)| {
                        let item = (*item).clone();
                        let id = item.id.clone();
                        let target = item.target;
                        let title = title.clone();
                        let move_item = |direction: isize| {
                            let id = id.clone();
                            move || {
                                state.save(|config| {
                                    if let Some(i) =
                                        config.countdowns.iter().position(|item| item.id == id)
                                    {
                                        let other = i.saturating_add_signed(direction);
                                        if config.sort == Sort::Manual
                                            && other < config.countdowns.len()
                                        {
                                            config.countdowns.swap(i, other);
                                        }
                                    }
                                });
                            }
                        };
                        view()
                            .gap(px(6.))
                            .rounded(px(6.))
                            .p(px(10.))
                            .bg(move || theme().card)
                            .child(
                                view()
                                    .flex_row()
                                    .flex_wrap()
                                    .gap(px(8.))
                                    .child(
                                        text(item.title.clone())
                                            .flex_1()
                                            .font_family(language.title_font())
                                            .font_weight(FontWeight::NORMAL)
                                            .text_color(move || theme().foreground),
                                    )
                                    .child(caption(move || {
                                        language.remaining(remaining(target, state.now.get()))
                                    })),
                            )
                            .child(caption(move || {
                                let _ = locale_settings();
                                target
                                    .with_timezone(&Local)
                                    .format("%Y-%m-%d %H:%M %Z")
                                    .to_string()
                            }))
                            .child(
                                view()
                                    .flex_row()
                                    .flex_wrap()
                                    .gap(px(6.))
                                    .child(command(
                                        language.tr("edit"),
                                        state.read_only,
                                        move || {
                                            draft.edit(&item);
                                            title.focus();
                                        },
                                    ))
                                    .child(command(
                                        language.tr("moveUp"),
                                        state.read_only
                                            || config.sort != Sort::Manual
                                            || index == 0,
                                        move_item(-1),
                                    ))
                                    .child(command(
                                        language.tr("moveDown"),
                                        state.read_only
                                            || config.sort != Sort::Manual
                                            || index + 1 == items.len(),
                                        move_item(1),
                                    ))
                                    .child(dynamic(move || {
                                        let armed = delete.with(|pending| {
                                            pending.as_deref() == Some(id.as_str())
                                        });
                                        let id = id.clone();
                                        view()
                                            .flex_row()
                                            .gap(px(6.))
                                            .child(command(
                                                language.tr(if armed {
                                                    "confirmDelete"
                                                } else {
                                                    "delete"
                                                }),
                                                state.read_only,
                                                move || {
                                                    if armed {
                                                        state.delete(&id);
                                                        delete.set(None);
                                                        if draft.id.with_untracked(|current| {
                                                            current.as_deref() == Some(id.as_str())
                                                        }) {
                                                            draft.reset();
                                                        }
                                                    } else {
                                                        delete.set(Some(id.clone()));
                                                    }
                                                },
                                            ))
                                            .child(armed.then(|| {
                                                command(language.tr("cancel"), false, move || {
                                                    delete.set(None)
                                                })
                                            }))
                                    })),
                            )
                            .into_element()
                    })
                    .collect::<Vec<_>>();
                view().gap(px(8.)).child(rows).into_element()
            })
        }))
}
