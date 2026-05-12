#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use sysinfo::{Disks, Pid, System};
use tauri::{Manager, PhysicalPosition, PhysicalSize};

#[derive(Serialize)]
struct HostProcess {
    pid: u32,
    name: String,
    memory_bytes: u64,
    virtual_memory_bytes: u64,
    cpu_usage_percent: f32,
    run_time_seconds: u64,
}

#[derive(Serialize)]
struct HostDisk {
    name: String,
    mount_point: String,
    file_system: String,
    total_bytes: u64,
    available_bytes: u64,
    used_bytes: u64,
}

#[derive(Serialize)]
struct HostDiagnostics {
    host_name: String,
    os_name: String,
    os_version: String,
    kernel_version: Option<String>,
    architecture: Option<String>,
    uptime_seconds: u64,
    cpu_count: usize,
    total_memory_bytes: u64,
    available_memory_bytes: u64,
    used_memory_bytes: u64,
    total_swap_bytes: u64,
    available_swap_bytes: u64,
    used_swap_bytes: u64,
    process: Option<HostProcess>,
    disks: Vec<HostDisk>,
}

#[derive(Serialize, Clone)]
struct OperatorApp {
    id: String,
    label: String,
    description: String,
    executable: String,
    category: String,
    icon_name: Option<String>,
    icon_path: Option<String>,
    desktop_file: Option<String>,
    native: bool,
}

struct OperatorAppDefinition {
    id: &'static str,
    label: &'static str,
    description: &'static str,
    executable: &'static str,
    category: &'static str,
    desktop_ids: &'static [&'static str],
    launch_args: &'static [&'static str],
}

#[derive(Serialize)]
struct OperatorAppLaunchResult {
    app_id: String,
    label: String,
    executable: String,
    launched: bool,
    pid: Option<u32>,
    message: String,
}

#[derive(Serialize, Clone, Debug, PartialEq, Eq)]
struct LinuxWindowSnapshot {
    id: String,
    title: String,
    app_id: String,
    icon_name: Option<String>,
    icon_path: Option<String>,
    focused: bool,
    minimized: bool,
    native: bool,
}

#[derive(Deserialize)]
struct LswtOutput {
    toplevels: Vec<LswtToplevel>,
}

#[derive(Deserialize)]
struct LswtToplevel {
    identifier: String,
    title: String,
    #[serde(rename = "app-id")]
    app_id: String,
    activated: Option<bool>,
    minimized: Option<bool>,
}

const OPERATOR_APPS: &[OperatorAppDefinition] = &[
    OperatorAppDefinition {
        id: "terminal",
        label: "Terminal",
        description: "Open a Foot terminal in the current FORGE operator session.",
        executable: "foot",
        category: "Workspace",
        desktop_ids: &["foot.desktop"],
        launch_args: &["--working-directory=/projectforge"],
    },
    OperatorAppDefinition {
        id: "files",
        label: "Files",
        description: "Open the PCManFM file manager in the current FORGE operator session.",
        executable: "pcmanfm",
        category: "Workspace",
        desktop_ids: &["pcmanfm.desktop"],
        launch_args: &["/projectforge"],
    },
    OperatorAppDefinition {
        id: "editor",
        label: "Editor",
        description: "Open the fixed operator text editor wrapper for the FORGE workspace.",
        executable: "foot",
        category: "Workspace",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-editor"],
    },
    OperatorAppDefinition {
        id: "archive-manager",
        label: "Archive Manager",
        description: "Open Xarchiver for inspecting and unpacking local archives.",
        executable: "xarchiver",
        category: "Workspace",
        desktop_ids: &["xarchiver.desktop"],
        launch_args: &[],
    },
    OperatorAppDefinition {
        id: "browser",
        label: "Browser",
        description: "Open Firefox for local docs, web consoles, and model tooling.",
        executable: "firefox",
        category: "Internet",
        desktop_ids: &["firefox.desktop"],
        launch_args: &[],
    },
    OperatorAppDefinition {
        id: "api-health",
        label: "API Health",
        description: "Run a fixed FORGE core health probe in a terminal.",
        executable: "foot",
        category: "Internet",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-core-status"],
    },
    OperatorAppDefinition {
        id: "ollama-status",
        label: "Ollama Status",
        description:
            "Show local Ollama process and model status without loading or unloading models.",
        executable: "foot",
        category: "AI Runtime",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-ollama-status"],
    },
    OperatorAppDefinition {
        id: "modelruntime-status",
        label: "Modelruntime Status",
        description: "Show governed FORGE modelruntime status through the local core API.",
        executable: "foot",
        category: "AI Runtime",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-models"],
    },
    OperatorAppDefinition {
        id: "system-monitor",
        label: "System Monitor",
        description: "Open the fixed btop/htop process monitor wrapper.",
        executable: "foot",
        category: "System",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-btop"],
    },
    OperatorAppDefinition {
        id: "core-logs",
        label: "Core Logs",
        description: "Show recent forge-core journal logs through a fixed read-only wrapper.",
        executable: "foot",
        category: "System",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-core-logs"],
    },
    OperatorAppDefinition {
        id: "network-diagnostics",
        label: "Network Diagnostics",
        description: "Show fixed read-only address, route, socket, and DNS diagnostics.",
        executable: "foot",
        category: "System",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-network-diagnostics"],
    },
    OperatorAppDefinition {
        id: "hardware-diagnostics",
        label: "Hardware Diagnostics",
        description: "Show fixed read-only PCI, USB, and process file diagnostics.",
        executable: "foot",
        category: "System",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-hardware-diagnostics"],
    },
    OperatorAppDefinition {
        id: "sqlite-browser",
        label: "SQLite Browser",
        description: "Open DB Browser for SQLite for local database inspection.",
        executable: "sqlitebrowser",
        category: "Developer",
        desktop_ids: &["sqlitebrowser.desktop"],
        launch_args: &[],
    },
    OperatorAppDefinition {
        id: "lazygit",
        label: "Git UI",
        description: "Open lazygit in the FORGE workspace through a fixed terminal wrapper.",
        executable: "foot",
        category: "Developer",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-lazygit"],
    },
    OperatorAppDefinition {
        id: "forge-status",
        label: "FORGE Status",
        description: "Show local forge-core health through a fixed read-only wrapper.",
        executable: "foot",
        category: "FORGE",
        desktop_ids: &[],
        launch_args: &["-e", "forge-operator-core-status"],
    },
];

fn application_dirs() -> Vec<PathBuf> {
    vec![
        PathBuf::from("/run/current-system/sw/share/applications"),
        PathBuf::from("/usr/share/applications"),
        PathBuf::from("/usr/local/share/applications"),
    ]
}

fn icon_dirs() -> Vec<PathBuf> {
    vec![
        PathBuf::from("/run/current-system/sw/share/icons/hicolor"),
        PathBuf::from("/run/current-system/sw/share/icons/Adwaita"),
        PathBuf::from("/run/current-system/sw/share/pixmaps"),
        PathBuf::from("/usr/share/icons/hicolor"),
        PathBuf::from("/usr/share/icons/Adwaita"),
        PathBuf::from("/usr/share/pixmaps"),
    ]
}

fn parse_desktop_value(contents: &str, key: &str) -> Option<String> {
    let prefix = format!("{key}=");
    contents.lines().find_map(|line| {
        let trimmed = line.trim();
        trimmed
            .strip_prefix(&prefix)
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty())
    })
}

fn find_desktop_file(app: &OperatorAppDefinition) -> Option<(PathBuf, String)> {
    for dir in application_dirs() {
        for desktop_id in app.desktop_ids {
            let candidate = dir.join(desktop_id);
            if let Ok(contents) = std::fs::read_to_string(&candidate) {
                return Some((candidate, contents));
            }
        }
    }
    None
}

fn find_icon_path(icon_name: &str) -> Option<String> {
    let icon_path = Path::new(icon_name);
    if icon_path.is_absolute() && icon_path.exists() {
        return Some(icon_name.to_string());
    }
    let sizes = [
        "512x512/apps",
        "256x256/apps",
        "128x128/apps",
        "64x64/apps",
        "48x48/apps",
        "32x32/apps",
        "24x24/apps",
        "16x16/apps",
        "scalable/apps",
        "symbolic/apps",
    ];
    let extensions = ["png", "svg", "xpm"];
    for root in icon_dirs() {
        if root.ends_with("pixmaps") {
            for ext in extensions {
                let candidate = root.join(format!("{icon_name}.{ext}"));
                if candidate.exists() {
                    return Some(candidate.to_string_lossy().into_owned());
                }
            }
            continue;
        }
        for size in sizes {
            for ext in extensions {
                let candidate = root.join(size).join(format!("{icon_name}.{ext}"));
                if candidate.exists() {
                    return Some(candidate.to_string_lossy().into_owned());
                }
            }
        }
    }
    None
}

fn find_desktop_file_by_id(desktop_id: &str) -> Option<(PathBuf, String)> {
    for dir in application_dirs() {
        let candidate = dir.join(desktop_id);
        if let Ok(contents) = std::fs::read_to_string(&candidate) {
            return Some((candidate, contents));
        }
    }
    None
}

fn resolve_linux_window_icon(app_id: &str) -> (Option<String>, Option<String>) {
    let clean = app_id.trim();
    if clean.is_empty() {
        return (None, None);
    }
    for desktop_id in [
        format!("{clean}.desktop"),
        format!("{}.desktop", clean.to_ascii_lowercase()),
    ] {
        if let Some((_path, contents)) = find_desktop_file_by_id(&desktop_id) {
            if let Some(icon) = parse_desktop_value(&contents, "Icon") {
                return (Some(icon.clone()), find_icon_path(&icon));
            }
        }
    }
    (Some(clean.to_string()), find_icon_path(clean))
}

fn is_forge_shell_toplevel(app_id: &str, title: &str) -> bool {
    let app = app_id.trim().to_ascii_lowercase();
    let title = title.trim().to_ascii_lowercase();
    app == "forge_desktop"
        || app == "dev.forge.workshop"
        || app.contains("forge_desktop")
        || title == "forge"
        || title.starts_with("forge build")
}

fn linux_window_from_toplevel(toplevel: LswtToplevel) -> Option<LinuxWindowSnapshot> {
    let app_id = toplevel.app_id.trim().to_string();
    let title = toplevel.title.trim().to_string();
    let identifier = toplevel.identifier.trim().to_string();
    if identifier.is_empty() || (app_id.is_empty() && title.is_empty()) {
        return None;
    }
    if is_forge_shell_toplevel(&app_id, &title) {
        return None;
    }
    let (icon_name, icon_path) = resolve_linux_window_icon(&app_id);
    Some(LinuxWindowSnapshot {
        id: identifier,
        title: if title.is_empty() {
            app_id.clone()
        } else {
            title
        },
        app_id,
        icon_name,
        icon_path,
        focused: toplevel.activated.unwrap_or(false),
        minimized: toplevel.minimized.unwrap_or(false),
        native: true,
    })
}

fn parse_linux_window_snapshots(raw: &str) -> Result<Vec<LinuxWindowSnapshot>, String> {
    let output: LswtOutput =
        serde_json::from_str(raw).map_err(|err| format!("failed to parse lswt output: {err}"))?;
    Ok(output
        .toplevels
        .into_iter()
        .filter_map(linux_window_from_toplevel)
        .collect())
}

fn enrich_operator_app(app: &OperatorAppDefinition) -> OperatorApp {
    let mut enriched = OperatorApp {
        id: app.id.to_string(),
        label: app.label.to_string(),
        description: app.description.to_string(),
        executable: app.executable.to_string(),
        category: app.category.to_string(),
        icon_name: None,
        icon_path: None,
        desktop_file: None,
        native: false,
    };
    if let Some((path, contents)) = find_desktop_file(app) {
        enriched.desktop_file = Some(path.to_string_lossy().into_owned());
        enriched.native = true;
        if let Some(name) = parse_desktop_value(&contents, "Name") {
            enriched.label = name;
        }
        if let Some(comment) = parse_desktop_value(&contents, "Comment") {
            enriched.description = comment;
        }
        if let Some(icon) = parse_desktop_value(&contents, "Icon") {
            enriched.icon_path = find_icon_path(&icon);
            enriched.icon_name = Some(icon);
        }
    }
    enriched
}

fn operator_desktop_locked() -> bool {
    std::env::var("FORGE_OPERATOR_DESKTOP_LOCKED")
        .map(|value| matches!(value.as_str(), "1" | "true" | "TRUE" | "yes" | "YES"))
        .unwrap_or(false)
}

fn fit_operator_desktop_window(window: &tauri::WebviewWindow) {
    let monitor = window
        .current_monitor()
        .ok()
        .flatten()
        .or_else(|| window.available_monitors().ok()?.into_iter().next());

    let Some(monitor) = monitor else {
        return;
    };

    let position = monitor.position();
    let size = monitor.size();
    if size.width < 640 || size.height < 360 {
        return;
    }

    let _ = window.set_position(PhysicalPosition::new(position.x, position.y));
    let _ = window.set_size(PhysicalSize::new(size.width, size.height));
}

#[tauri::command]
fn read_system_diagnostics() -> Result<HostDiagnostics, String> {
    let mut system = System::new_all();
    system.refresh_all();

    let host_name = System::host_name().unwrap_or_else(|| "unknown".to_string());
    let os_name = System::name().unwrap_or_else(|| "unknown".to_string());
    let os_version = System::os_version().unwrap_or_else(|| "unknown".to_string());
    let kernel_version = System::kernel_version();
    let architecture = System::cpu_arch();
    let uptime_seconds = System::uptime();
    let cpu_count = system.cpus().len();

    let total_memory_bytes = system.total_memory();
    let available_memory_bytes = system.available_memory();
    let used_memory_bytes = total_memory_bytes.saturating_sub(available_memory_bytes);
    let total_swap_bytes = system.total_swap();
    let available_swap_bytes = system.free_swap();
    let used_swap_bytes = total_swap_bytes.saturating_sub(available_swap_bytes);

    let process = sysinfo::get_current_pid().ok().and_then(|pid: Pid| {
        system.process(pid).map(|proc| HostProcess {
            pid: pid.as_u32(),
            name: proc.name().to_string_lossy().into_owned(),
            memory_bytes: proc.memory(),
            virtual_memory_bytes: proc.virtual_memory(),
            cpu_usage_percent: proc.cpu_usage(),
            run_time_seconds: proc.run_time(),
        })
    });

    let mut disks = Vec::new();
    for disk in Disks::new_with_refreshed_list().list() {
        let total_bytes = disk.total_space();
        let available_bytes = disk.available_space();
        disks.push(HostDisk {
            name: disk.name().to_string_lossy().into_owned(),
            mount_point: disk.mount_point().to_string_lossy().into_owned(),
            file_system: disk.file_system().to_string_lossy().into_owned(),
            total_bytes,
            available_bytes,
            used_bytes: total_bytes.saturating_sub(available_bytes),
        });
    }

    Ok(HostDiagnostics {
        host_name,
        os_name,
        os_version,
        kernel_version,
        architecture,
        uptime_seconds,
        cpu_count,
        total_memory_bytes,
        available_memory_bytes,
        used_memory_bytes,
        total_swap_bytes,
        available_swap_bytes,
        used_swap_bytes,
        process,
        disks,
    })
}

#[tauri::command]
fn list_operator_apps() -> Vec<OperatorApp> {
    OPERATOR_APPS.iter().map(enrich_operator_app).collect()
}

fn resolve_operator_app(app_id: &str) -> Result<&'static OperatorAppDefinition, String> {
    OPERATOR_APPS
        .iter()
        .find(|candidate| candidate.id == app_id.trim())
        .ok_or_else(|| "operator app is not allowlisted".to_string())
}

#[tauri::command]
fn launch_operator_app(app_id: String) -> Result<OperatorAppLaunchResult, String> {
    let app = resolve_operator_app(&app_id)?;

    let child = std::process::Command::new(app.executable)
        .args(app.launch_args)
        .spawn()
        .map_err(|err| format!("failed to launch {}: {}", app.label, err))?;

    Ok(OperatorAppLaunchResult {
        app_id: app.id.to_string(),
        label: app.label.to_string(),
        executable: app.executable.to_string(),
        launched: true,
        pid: Some(child.id()),
        message: format!("{} launch requested", app.label),
    })
}

#[tauri::command]
fn list_linux_windows() -> Result<Vec<LinuxWindowSnapshot>, String> {
    if !cfg!(target_os = "linux") {
        return Ok(Vec::new());
    }
    let output = std::process::Command::new("lswt")
        .arg("--json")
        .output()
        .map_err(|err| format!("failed to run lswt: {err}"))?;
    if !output.status.success() {
        return Err(String::from_utf8_lossy(&output.stderr).trim().to_string());
    }
    let stdout = String::from_utf8_lossy(&output.stdout);
    parse_linux_window_snapshots(&stdout)
}

#[tauri::command]
fn focus_linux_window(window_id: String) -> Result<bool, String> {
    let target = list_linux_windows()?
        .into_iter()
        .find(|window| window.id == window_id)
        .ok_or_else(|| "linux window is no longer available".to_string())?;

    let mut command = std::process::Command::new("wlrctl");
    command.args(["toplevel", "focus"]);
    if !target.app_id.trim().is_empty() {
        command.arg(format!("app_id:{}", target.app_id));
    }
    if !target.title.trim().is_empty() {
        command.arg(format!("title:{}", target.title));
    }

    let output = command
        .output()
        .map_err(|err| format!("failed to run wlrctl: {err}"))?;
    if output.status.success() {
        Ok(true)
    } else {
        Err(String::from_utf8_lossy(&output.stderr).trim().to_string())
    }
}

fn main() {
    tauri::Builder::default()
        .on_window_event(|window, event| {
            if operator_desktop_locked() && window.label() == "main" {
                if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                    api.prevent_close();
                }
            }
        })
        .setup(|app| {
            if operator_desktop_locked() {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.set_decorations(false);
                    let _ = window.set_resizable(true);
                    fit_operator_desktop_window(&window);
                    let _ = window.maximize();
                    let _ = window.set_focus();
                }
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            read_system_diagnostics,
            list_operator_apps,
            launch_operator_app,
            list_linux_windows,
            focus_linux_window
        ])
        .run(tauri::generate_context!())
        .expect("error while running FORGE desktop");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn operator_apps_cover_toolbelt_categories() {
        let categories: std::collections::BTreeSet<&str> =
            OPERATOR_APPS.iter().map(|app| app.category).collect();
        for category in [
            "Workspace",
            "Internet",
            "AI Runtime",
            "System",
            "Developer",
            "FORGE",
        ] {
            assert!(
                categories.contains(category),
                "missing operator app category {category}"
            );
        }
    }

    #[test]
    fn operator_cli_apps_use_fixed_forge_wrappers() {
        for app_id in [
            "editor",
            "ollama-status",
            "modelruntime-status",
            "system-monitor",
            "lazygit",
            "core-logs",
            "network-diagnostics",
            "hardware-diagnostics",
            "forge-status",
        ] {
            let app = OPERATOR_APPS
                .iter()
                .find(|candidate| candidate.id == app_id)
                .unwrap_or_else(|| panic!("missing operator app {app_id}"));
            assert_eq!(app.executable, "foot");
            assert!(app.launch_args.contains(&"-e"));
            assert!(
                app.launch_args
                    .iter()
                    .any(|arg| arg.starts_with("forge-operator-")),
                "CLI launcher {app_id} must use a fixed forge-operator-* wrapper"
            );
        }
    }

    #[test]
    fn operator_wrapper_apps_cover_all_toolbelt_wrappers() {
        for wrapper in [
            "forge-operator-editor",
            "forge-operator-ollama-status",
            "forge-operator-models",
            "forge-operator-btop",
            "forge-operator-lazygit",
            "forge-operator-core-logs",
            "forge-operator-core-status",
            "forge-operator-network-diagnostics",
            "forge-operator-hardware-diagnostics",
        ] {
            assert!(
                OPERATOR_APPS
                    .iter()
                    .any(|app| app.executable == "foot" && app.launch_args == &["-e", wrapper]),
                "missing fixed operator launcher for {wrapper}"
            );
        }
    }

    #[test]
    fn operator_app_resolution_rejects_unknown_ids() {
        assert_eq!(
            resolve_operator_app("not-real")
                .err()
                .expect("unknown app id should fail"),
            "operator app is not allowlisted"
        );
        assert_eq!(
            resolve_operator_app("  api-health  ")
                .expect("trimmed app id should resolve")
                .id,
            "api-health"
        );
    }

    #[test]
    fn operator_launcher_does_not_expose_shell_injection_surface() {
        for app in OPERATOR_APPS {
            assert_ne!(app.executable, "sh");
            assert_ne!(app.executable, "bash");
            assert_ne!(app.executable, "systemctl");
            assert_ne!(app.executable, "nixos-rebuild");
            for arg in app.launch_args {
                assert!(!arg.contains("{}"), "placeholder arg found in {}", app.id);
                assert!(!arg.contains("$1"), "positional arg found in {}", app.id);
                assert!(!arg.contains(";"), "shell separator found in {}", app.id);
                assert!(!arg.contains("&&"), "shell separator found in {}", app.id);
                assert!(!arg.contains("|"), "pipe found in {}", app.id);
            }
        }
    }

    #[test]
    fn linux_window_snapshots_filter_shell_and_keep_native_apps() {
        let raw = r#"{
            "toplevels": [
                {
                    "identifier": "firefox-window",
                    "title": "Mozilla Firefox",
                    "app-id": "firefox"
                },
                {
                    "identifier": "forge-window",
                    "title": "FORGE Build",
                    "app-id": "forge_desktop"
                }
            ]
        }"#;

        let windows = parse_linux_window_snapshots(raw).expect("valid lswt json");

        assert_eq!(
            windows,
            vec![LinuxWindowSnapshot {
                id: "firefox-window".to_string(),
                title: "Mozilla Firefox".to_string(),
                app_id: "firefox".to_string(),
                icon_name: Some("firefox".to_string()),
                icon_path: find_icon_path("firefox"),
                focused: false,
                minimized: false,
                native: true,
            }]
        );
    }
}
