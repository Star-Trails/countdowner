use std::{
    cell::Cell,
    f32::consts::{FRAC_PI_2, TAU},
    rc::Rc,
    time::Duration,
};

use chrono::Local;
use nsh_base::theme;
use nsh_core::{desktop::ResizeEdge, prelude::*};

use crate::{
    State,
    model::{Countdown, Indicator, indicator, remaining},
};

const ROW_GAP: f32 = 14.0;
const REORDER_HOLD: Duration = Duration::from_millis(300);

pub(super) fn content(state: State) -> impl IntoElement {
    let language = computed(move || state.config.with(|config| config.language));
    let count = computed(move || state.config.with(|config| config.countdowns.len()));
    let padding = computed(|| ((window().width.0 - 240.) * 0.035 + 12.).clamp(12., 20.));
    let countdown_size = computed(|| (20. + (window().width.0 - 240.) * 0.05).clamp(20., 26.));
    let row_height = computed(move || {
        let rows = count.get().clamp(1, 6) as f32;
        let available = window().height.0 - 2. * padding.get() - 45.;
        // Reserve two lines each for the title and countdown plus the date and gaps.
        let minimum = 89. + 2. * (countdown_size.get() - 20.);
        ((available - (rows - 1.) * ROW_GAP) / rows).clamp(minimum, 110.)
    });
    view()
        .relative()
        .flex_1()
        .min_h(px(0.))
        .rounded(px(8.))
        .font_family(move || language.get().ui_font())
        .bg(move || theme().background)
        .py(move || px(padding.get()))
        .px(move || px(padding.get().max((window().width.0 - 386.) / 2.)))
        .gap(px(8.))
        .on_press(move || state.deleting.set(None))
        .child(
            view()
                .flex_row()
                .items_center()
                .gap(px(8.))
                .window_drag(true)
                .child(
                    text("Countdowner")
                        .font_family(crate::model::Language::En.ui_font())
                        .font_weight(FontWeight::SEMIBOLD)
                        .flex_1()
                        .text_color(move || theme().foreground),
                )
                .child(
                    view().window_drag(false).child(
                        nsh_base::button()
                            .size(px(28.))
                            .rounded(px(5.))
                            .items_center()
                            .justify_center()
                            .border_1()
                            .border_color(move || theme().border)
                            .hover(|style| style.bg(move || theme().accent))
                            .focus_visible(|style| style.border_color(move || theme().ring))
                            .accessibility_label(move || {
                                Some(SharedString::from(language.get().tr("settings")))
                            })
                            .on_press(move || state.show_settings())
                            .child(canvas(gear).size(px(18.))),
                    ),
                ),
        )
        .child(view().h(px(1.)).flex_shrink_0().bg(move || theme().border))
        .child(scroll().flex_1().min_h(px(0.)).child(dynamic(move || {
            state.config.with(|config| {
                let items = config.ordered();
                if items.is_empty() {
                    return text(config.language.tr("emptyCountdowns"))
                        .text_color(move || theme().muted_foreground)
                        .into_element();
                }
                view()
                    .gap(px(ROW_GAP))
                    .child(
                        items
                            .iter()
                            .enumerate()
                            .map(|(index, item)| {
                                row(
                                    state,
                                    (*item).clone(),
                                    index,
                                    items.len(),
                                    row_height,
                                    countdown_size,
                                )
                                .into_element()
                            })
                            .collect::<Vec<_>>(),
                    )
                    .into_element()
            })
        })))
        .child(resize_handles())
}

fn row(
    state: State,
    item: Countdown,
    index: usize,
    count: usize,
    row_height: Memo<f32>,
    countdown_size: Memo<f32>,
) -> impl IntoElement {
    let language = state.config.with_untracked(|config| config.language);
    let target = item.target;
    let id = item.id;
    let graphic_scale = computed(|| ((window().width.0 - 240.) / 340. + 0.73).clamp(0.73, 1.06));
    let deleting = {
        let id = id.clone();
        computed(move || {
            state
                .deleting
                .with(|pending| pending.as_deref() == Some(id.as_str()))
        })
    };
    let drag_started = Rc::new(Cell::new(None::<nsh_core::time::Instant>));
    let dragging = signal(false);
    let handle = view()
        .size(px(24.))
        .flex_shrink_0()
        .items_center()
        .justify_center()
        .rounded(px(4.))
        .hover(|style| style.bg(move || theme().muted))
        .label(language.tr("reorder"))
        .pan_enabled(!state.read_only)
        .on_press_down({
            let started = Rc::clone(&drag_started);
            move || {
                state.deleting.set(None);
                started.set(Some(nsh_core::time::now()));
            }
        })
        .on_pan_start(move |_: &PanEvent| {
            dragging.set(
                drag_started
                    .take()
                    .is_some_and(|start| nsh_core::time::now() - start >= REORDER_HOLD),
            );
        })
        .on_pan_end({
            let id = id.clone();
            move |pan: &PanEvent| {
                let dragged = dragging.get_untracked();
                dragging.set(false);
                if !dragged || pan.cancelled {
                    return;
                }
                let steps =
                    (pan.translation.y.0 / (row_height.get_untracked() + ROW_GAP)).round() as isize;
                let anchor = index.saturating_add_signed(steps).min(count - 1);
                if anchor != index {
                    state.save(|config| {
                        let anchor = config.ordered()[anchor].id.clone();
                        config.reorder(&id, &anchor);
                    });
                }
            }
        })
        .child(text("⠿").text_color(move || theme().muted_foreground));
    let remove = nsh_base::button()
        .size(px(24.))
        .items_center()
        .justify_center()
        .rounded(px(4.))
        .bg(move || {
            if deleting.get() {
                hsla(0.99, 0.64, 0.44, 1.)
            } else {
                hsla(0., 0., 0., 0.)
            }
        })
        .hover(|style| {
            style.bg(move || {
                if deleting.get() {
                    hsla(0.99, 0.64, 0.38, 1.).into()
                } else {
                    theme().accent
                }
            })
        })
        .focus_visible(|style| style.border_1().border_color(move || theme().ring))
        .accessibility_label(move || {
            Some(SharedString::from(language.tr(if deleting.get() {
                "confirmDelete"
            } else {
                "delete"
            })))
        })
        .disabled(state.read_only)
        .on_press(move || {
            if deleting.get_untracked() {
                state.delete(&id);
            } else {
                state.deleting.set(Some(id.clone()));
            }
        })
        .child(
            canvas(move || {
                let ink = if deleting.get() {
                    hsla(0., 0., 1., 1.)
                } else {
                    color(theme().foreground)
                };
                Painter::new()
                    .stroke_color(ink)
                    .line_width(1.25)
                    .begin_path()
                    .move_to(5., 7.)
                    .line_to(15., 7.)
                    .move_to(8., 7.)
                    .line_to(8., 5.)
                    .line_to(12., 5.)
                    .line_to(12., 7.)
                    .move_to(6.5, 7.)
                    .line_to(7.4, 15.)
                    .line_to(12.6, 15.)
                    .line_to(13.5, 7.)
                    .stroke()
            })
            .size(px(20.)),
        );
    view()
        .h(move || px(row_height.get()))
        .flex_shrink_0()
        .flex_row()
        .items_center()
        .gap(px(8.))
        .rounded(px(4.))
        .bg(move || {
            if dragging.get() {
                theme().muted
            } else {
                hsla(0., 0., 0., 0.).into()
            }
        })
        .child(
            view()
                .flex_1()
                .min_w(px(0.))
                .gap(px(2.))
                .child(
                    text(item.title)
                        .font_family("system-ui")
                        .text_size(px(14.))
                        .line_height(px(16.))
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(move || theme().foreground)
                        .number_of_lines(2u32),
                )
                .child(
                    text(move || language.remaining(remaining(target, state.now.get())))
                        .text_size(move || px(countdown_size.get()))
                        .line_height(move || px(countdown_size.get() + 1.))
                        .number_of_lines(2u32)
                        .tabular_nums()
                        .font_weight(FontWeight::BOLD)
                        .text_color(move || theme().foreground),
                )
                .child(
                    text(move || {
                        let _ = locale_settings();
                        target
                            .with_timezone(&Local)
                            .format("%Y-%m-%d %H:%M")
                            .to_string()
                    })
                    .text_xs()
                    .line_height(px(13.))
                    .tabular_nums()
                    .text_color(move || theme().muted_foreground)
                    .truncate(),
                ),
        )
        .child(dynamic(move || {
            let state = indicator(target, state.now.get());
            if matches!(state, Indicator::Empty) {
                return None;
            }
            Some(
                canvas(move || paint_indicator(&state, graphic_scale.get()))
                    .w(move || px(96. * graphic_scale.get()))
                    .h(move || px(72. * graphic_scale.get()))
                    .flex_shrink_0()
                    .rounded(px(6.))
                    .bg(move || theme().muted)
                    .label(language.tr("countdownIndicator")),
            )
        }))
        .child(view().gap(px(4.)).child(remove).child(handle))
}

fn gear() -> Painter {
    let mut painter = Painter::new()
        .stroke_color(color(theme().foreground))
        .line_width(1.25)
        .begin_path();
    for tooth in 0..32 {
        let angle = TAU * tooth as f32 / 32.;
        let radius = if tooth % 4 < 2 { 7. } else { 5.5 };
        let (x, y) = (9. + radius * angle.cos(), 9. + radius * angle.sin());
        painter = if tooth == 0 {
            painter.move_to(x, y)
        } else {
            painter.line_to(x, y)
        };
    }
    painter
        .close_path()
        .stroke()
        .begin_path()
        .arc(9., 9., 2.4, 0., TAU)
        .stroke()
}

fn paint_indicator(indicator: &Indicator, scale: f32) -> Painter {
    let dark = dark();
    let neutral = if dark { 0x505862 } else { 0xcdd2d7 };
    let hues = if dark {
        [0x62b879, 0xd9b44a, 0xf07878, 0xf2a44d, 0xfa8458]
    } else {
        [0x238636, 0x9a7400, 0xc33236, 0xb55b12, 0xc24820]
    };
    match *indicator {
        Indicator::Grid { days, urgency } => {
            let mut painter = Painter::new().scale(scale, scale);
            for index in 0..31 {
                let fill = if index < days { hues[urgency] } else { neutral };
                let x = 7.0 + (index % 7) as f32 * 12.0;
                let y = 7.0 + (index / 7) as f32 * 12.0;
                painter = painter
                    .fill_color(color(rgb(fill)))
                    .begin_path()
                    .round_rect(
                        Bounds::new(point(px(x), px(y)), size(px(10.), px(10.))),
                        px(1.5),
                    )
                    .fill();
            }
            painter
        }
        Indicator::Ring { fraction, urgency } => track(neutral, scale)
            .stroke_color(color(rgb(hues[urgency])))
            .begin_path()
            .arc(
                48.,
                36.,
                25.,
                -FRAC_PI_2 + TAU * (1. - fraction),
                -FRAC_PI_2 + TAU,
            )
            .stroke(),
        Indicator::Expired => track(neutral, scale),
        Indicator::Empty => Painter::new(),
    }
}

fn track(neutral: u32, scale: f32) -> Painter {
    Painter::new()
        .scale(scale, scale)
        .stroke_color(color(rgb(neutral)))
        .line_width(4.)
        .begin_path()
        .arc(48., 36., 25., 0., TAU)
        .stroke()
}

fn resize_handles() -> [View; 8] {
    use ResizeEdge::*;
    [
        view()
            .absolute()
            .left_0()
            .right_0()
            .top_0()
            .h(px(4.))
            .window_resize(Top),
        view()
            .absolute()
            .left_0()
            .right_0()
            .bottom_0()
            .h(px(4.))
            .window_resize(Bottom),
        view()
            .absolute()
            .left_0()
            .top_0()
            .bottom_0()
            .w(px(4.))
            .window_resize(Left),
        view()
            .absolute()
            .right_0()
            .top_0()
            .bottom_0()
            .w(px(4.))
            .window_resize(Right),
        view()
            .absolute()
            .top_0()
            .left_0()
            .size(px(8.))
            .window_resize(TopLeft),
        view()
            .absolute()
            .top_0()
            .right_0()
            .size(px(8.))
            .window_resize(TopRight),
        view()
            .absolute()
            .bottom_0()
            .left_0()
            .size(px(8.))
            .window_resize(BottomLeft),
        view()
            .absolute()
            .bottom_0()
            .right_0()
            .size(px(8.))
            .window_resize(BottomRight),
    ]
}
