#![cfg_attr(target_os = "windows", windows_subsystem = "windows")]

#[cfg(target_os = "linux")]
use nsh_platform_linux as platform;
#[cfg(target_os = "macos")]
use nsh_platform_macos as platform;
#[cfg(target_os = "windows")]
use nsh_platform_windows as platform;

fn main() -> impl std::process::Termination {
    platform::run_app(
        platform::AppSource::Rust(
            platform::Root::of::<countdowner::App>().app_id("local.countdowner"),
        ),
        platform::AppOptions {
            title: "Countdowner".to_owned(),
            app_id: "local.countdowner".to_owned(),
            #[cfg(any(target_os = "windows", target_os = "linux"))]
            single_instance: true,
            login_item: true,
            ..Default::default()
        },
    )
}
