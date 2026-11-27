use std::sync::LazyLock;

use serde_json::Value;
use nsh_core::element::{register_font, spawn};

use crate::model::{Language, Remaining};

static CATALOGS: LazyLock<[Value; 5]> = LazyLock::new(|| {
    [
        include_str!("../locales/en.json"),
        include_str!("../locales/zh-Hant.json"),
        include_str!("../locales/zh-Hans.json"),
        include_str!("../locales/ja.json"),
        include_str!("../locales/ko.json"),
    ]
    .map(|catalog| serde_json::from_str(catalog).expect("valid embedded translation catalog"))
});

pub(crate) const LANGUAGES: [(Language, &str); 5] = [
    (Language::En, "English"),
    (Language::ZhHant, "正體中文"),
    (Language::ZhHans, "简体中文"),
    (Language::Ja, "日本語"),
    (Language::Ko, "한국어"),
];

pub(super) fn register_ui_fonts() {
    spawn(async {
        for font in [
            include_bytes!("../resources/fonts/Latin.ttf").as_slice(),
            include_bytes!("../resources/fonts/TC.ttf").as_slice(),
            include_bytes!("../resources/fonts/SC.ttf").as_slice(),
            include_bytes!("../resources/fonts/JP.ttf").as_slice(),
            include_bytes!("../resources/fonts/KR.ttf").as_slice(),
        ] {
            if let Err(error) = register_font(font).await {
                log::warn!("Could not register Countdowner UI font: {error}");
            }
        }
    });
}

impl Language {
    pub(super) fn ui_font(self) -> &'static str {
        match self {
            Self::En => "Countdowner UI Latin, system-ui",
            Self::ZhHant => "Countdowner UI TC, system-ui",
            Self::ZhHans => "Countdowner UI SC, system-ui",
            Self::Ja => "Countdowner UI JP, system-ui",
            Self::Ko => "Countdowner UI KR, system-ui",
        }
    }

    pub(crate) fn tr(self, key: &str) -> &'static str {
        let index = LANGUAGES
            .iter()
            .position(|(language, _)| *language == self)
            .unwrap();
        CATALOGS[index][key]
            .as_str()
            .unwrap_or_else(|| CATALOGS[0][key].as_str().expect("known translation key"))
    }

    pub(crate) fn remaining(self, remaining: Remaining) -> String {
        match remaining {
            Remaining::Expired => self.tr("expired").to_owned(),
            Remaining::Days(days) => {
                plural(self.tr("remainingDays"), days).replace('#', &days.to_string())
            }
            Remaining::Clock { hours, minutes } => {
                let pattern = self.tr("remainingClock");
                let hours = hours.to_string();
                let minutes = minutes.to_string();
                if self == Self::En {
                    let word = if hours == "1" { "hour" } else { "hours" };
                    format!("{hours} {word} {minutes} min")
                } else {
                    pattern
                        .replace("{hours}", &hours)
                        .replace("{minutes}", &minutes)
                }
            }
        }
    }
}

fn plural(pattern: &str, count: i64) -> &str {
    let one = (count == 1).then(|| pattern.split_once("one {")).flatten();
    one.or_else(|| pattern.split_once("other {"))
        .and_then(|(_, branch)| branch.split_once('}'))
        .map_or(pattern, |(text, _)| text)
}
