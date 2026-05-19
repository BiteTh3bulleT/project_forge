#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod desktop_metadata;
mod linux_windows;
mod window_manager;

use desktop_metadata::{find_desktop_file_for_ids, find_icon_path, parse_desktop_value};

use serde::Serialize;
use std::path::PathBuf;
use std::process::Command;
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

#[derive(Serialize)]
struct HostPowerActionResult {
    action: String,
    requested: bool,
    message: String,
}

const OPERATOR_APPS: &[OperatorAppDefinition] = &[
    OperatorAppDefinition {
        id: "terminal",
        label: "Terminal",
        description: "Open a Foot terminal in the current FORGE operator session.",
        executable: "foot",
        category: "Workspace",
        desktop_ids: &["foot.desktop"],
        launch_args: &["--working-directory=/forge/workspaces/default"],
    },
    OperatorAppDefinition {
        id: "files",
        label: "Files",
        description: "Open the PCManFM file manager in the current FORGE operator session.",
        executable: "pcmanfm",
        category: "Workspace",
        desktop_ids: &["pcmanfm.desktop"],
        launch_args: &["/forge/workspaces/default"],
    },
    OperatorAppDefinition {
        id: "editor",
        label: "Editor",
        description: "Open the native Mousepad text editor.",
        executable: "mousepad",
        category: "Workspace",
        desktop_ids: &["org.xfce.mousepad.desktop", "mousepad.desktop"],
        launch_args: &[],
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

fn find_desktop_file(app: &OperatorAppDefinition) -> Option<(PathBuf, String)> {
    find_desktop_file_for_ids(app.desktop_ids)
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

fn forge_data_dir() -> Option<PathBuf> {
    if let Ok(value) = std::env::var("FORGE_DATA_DIR") {
        let trimmed = value.trim();
        if !trimmed.is_empty() {
            return Some(PathBuf::from(trimmed));
        }
    }
    if cfg!(windows) {
        std::env::var("APPDATA")
            .ok()
            .filter(|value| !value.trim().is_empty())
            .map(|value| PathBuf::from(value).join("forge"))
    } else {
        std::env::var("XDG_CONFIG_HOME")
            .ok()
            .filter(|value| !value.trim().is_empty())
            .map(PathBuf::from)
            .or_else(|| {
                std::env::var("HOME")
                    .ok()
                    .filter(|value| !value.trim().is_empty())
                    .map(|value| PathBuf::from(value).join(".config"))
            })
            .map(|base| base.join("forge"))
    }
}

fn forge_api_token_file() -> Option<PathBuf> {
    if let Ok(value) = std::env::var("FORGE_API_TOKEN_FILE") {
        let trimmed = value.trim();
        if !trimmed.is_empty() {
            return Some(PathBuf::from(trimmed));
        }
    }
    forge_data_dir().map(|dir| dir.join("auth").join("api_token"))
}

#[tauri::command]
fn read_forge_api_token() -> Result<Option<String>, String> {
    if let Ok(value) = std::env::var("FORGE_API_TOKEN") {
        let token = value.trim();
        if !token.is_empty() {
            return Ok(Some(token.to_string()));
        }
    }
    let Some(path) = forge_api_token_file() else {
        return Ok(None);
    };
    match std::fs::read_to_string(&path) {
        Ok(body) => {
            let token = body.trim();
            if token.is_empty() {
                Ok(None)
            } else {
                Ok(Some(token.to_string()))
            }
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(err) => Err(format!("failed to read FORGE API token: {}", err)),
    }
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

fn spawn_host_power_command(action: &str) -> Result<(), String> {
    let mut command = match action {
        "shutdown" => {
            #[cfg(target_os = "windows")]
            {
                let mut command = Command::new("shutdown");
                command.args(["/s", "/t", "0"]);
                command
            }
            #[cfg(target_os = "macos")]
            {
                let mut command = Command::new("osascript");
                command.args(["-e", "tell app \"System Events\" to shut down"]);
                command
            }
            #[cfg(all(unix, not(target_os = "macos")))]
            {
                let mut command = Command::new("systemctl");
                command.arg("poweroff");
                command
            }
        }
        "reboot" => {
            #[cfg(target_os = "windows")]
            {
                let mut command = Command::new("shutdown");
                command.args(["/r", "/t", "0"]);
                command
            }
            #[cfg(target_os = "macos")]
            {
                let mut command = Command::new("osascript");
                command.args(["-e", "tell app \"System Events\" to restart"]);
                command
            }
            #[cfg(all(unix, not(target_os = "macos")))]
            {
                let mut command = Command::new("systemctl");
                command.arg("reboot");
                command
            }
        }
        _ => return Err("host power action is not allowlisted".to_string()),
    };

    command
        .spawn()
        .map(|_| ())
        .map_err(|err| format!("failed to request {action}: {err}"))
}

fn direct_system_control_enabled() -> bool {
    std::env::var("FORGE_SHELL_DIRECT_SYSTEM_CONTROL")
        .map(|value| matches!(value.trim(), "1" | "true" | "TRUE" | "yes" | "YES"))
        .unwrap_or(false)
}

fn request_host_power_action_with_policy<F>(
    action: String,
    direct_control_enabled: bool,
    mut runner: F,
) -> Result<HostPowerActionResult, String>
where
    F: FnMut(&str) -> Result<(), String>,
{
    let normalized = action.trim().to_ascii_lowercase();
    if normalized != "shutdown" && normalized != "reboot" {
        return Err("host power action is not allowlisted".to_string());
    }

    if !direct_control_enabled {
        return Ok(HostPowerActionResult {
            action: normalized,
            requested: false,
            message: "Host power controls are disabled by FORGE_SHELL_DIRECT_SYSTEM_CONTROL policy"
                .to_string(),
        });
    }

    runner(&normalized)?;
    Ok(HostPowerActionResult {
        action: normalized.clone(),
        requested: true,
        message: format!("Host {normalized} requested"),
    })
}

#[tauri::command]
fn request_host_power_action(action: String) -> Result<HostPowerActionResult, String> {
    request_host_power_action_with_policy(action, direct_system_control_enabled(), |action| {
        spawn_host_power_command(action)
    })
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_os::init())
        .manage(linux_windows::LinuxWindowRegistryState::default())
        .manage(window_manager::WindowManagerState::default())
        .on_window_event(|window, event| {
            if operator_desktop_locked() && window.label() == "main" {
                if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                    api.prevent_close();
                }
            }
            let app = window.app_handle();
            let state = app.state::<window_manager::WindowManagerState>();
            window_manager::handle_window_event(&app, state.inner(), window.label(), event);
        })
        .setup(|app| {
            let state = app.state::<window_manager::WindowManagerState>();
            window_manager::register_main_window(app.handle(), state.inner())
                .map_err(|err| Box::<dyn std::error::Error>::from(err))?;
            if operator_desktop_locked() {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.set_decorations(false);
                    let _ = window.set_resizable(true);
                    fit_operator_desktop_window(&window);
                    let _ = window.set_focus();
                }
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            read_forge_api_token,
            read_system_diagnostics,
            list_operator_apps,
            launch_operator_app,
            request_host_power_action,
            linux_windows::list_linux_windows,
            linux_windows::focus_linux_window,
            linux_windows::control_linux_window,
            window_manager::forge_window_open,
            window_manager::forge_window_close,
            window_manager::forge_window_focus,
            window_manager::forge_window_show,
            window_manager::forge_window_hide,
            window_manager::forge_window_toggle,
            window_manager::forge_window_minimize,
            window_manager::forge_window_list,
            window_manager::forge_window_snapshot,
            window_manager::forge_window_restore_layout,
            window_manager::forge_window_sync_state
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
    fn operator_editor_uses_native_gui_editor() {
        let app = OPERATOR_APPS
            .iter()
            .find(|candidate| candidate.id == "editor")
            .expect("editor launcher should exist");
        assert_eq!(app.executable, "mousepad");
        assert!(app.desktop_ids.contains(&"org.xfce.mousepad.desktop"));
        assert!(app.launch_args.is_empty());
    }

    #[test]
    fn operator_wrapper_apps_cover_all_toolbelt_wrappers() {
        for wrapper in [
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
    fn operator_workspace_launchers_use_forge_workspace_path() {
        let terminal = OPERATOR_APPS
            .iter()
            .find(|candidate| candidate.id == "terminal")
            .expect("terminal launcher should exist");
        assert_eq!(
            terminal.launch_args,
            &["--working-directory=/forge/workspaces/default"]
        );

        let files = OPERATOR_APPS
            .iter()
            .find(|candidate| candidate.id == "files")
            .expect("files launcher should exist");
        assert_eq!(files.launch_args, &["/forge/workspaces/default"]);

        for app in OPERATOR_APPS {
            assert!(
                app.launch_args
                    .iter()
                    .all(|arg| !arg.contains("/projectforge")),
                "{} still depends on missing /projectforge path",
                app.id
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
    fn host_power_action_is_policy_disabled_by_default() {
        let mut called = false;
        let result = request_host_power_action_with_policy("reboot".to_string(), false, |_| {
            called = true;
            Ok(())
        })
        .expect("disabled policy should return a successful declined result");

        assert_eq!(result.action, "reboot");
        assert!(!result.requested);
        assert!(!called);
        assert!(result.message.contains("disabled"));
    }

    #[test]
    fn host_power_action_uses_runner_only_when_policy_enabled() {
        let mut requested = String::new();
        let result =
            request_host_power_action_with_policy(" shutdown ".to_string(), true, |action| {
                requested = action.to_string();
                Ok(())
            })
            .expect("enabled policy should call the runner");

        assert_eq!(requested, "shutdown");
        assert_eq!(result.action, "shutdown");
        assert!(result.requested);
    }
}
