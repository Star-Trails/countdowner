use std::collections::HashSet;
use std::fs;
use std::io::{self, Write};
use std::path::Path;
use std::time::Duration;

use chrono::{
    DateTime, Datelike, FixedOffset, Local, LocalResult, NaiveDateTime, TimeZone, Timelike, Utc,
};
use serde::{Deserialize, Deserializer, Serialize};
use uuid::Uuid;

const CONFIG_VERSION: i32 = 1;

const MILLI_NANOS: i128 = 1_000_000;
const SEC_NANOS: i128 = 1_000_000_000;
const MIN_NANOS: i128 = 60 * SEC_NANOS;
const HOUR_NANOS: i128 = 60 * MIN_NANOS;
const DAY_NANOS: i128 = 24 * HOUR_NANOS;

const URGENCY_GREEN: usize = 0;
const URGENCY_YELLOW: usize = 1;
const URGENCY_RED: usize = 2;
const URGENCY_ORANGE: usize = 3;
const URGENCY_ORANGE_RED: usize = 4;

#[derive(Clone, Debug, PartialEq, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct Countdown {
    pub(crate) id: String,
    pub(crate) title: String,
    pub(crate) target: DateTime<FixedOffset>,
    #[serde(default, deserialize_with = "deserialize_null_default")]
    pub(crate) time_zone: String,
}

#[derive(Clone, Debug, PartialEq, Deserialize, Serialize)]
pub(crate) struct Position {
    #[serde(default, deserialize_with = "deserialize_null_default")]
    pub(crate) monitor: String,
    #[serde(default)]
    pub(crate) x: f64,
    #[serde(default)]
    pub(crate) y: f64,
}

impl Default for Position {
    fn default() -> Self {
        Self {
            monitor: String::new(),
            x: 40.0,
            y: 40.0,
        }
    }
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(default)]
pub(crate) struct WidgetSize {
    pub(crate) width: i32,
    pub(crate) height: i32,
}

impl WidgetSize {
    fn is_auto(&self) -> bool {
        *self == Self::default()
    }
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Theme {
    #[default]
    System,
    Light,
    Dark,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Sort {
    #[default]
    Deadline,
    Manual,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Language {
    #[default]
    En,
    #[serde(rename = "zh-Hant")]
    ZhHant,
    #[serde(rename = "zh-Hans")]
    ZhHans,
    Ja,
    Ko,
}

impl Language {
    pub(crate) fn code(self) -> &'static str {
        match self {
            Self::En => "en",
            Self::ZhHant => "zh-Hant",
            Self::ZhHans => "zh-Hans",
            Self::Ja => "ja",
            Self::Ko => "ko",
        }
    }

    pub(crate) fn from_locale(locale: &str) -> Self {
        let base = locale.trim().split(['.', '@']).next().unwrap_or("");
        let mut parts = base.split(['-', '_']).filter(|part| !part.is_empty());
        let Some(lang) = parts.next() else {
            return Self::En;
        };
        if lang.eq_ignore_ascii_case("ja") || lang.eq_ignore_ascii_case("jpn") {
            return Self::Ja;
        }
        if lang.eq_ignore_ascii_case("ko") || lang.eq_ignore_ascii_case("kor") {
            return Self::Ko;
        }
        if lang.eq_ignore_ascii_case("zh")
            || lang.eq_ignore_ascii_case("zho")
            || lang.eq_ignore_ascii_case("chi")
        {
            for tag in parts {
                if tag.eq_ignore_ascii_case("hant")
                    || tag.eq_ignore_ascii_case("tw")
                    || tag.eq_ignore_ascii_case("hk")
                    || tag.eq_ignore_ascii_case("mo")
                    || tag.eq_ignore_ascii_case("cht")
                {
                    return Self::ZhHant;
                }
                if tag.eq_ignore_ascii_case("hans")
                    || tag.eq_ignore_ascii_case("cn")
                    || tag.eq_ignore_ascii_case("sg")
                    || tag.eq_ignore_ascii_case("chs")
                {
                    return Self::ZhHans;
                }
            }
            return Self::ZhHans;
        }
        Self::En
    }
}

#[derive(Clone, Debug, PartialEq, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct Config {
    #[serde(default)]
    version: i32,
    #[serde(default, deserialize_with = "deserialize_null_default")]
    pub(crate) countdowns: Vec<Countdown>,
    pub(crate) sort: Sort,
    pub(crate) theme: Theme,
    pub(crate) language: Language,
    #[serde(default)]
    pub(crate) autostart: bool,
    #[serde(default = "empty_position", deserialize_with = "deserialize_position")]
    pub(crate) position: Position,
    #[serde(
        default,
        deserialize_with = "deserialize_null_default",
        skip_serializing_if = "WidgetSize::is_auto"
    )]
    pub(crate) widget_size: WidgetSize,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            version: CONFIG_VERSION,
            countdowns: Vec::new(),
            sort: Sort::default(),
            theme: Theme::default(),
            language: Language::default(),
            autostart: false,
            position: Position::default(),
            widget_size: WidgetSize::default(),
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum DraftError {
    Title,
    Date,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum Remaining {
    Expired,
    Days(i64),
    Clock { hours: i64, minutes: i64 },
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) enum Indicator {
    Empty,
    Expired,
    Grid { days: i64, urgency: usize },
    Ring { fraction: f32, urgency: usize },
}

impl Countdown {
    pub(crate) fn from_draft(
        title: &str,
        target: NaiveDateTime,
        original: Option<&Countdown>,
    ) -> Result<Self, DraftError> {
        Self::from_draft_in_tz(title, target, original, &Local, "Local")
    }

    fn from_draft_in_tz<Tz: TimeZone>(
        title: &str,
        target: NaiveDateTime,
        original: Option<&Countdown>,
        tz: &Tz,
        tz_name: &str,
    ) -> Result<Self, DraftError> {
        let title = title.trim();
        if title.is_empty() {
            return Err(DraftError::Title);
        }

        if let Some(orig) = original {
            let orig_local = orig.target.with_timezone(tz).naive_local();
            if orig_local.date() == target.date()
                && orig_local.hour() == target.hour()
                && orig_local.minute() == target.minute()
            {
                return Ok(Self {
                    id: orig.id.clone(),
                    title: title.to_string(),
                    target: orig.target,
                    time_zone: orig.time_zone.clone(),
                });
            }
        }

        let resolved = match tz.from_local_datetime(&target) {
            LocalResult::Single(dt) => dt.fixed_offset(),
            LocalResult::Ambiguous(earliest, _) => earliest.fixed_offset(),
            LocalResult::None => return Err(DraftError::Date),
        };
        if !(1..=9999).contains(&resolved.year()) || is_zero_target(resolved) {
            return Err(DraftError::Date);
        }

        let id = match original {
            Some(orig) => orig.id.clone(),
            None => Uuid::new_v4().simple().to_string(),
        };
        Ok(Self {
            id,
            title: title.to_string(),
            target: resolved,
            time_zone: tz_name.to_string(),
        })
    }
}

impl Config {
    fn validate(&self) -> Result<(), String> {
        if self.version != CONFIG_VERSION {
            return Err(format!(
                "unsupported config version {}; original file is preserved",
                self.version
            ));
        }
        if self.widget_size.width < 0
            || self.widget_size.height < 0
            || (self.widget_size.width == 0) != (self.widget_size.height == 0)
        {
            return Err("invalid widget size".to_string());
        }
        if !self.position.x.is_finite() || !self.position.y.is_finite() {
            return Err("invalid widget position".to_string());
        }
        let mut seen = HashSet::with_capacity(self.countdowns.len());
        for item in &self.countdowns {
            if item.id.is_empty()
                || !seen.insert(item.id.as_str())
                || item.title.trim().is_empty()
                || !(0..=9999).contains(&item.target.year())
                || is_zero_target(item.target)
            {
                return Err("invalid countdown configuration".to_string());
            }
        }
        Ok(())
    }

    pub(crate) fn load(path: &Path) -> Result<Option<Self>, String> {
        let bytes = match fs::read(path) {
            Ok(bytes) => bytes,
            Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(None),
            Err(err) => return Err(format!("failed to read config: {err}")),
        };
        let config: Self = serde_json::from_slice(&bytes)
            .map_err(|err| format!("failed to parse config; original file is preserved: {err}"))?;
        config.validate()?;
        Ok(Some(config))
    }

    pub(crate) fn save(&self, path: &Path) -> Result<(), String> {
        self.validate()?;
        let mut json = serde_json::to_string_pretty(self)
            .map_err(|err| format!("failed to serialize config: {err}"))?;
        json.push('\n');

        let dir = match path.parent() {
            Some(parent) if !parent.as_os_str().is_empty() => {
                fs::create_dir_all(parent)
                    .map_err(|err| format!("failed to create config directory: {err}"))?;
                parent
            }
            Some(_) => Path::new("."),
            None => return Err("invalid config path".to_string()),
        };

        let mut tmp = tempfile::Builder::new()
            .prefix(".config-")
            .suffix(".tmp")
            .tempfile_in(dir)
            .map_err(|err| format!("failed to create temporary config file: {err}"))?;
        tmp.as_file_mut()
            .write_all(json.as_bytes())
            .map_err(|err| format!("failed to write temporary config file: {err}"))?;
        tmp.as_file()
            .sync_all()
            .map_err(|err| format!("failed to sync temporary config file: {err}"))?;
        tmp.persist(path)
            .map_err(|err| format!("failed to replace config file: {}", err.error))?;
        Ok(())
    }

    pub(crate) fn ordered(&self) -> Vec<&Countdown> {
        let mut items: Vec<&Countdown> = self.countdowns.iter().collect();
        if self.sort == Sort::Deadline {
            items.sort_by(|a, b| a.target.cmp(&b.target));
        }
        items
    }

    /// Moves `id` onto `anchor`'s position in the visible order and switches to
    /// manual sorting. Moving down inserts after `anchor`; moving up inserts
    /// before it.
    pub(crate) fn reorder(&mut self, id: &str, anchor: &str) -> bool {
        if id == anchor {
            return false;
        }
        let mut order: Vec<usize> = (0..self.countdowns.len()).collect();
        if self.sort == Sort::Deadline {
            order.sort_by(|&a, &b| self.countdowns[a].target.cmp(&self.countdowns[b].target));
        }
        let Some(from) = order.iter().position(|&idx| self.countdowns[idx].id == id) else {
            return false;
        };
        let Some(to) = order
            .iter()
            .position(|&idx| self.countdowns[idx].id == anchor)
        else {
            return false;
        };
        if from == to {
            return false;
        }

        let moved = order.remove(from);
        order.insert(to, moved);

        let mut slots: Vec<Option<Countdown>> = self.countdowns.drain(..).map(Some).collect();
        self.countdowns = order
            .into_iter()
            .map(|idx| slots[idx].take().expect("unique index in reorder"))
            .collect();
        self.sort = Sort::Manual;
        true
    }

    /// Keeps minute-level progress fresh and catches earlier text, mode, and
    /// urgency transitions. Expired-only or empty lists need no clock refresh.
    pub(crate) fn next_refresh(&self, now: DateTime<Utc>) -> Option<Duration> {
        const BOUNDARIES: [i128; 6] = [
            31 * DAY_NANOS,
            20 * DAY_NANOS,
            10 * DAY_NANOS,
            DAY_NANOS,
            12 * HOUR_NANOS,
            3 * HOUR_NANOS,
        ];

        let mut next: Option<i128> = None;
        for item in &self.countdowns {
            let Some(d) = remaining_nanos(item.target, now) else {
                continue;
            };
            let unit = if d > DAY_NANOS { DAY_NANOS } else { MIN_NANOS };
            let mut delay = MIN_NANOS.min((d - 1) % unit + 1);
            for boundary in BOUNDARIES {
                if d > boundary {
                    delay = delay.min(d - boundary);
                }
            }
            // At these exact instants the next representable duration changes
            // the time unit or color; don't retain that frame for a full minute.
            if d == DAY_NANOS || d == 12 * HOUR_NANOS {
                delay = delay.min(MILLI_NANOS);
            }
            next = Some(match next {
                Some(cur) => cur.min(delay),
                None => delay,
            });
        }
        next.map(|nanos| Duration::from_nanos(nanos as u64))
    }
}

pub(crate) fn remaining(target: DateTime<FixedOffset>, now: DateTime<Utc>) -> Remaining {
    let Some(d) = remaining_nanos(target, now) else {
        return Remaining::Expired;
    };
    if d >= DAY_NANOS {
        return Remaining::Days(((d - 1) / DAY_NANOS + 1) as i64);
    }
    let minutes = ((d - 1) / MIN_NANOS + 1) as i64;
    Remaining::Clock {
        hours: minutes / 60,
        minutes: minutes % 60,
    }
}

pub(crate) fn indicator(target: DateTime<FixedOffset>, now: DateTime<Utc>) -> Indicator {
    let Some(d) = remaining_nanos(target, now) else {
        return Indicator::Expired;
    };
    if d > 31 * DAY_NANOS {
        return Indicator::Empty;
    }
    if d > DAY_NANOS {
        let days = ((d - 1) / DAY_NANOS + 1) as i64;
        let urgency = if days >= 21 {
            URGENCY_GREEN
        } else if days >= 11 {
            URGENCY_YELLOW
        } else {
            URGENCY_RED
        };
        return Indicator::Grid { days, urgency };
    }
    let urgency = if d >= 12 * HOUR_NANOS {
        URGENCY_ORANGE
    } else if d > 3 * HOUR_NANOS {
        URGENCY_ORANGE_RED
    } else {
        URGENCY_RED
    };
    Indicator::Ring {
        fraction: (d as f64 / DAY_NANOS as f64) as f32,
        urgency,
    }
}

fn remaining_nanos(target: DateTime<FixedOffset>, now: DateTime<Utc>) -> Option<i128> {
    if target <= now {
        return None;
    }
    let delta = target.signed_duration_since(now);
    Some(i128::from(delta.num_seconds()) * SEC_NANOS + i128::from(delta.subsec_nanos()))
}

fn is_zero_target(target: DateTime<FixedOffset>) -> bool {
    let utc = target.naive_utc();
    utc.year() == 1
        && utc.month() == 1
        && utc.day() == 1
        && utc.hour() == 0
        && utc.minute() == 0
        && utc.second() == 0
        && utc.nanosecond() == 0
}

fn deserialize_null_default<'de, D, T>(deserializer: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Default + Deserialize<'de>,
{
    Ok(Option::<T>::deserialize(deserializer)?.unwrap_or_default())
}

fn empty_position() -> Position {
    Position {
        monitor: String::new(),
        x: 0.0,
        y: 0.0,
    }
}

fn deserialize_position<'de, D: Deserializer<'de>>(deserializer: D) -> Result<Position, D::Error> {
    Ok(Option::<Position>::deserialize(deserializer)?.unwrap_or_else(empty_position))
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::{NaiveDate, TimeDelta};

    fn utc(y: i32, m: u32, d: u32, hh: u32, mm: u32, ss: u32) -> DateTime<Utc> {
        NaiveDate::from_ymd_opt(y, m, d)
            .unwrap()
            .and_hms_opt(hh, mm, ss)
            .unwrap()
            .and_utc()
    }

    fn fixed_utc(dt: DateTime<Utc>) -> DateTime<FixedOffset> {
        dt.fixed_offset()
    }

    fn ids(config: &Config) -> Vec<&str> {
        config
            .ordered()
            .into_iter()
            .map(|c| c.id.as_str())
            .collect()
    }

    #[test]
    fn remaining_boundaries() {
        let now = utc(2026, 10, 10, 12, 0, 0);
        let cases = [
            (TimeDelta::hours(25), Remaining::Days(2)),
            (TimeDelta::hours(24), Remaining::Days(1)),
            (
                TimeDelta::hours(24) - TimeDelta::seconds(1),
                Remaining::Clock {
                    hours: 24,
                    minutes: 0,
                },
            ),
            (
                TimeDelta::minutes(90),
                Remaining::Clock {
                    hours: 1,
                    minutes: 30,
                },
            ),
            (
                TimeDelta::minutes(59) + TimeDelta::seconds(1),
                Remaining::Clock {
                    hours: 1,
                    minutes: 0,
                },
            ),
            (
                TimeDelta::seconds(30),
                Remaining::Clock {
                    hours: 0,
                    minutes: 1,
                },
            ),
            (
                TimeDelta::nanoseconds(1),
                Remaining::Clock {
                    hours: 0,
                    minutes: 1,
                },
            ),
            (TimeDelta::zero(), Remaining::Expired),
            (-TimeDelta::minutes(1), Remaining::Expired),
        ];
        for (delta, want) in cases {
            assert_eq!(
                remaining(fixed_utc(now + delta), now),
                want,
                "delta={delta}"
            );
        }
    }

    #[test]
    fn indicator_boundaries_and_partial_days() {
        let now = utc(2026, 12, 31, 12, 0, 0);
        let day = TimeDelta::hours(24);
        let ns = TimeDelta::nanoseconds(1);
        let cases = [
            (
                day * 25,
                Indicator::Grid {
                    days: 25,
                    urgency: 0,
                },
            ),
            (
                day * 15,
                Indicator::Grid {
                    days: 15,
                    urgency: 1,
                },
            ),
            (
                day * 5,
                Indicator::Grid {
                    days: 5,
                    urgency: 2,
                },
            ),
            (
                day * 31,
                Indicator::Grid {
                    days: 31,
                    urgency: 0,
                },
            ),
            (day * 31 + ns, Indicator::Empty),
            (day * 45, Indicator::Empty),
            (
                day * 20 + ns,
                Indicator::Grid {
                    days: 21,
                    urgency: 0,
                },
            ),
            (
                day * 20,
                Indicator::Grid {
                    days: 20,
                    urgency: 1,
                },
            ),
            (
                day * 10 + ns,
                Indicator::Grid {
                    days: 11,
                    urgency: 1,
                },
            ),
            (
                day * 10,
                Indicator::Grid {
                    days: 10,
                    urgency: 2,
                },
            ),
            (
                day + ns,
                Indicator::Grid {
                    days: 2,
                    urgency: 2,
                },
            ),
            (
                day,
                Indicator::Ring {
                    fraction: 1.0,
                    urgency: 3,
                },
            ),
            (
                TimeDelta::hours(12),
                Indicator::Ring {
                    fraction: 0.5,
                    urgency: 3,
                },
            ),
            (
                TimeDelta::hours(6),
                Indicator::Ring {
                    fraction: 0.25,
                    urgency: 4,
                },
            ),
            (
                TimeDelta::hours(3),
                Indicator::Ring {
                    fraction: 0.125,
                    urgency: 2,
                },
            ),
            (
                TimeDelta::minutes(1),
                Indicator::Ring {
                    fraction: (1.0_f64 / 1440.0_f64) as f32,
                    urgency: 2,
                },
            ),
            (TimeDelta::zero(), Indicator::Expired),
            (-TimeDelta::hours(1), Indicator::Expired),
        ];
        for (delta, want) in cases {
            assert_eq!(
                indicator(fixed_utc(now + delta), now),
                want,
                "delta={delta}"
            );
        }

        // Ceil exact duration; partial days do not use calendar boundaries.
        for days in 2..=31_i64 {
            let got = indicator(fixed_utc(now + day * (days as i32 - 1) + ns), now);
            assert!(
                matches!(got, Indicator::Grid { days: d, .. } if d == days),
                "days={days}, got={got:?}"
            );
        }

        // Calendar month/leap-year 48h boundaries and 23h/25h DST lengths.
        for date in [
            utc(2025, 2, 28, 12, 0, 0),
            utc(2024, 2, 28, 12, 0, 0),
            utc(2024, 2, 29, 12, 0, 0),
            utc(2026, 4, 30, 12, 0, 0),
            utc(2026, 7, 31, 12, 0, 0),
            utc(2026, 12, 31, 12, 0, 0),
        ] {
            assert_eq!(
                indicator(fixed_utc(date + day * 2), date),
                Indicator::Grid {
                    days: 2,
                    urgency: 2
                }
            );
        }
        assert_eq!(
            indicator(fixed_utc(now + TimeDelta::hours(23)), now),
            Indicator::Ring {
                fraction: (23.0_f64 / 24.0_f64) as f32,
                urgency: 3,
            }
        );
        assert_eq!(
            indicator(fixed_utc(now + TimeDelta::hours(25)), now),
            Indicator::Grid {
                days: 2,
                urgency: 2
            }
        );
    }

    #[test]
    fn next_refresh_boundaries() {
        let now = utc(2026, 10, 10, 12, 0, 0);
        let day = TimeDelta::hours(24);
        let mut config = Config::default();
        assert_eq!(config.next_refresh(now), None);

        let cases = [
            (day * 45, Some(Duration::from_secs(60))),
            (
                day * 31 + TimeDelta::seconds(5),
                Some(Duration::from_secs(5)),
            ),
            (
                day * 20 + TimeDelta::seconds(7),
                Some(Duration::from_secs(7)),
            ),
            (
                day * 10 + TimeDelta::seconds(9),
                Some(Duration::from_secs(9)),
            ),
            (day + TimeDelta::seconds(10), Some(Duration::from_secs(10))),
            (day, Some(Duration::from_millis(1))),
            (
                TimeDelta::hours(12) + TimeDelta::seconds(10),
                Some(Duration::from_secs(10)),
            ),
            (TimeDelta::hours(12), Some(Duration::from_millis(1))),
            (
                TimeDelta::hours(3) + TimeDelta::seconds(10),
                Some(Duration::from_secs(10)),
            ),
            (
                TimeDelta::hours(2) + TimeDelta::seconds(30),
                Some(Duration::from_secs(30)),
            ),
            (TimeDelta::seconds(15), Some(Duration::from_secs(15))),
            (TimeDelta::zero(), None),
            (-TimeDelta::hours(2), None),
        ];
        for (delta, want) in cases {
            config.countdowns = vec![Countdown {
                id: "a".to_string(),
                title: "A".to_string(),
                target: fixed_utc(now + delta),
                time_zone: "UTC".to_string(),
            }];
            assert_eq!(config.next_refresh(now), want, "delta={delta}");
        }

        // Picks earliest refresh across multiple items while ignoring expired ones.
        config.countdowns = vec![
            Countdown {
                id: "past".to_string(),
                title: "Past".to_string(),
                target: fixed_utc(now - TimeDelta::seconds(5)),
                time_zone: "UTC".to_string(),
            },
            Countdown {
                id: "far".to_string(),
                title: "Far".to_string(),
                target: fixed_utc(now + day * 45),
                time_zone: "UTC".to_string(),
            },
            Countdown {
                id: "soon".to_string(),
                title: "Soon".to_string(),
                target: fixed_utc(now + TimeDelta::seconds(12)),
                time_zone: "UTC".to_string(),
            },
        ];
        assert_eq!(config.next_refresh(now), Some(Duration::from_secs(12)));
    }

    #[test]
    fn go_v1_compatibility_atomic_save_and_validation() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("config.json");

        // Missing file loads as Ok(None).
        assert_eq!(Config::load(&path).unwrap(), None);

        // Go v1 JSON with missing widgetSize and null countdowns is valid.
        let go_null_json = r#"{
  "version": 1,
  "countdowns": null,
  "sort": "deadline",
  "theme": "system",
  "language": "en",
  "autostart": false,
  "position": {
    "monitor": "",
    "x": 40,
    "y": 40
  }
}
"#;
        fs::write(&path, go_null_json).unwrap();
        let loaded = Config::load(&path).unwrap().unwrap();
        assert_eq!(loaded, Config::default());

        // Round-trip including non-UTC offset, CJK titles, negative monitor x, and existing-file replacement.
        let plus9 = FixedOffset::east_opt(9 * 3600).unwrap();
        let first = Countdown {
            id: "0123456789abcdef0123456789abcdef".to_string(),
            title: "專案交付 日本語 한국어".to_string(),
            target: plus9.with_ymd_and_hms(2026, 12, 2, 12, 0, 0).unwrap(),
            time_zone: "Asia/Tokyo".to_string(),
        };
        let second = Countdown {
            id: "fedcba9876543210fedcba9876543210".to_string(),
            title: "English 简体中文".to_string(),
            target: fixed_utc(utc(2026, 12, 1, 12, 0, 0)),
            time_zone: "UTC".to_string(),
        };
        let mut config = Config {
            version: CONFIG_VERSION,
            countdowns: vec![first.clone(), second.clone()],
            sort: Sort::Deadline,
            theme: Theme::Dark,
            language: Language::ZhHant,
            autostart: true,
            position: Position {
                monitor: "monitor-id".to_string(),
                x: -80.5,
                y: 40.25,
            },
            widget_size: WidgetSize::default(),
        };
        config.save(&path).unwrap();
        let raw_auto = fs::read_to_string(&path).unwrap();
        assert!(raw_auto.contains("\"version\": 1"));
        assert!(!raw_auto.contains("\"widgetSize\""));

        config.position.x = 120.0;
        config.widget_size = WidgetSize {
            width: 480,
            height: 360,
        };
        config.save(&path).unwrap();
        let raw_sized = fs::read_to_string(&path).unwrap();
        assert!(raw_sized.contains("\"widgetSize\""));

        let loaded = Config::load(&path).unwrap().unwrap();
        assert_eq!(loaded, config);
        assert_eq!(loaded.countdowns[0].target.offset(), &plus9);
        assert_eq!(loaded.ordered()[0].id, second.id);
        assert_eq!(loaded.countdowns[0].id, first.id);

        // Damaged or unsupported configs must error and preserve disk contents untouched.
        let invalid_payloads = [
            "{broken",
            r#"{"version":2,"countdowns":[],"sort":"deadline","theme":"system","language":"en","position":{"monitor":"","x":0,"y":0}}"#,
            r#"{"countdowns":[],"sort":"deadline","theme":"system","language":"en","position":{"monitor":"","x":0,"y":0}}"#,
            r#"{"version":1,"countdowns":[],"sort":"alpha","theme":"system","language":"en","position":{"monitor":"","x":0,"y":0}}"#,
            r#"{"version":1,"countdowns":[],"sort":"deadline","theme":"sepia","language":"en","position":{"monitor":"","x":0,"y":0}}"#,
            r#"{"version":1,"countdowns":[],"sort":"deadline","theme":"system","language":"fr","position":{"monitor":"","x":0,"y":0}}"#,
            r#"{"version":1,"countdowns":[],"sort":"deadline","theme":"system","language":"en","position":{"monitor":"","x":0,"y":0},"widgetSize":{"width":400,"height":0}}"#,
            r#"{"version":1,"countdowns":[],"sort":"deadline","theme":"system","language":"en","position":{"monitor":"","x":0,"y":0},"widgetSize":{"width":-10,"height":200}}"#,
            r#"{"version":1,"countdowns":[{"id":"","title":"A","target":"2026-12-01T00:00:00Z","timeZone":"UTC"}],"sort":"deadline","theme":"system","language":"en"}"#,
            r#"{"version":1,"countdowns":[{"id":"dup","title":"A","target":"2026-12-01T00:00:00Z","timeZone":"UTC"},{"id":"dup","title":"B","target":"2026-12-02T00:00:00Z","timeZone":"UTC"}],"sort":"deadline","theme":"system","language":"en"}"#,
            r#"{"version":1,"countdowns":[{"id":"a","title":"   ","target":"2026-12-01T00:00:00Z","timeZone":"UTC"}],"sort":"deadline","theme":"system","language":"en"}"#,
            r#"{"version":1,"countdowns":[{"id":"a","title":"Zero","target":"0001-01-01T00:00:00Z","timeZone":"UTC"}],"sort":"deadline","theme":"system","language":"en"}"#,
            r#"{"version":1,"countdowns":[{"id":"a","title":"BadDate","target":"2026-12-01","timeZone":"UTC"}],"sort":"deadline","theme":"system","language":"en"}"#,
        ];
        for payload in invalid_payloads {
            fs::write(&path, payload).unwrap();
            assert!(Config::load(&path).is_err(), "expected err for {payload}");
            assert_eq!(fs::read_to_string(&path).unwrap(), payload);
        }

        // Invalid in-memory config cannot overwrite an existing file.
        let valid_snapshot = raw_sized;
        fs::write(&path, &valid_snapshot).unwrap();
        let mut bad = config.clone();
        bad.widget_size = WidgetSize {
            width: 300,
            height: 0,
        };
        assert!(bad.save(&path).is_err());
        assert_eq!(fs::read_to_string(&path).unwrap(), valid_snapshot);
    }

    #[test]
    fn manual_reorder_and_restore_deadline_sort() {
        let now = utc(2026, 10, 10, 12, 0, 0);
        let day = TimeDelta::hours(24);
        let mut config = Config::default();
        assert_eq!(config.sort, Sort::Deadline);
        assert_eq!(
            config.position,
            Position {
                monitor: String::new(),
                x: 40.0,
                y: 40.0
            }
        );
        assert_eq!(
            config.widget_size,
            WidgetSize {
                width: 0,
                height: 0
            }
        );

        config.countdowns = vec![
            Countdown {
                id: "late".to_string(),
                title: "Late".to_string(),
                target: fixed_utc(now + day * 3),
                time_zone: String::new(),
            },
            Countdown {
                id: "early".to_string(),
                title: "Early".to_string(),
                target: fixed_utc(now + day),
                time_zone: String::new(),
            },
            Countdown {
                id: "middle".to_string(),
                title: "Middle".to_string(),
                target: fixed_utc(now + day * 2),
                time_zone: String::new(),
            },
        ];
        assert_eq!(ids(&config), vec!["early", "middle", "late"]);

        // Missing or identical IDs must be no-ops without changing sort mode.
        assert!(!config.reorder("missing", "early"));
        assert!(!config.reorder("early", "missing"));
        assert!(!config.reorder("early", "early"));
        assert_eq!(config.sort, Sort::Deadline);

        // Downward reorder inserts after anchor and enters manual mode.
        assert!(config.reorder("early", "late"));
        assert_eq!(config.sort, Sort::Manual);
        assert_eq!(ids(&config), vec!["middle", "late", "early"]);

        // Upward reorder inserts before anchor.
        assert!(config.reorder("early", "middle"));
        assert_eq!(ids(&config), vec!["early", "middle", "late"]);

        assert!(config.reorder("late", "early"));
        assert_eq!(ids(&config), vec!["late", "early", "middle"]);

        // Persisting and reloading preserves manual order.
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("config.json");
        config.save(&path).unwrap();
        let mut loaded = Config::load(&path).unwrap().unwrap();
        assert_eq!(ids(&loaded), vec!["late", "early", "middle"]);

        // Editing deadline or appending does not disrupt manual order.
        loaded.countdowns[0].target = fixed_utc(now - day);
        assert_eq!(ids(&loaded), vec!["late", "early", "middle"]);
        loaded.countdowns.push(Countdown {
            id: "new".to_string(),
            title: "New".to_string(),
            target: fixed_utc(now),
            time_zone: String::new(),
        });
        assert_eq!(ids(&loaded), vec!["late", "early", "middle", "new"]);

        // Restoring deadline sort orders by target instant.
        loaded.sort = Sort::Deadline;
        assert_eq!(ids(&loaded), vec!["late", "new", "early", "middle"]);
    }

    #[derive(Clone)]
    struct MockDstTz;

    #[allow(deprecated)]
    impl TimeZone for MockDstTz {
        type Offset = FixedOffset;

        fn from_offset(_offset: &Self::Offset) -> Self {
            Self
        }

        fn offset_from_local_date(&self, local: &NaiveDate) -> LocalResult<Self::Offset> {
            self.offset_from_local_datetime(&local.and_hms_opt(0, 0, 0).unwrap())
        }

        fn offset_from_local_datetime(&self, local: &NaiveDateTime) -> LocalResult<Self::Offset> {
            let std_off = FixedOffset::west_opt(5 * 3600).unwrap();
            let dst_off = FixedOffset::west_opt(4 * 3600).unwrap();
            let gap_start = NaiveDate::from_ymd_opt(2026, 3, 8)
                .unwrap()
                .and_hms_opt(2, 0, 0)
                .unwrap();
            let gap_end = NaiveDate::from_ymd_opt(2026, 3, 8)
                .unwrap()
                .and_hms_opt(3, 0, 0)
                .unwrap();
            let ambig_start = NaiveDate::from_ymd_opt(2026, 11, 1)
                .unwrap()
                .and_hms_opt(1, 0, 0)
                .unwrap();
            let ambig_end = NaiveDate::from_ymd_opt(2026, 11, 1)
                .unwrap()
                .and_hms_opt(2, 0, 0)
                .unwrap();
            if *local >= gap_start && *local < gap_end {
                LocalResult::None
            } else if *local >= ambig_start && *local < ambig_end {
                LocalResult::Ambiguous(dst_off, std_off)
            } else if *local >= gap_end && *local < ambig_start {
                LocalResult::Single(dst_off)
            } else {
                LocalResult::Single(std_off)
            }
        }

        fn offset_from_utc_date(&self, utc_date: &NaiveDate) -> Self::Offset {
            self.offset_from_utc_datetime(&utc_date.and_hms_opt(0, 0, 0).unwrap())
        }

        fn offset_from_utc_datetime(&self, utc_dt: &NaiveDateTime) -> Self::Offset {
            let std_off = FixedOffset::west_opt(5 * 3600).unwrap();
            let dst_off = FixedOffset::west_opt(4 * 3600).unwrap();
            let dst_start_utc = NaiveDate::from_ymd_opt(2026, 3, 8)
                .unwrap()
                .and_hms_opt(7, 0, 0)
                .unwrap();
            let dst_end_utc = NaiveDate::from_ymd_opt(2026, 11, 1)
                .unwrap()
                .and_hms_opt(6, 0, 0)
                .unwrap();
            if *utc_dt >= dst_start_utc && *utc_dt < dst_end_utc {
                dst_off
            } else {
                std_off
            }
        }
    }

    #[test]
    fn from_draft_validation_and_instant_preservation() {
        let dt = NaiveDate::from_ymd_opt(2026, 12, 15)
            .unwrap()
            .and_hms_opt(13, 28, 0)
            .unwrap();
        assert_eq!(
            Countdown::from_draft("   ", dt, None),
            Err(DraftError::Title)
        );

        let created = Countdown::from_draft("  Launch  ", dt, None).unwrap();
        assert_eq!(created.title, "Launch");
        assert_eq!(created.id.len(), 32);
        assert!(
            created
                .id
                .chars()
                .all(|c| c.is_ascii_hexdigit() && !c.is_ascii_uppercase())
        );
        assert_eq!(created.time_zone, "Local");

        // Editing title with minute-matched local time preserves exact original instant, offset, ID, and time_zone.
        let plus9 = FixedOffset::east_opt(9 * 3600).unwrap();
        let orig = Countdown {
            id: "keep-id".to_string(),
            title: "Old".to_string(),
            target: plus9.with_ymd_and_hms(2026, 12, 15, 14, 28, 45).unwrap(),
            time_zone: "Asia/Tokyo".to_string(),
        };
        let orig_local_minute = orig
            .target
            .with_timezone(&Local)
            .naive_local()
            .with_second(0)
            .unwrap()
            .with_nanosecond(0)
            .unwrap();
        let edited = Countdown::from_draft("Renamed", orig_local_minute, Some(&orig)).unwrap();
        assert_eq!(edited.id, "keep-id");
        assert_eq!(edited.title, "Renamed");
        assert_eq!(edited.target, orig.target);
        assert_eq!(edited.target.offset(), &plus9);
        assert_eq!(edited.time_zone, "Asia/Tokyo");

        // Changing target minute updates target while keeping ID.
        let shifted = orig_local_minute + TimeDelta::minutes(1);
        let rescheduled = Countdown::from_draft("Moved", shifted, Some(&orig)).unwrap();
        assert_eq!(rescheduled.id, "keep-id");
        assert_ne!(rescheduled.target, orig.target);
        assert_eq!(rescheduled.time_zone, "Local");

        // DST spring-forward gap returns DraftError::Date.
        let gap = NaiveDate::from_ymd_opt(2026, 3, 8)
            .unwrap()
            .and_hms_opt(2, 30, 0)
            .unwrap();
        assert_eq!(
            Countdown::from_draft_in_tz("Gap", gap, None, &MockDstTz, "Mock/DST"),
            Err(DraftError::Date)
        );

        // DST fall-back ambiguous local time picks earliest instant; editing title on the
        // second occurrence preserves the exact second occurrence's offset (-05:00).
        let fall_back = NaiveDate::from_ymd_opt(2026, 11, 1)
            .unwrap()
            .and_hms_opt(1, 30, 0)
            .unwrap();
        let ambig_new =
            Countdown::from_draft_in_tz("Ambig", fall_back, None, &MockDstTz, "Mock/DST").unwrap();
        assert_eq!(
            ambig_new.target.offset(),
            &FixedOffset::west_opt(4 * 3600).unwrap()
        );

        let second_occurrence = Countdown {
            id: "dst-id".to_string(),
            title: "Second".to_string(),
            target: FixedOffset::west_opt(5 * 3600)
                .unwrap()
                .with_ymd_and_hms(2026, 11, 1, 1, 30, 0)
                .unwrap(),
            time_zone: "Mock/DST".to_string(),
        };
        let preserved = Countdown::from_draft_in_tz(
            "Second Edited",
            fall_back,
            Some(&second_occurrence),
            &MockDstTz,
            "Mock/DST",
        )
        .unwrap();
        assert_eq!(preserved.id, "dst-id");
        assert_eq!(preserved.target, second_occurrence.target);
        assert_eq!(
            preserved.target.offset(),
            &FixedOffset::west_opt(5 * 3600).unwrap()
        );
    }

    #[test]
    fn language_codes_and_locale_matching() {
        assert_eq!(Language::En.code(), "en");
        assert_eq!(Language::ZhHant.code(), "zh-Hant");
        assert_eq!(Language::ZhHans.code(), "zh-Hans");
        assert_eq!(Language::Ja.code(), "ja");
        assert_eq!(Language::Ko.code(), "ko");

        let cases = [
            ("", Language::En),
            ("en", Language::En),
            ("en-US", Language::En),
            ("de-DE", Language::En),
            ("zh-Hant", Language::ZhHant),
            ("zh_Hant_TW", Language::ZhHant),
            ("zh-TW", Language::ZhHant),
            ("zh_HK.UTF-8", Language::ZhHant),
            ("zh-MO", Language::ZhHant),
            ("zh-CHT", Language::ZhHant),
            ("zh-Hans", Language::ZhHans),
            ("zh-Hans-HK", Language::ZhHans),
            ("zh-CN", Language::ZhHans),
            ("zh_SG", Language::ZhHans),
            ("zh-CHS", Language::ZhHans),
            ("zh", Language::ZhHans),
            ("ja", Language::Ja),
            ("ja-JP", Language::Ja),
            ("ko", Language::Ko),
            ("ko_KR.UTF-8", Language::Ko),
        ];
        for (locale, want) in cases {
            assert_eq!(Language::from_locale(locale), want, "locale={locale}");
        }
    }
}
