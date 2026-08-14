#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod desktop_metadata;
mod linux_windows;
mod notifications;
mod window_manager;

use desktop_metadata::{
    application_dirs, find_desktop_file_by_id, find_desktop_file_for_ids, find_icon_path,
    parse_desktop_value,
};

use serde::Serialize;
use std::collections::BTreeSet;
use std::path::{Path, PathBuf};
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

struct OperatorAppLaunchPlan {
    app_id: String,
    label: String,
    executable: String,
    args: Vec<String>,
}

#[derive(Serialize)]
struct HostPowerActionResult {
    action: String,
    requested: bool,
    message: String,
}

#[derive(Serialize)]
struct ShellSessionActionResult {
    action: String,
    requested: bool,
    message: String,
}

#[derive(Serialize)]
struct HostPowerPolicy {
    direct_system_control_enabled: bool,
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

fn desktop_entry_bool(contents: &str, key: &str) -> bool {
    parse_desktop_value(contents, key)
        .map(|value| matches!(value.as_str(), "true" | "True" | "TRUE" | "1"))
        .unwrap_or(false)
}

fn desktop_entry_is_visible_application(contents: &str) -> bool {
    if desktop_entry_bool(contents, "Hidden")
        || desktop_entry_bool(contents, "NoDisplay")
        || desktop_entry_bool(contents, "Terminal")
    {
        return false;
    }
    parse_desktop_value(contents, "Type")
        .map(|kind| kind == "Application")
        .unwrap_or(true)
}

fn desktop_category(contents: &str) -> String {
    let categories = parse_desktop_value(contents, "Categories").unwrap_or_default();
    if categories.contains("Development") {
        "Developer".to_string()
    } else if categories.contains("Network") {
        "Internet".to_string()
    } else if categories.contains("Settings")
        || categories.contains("System")
        || categories.contains("Utility")
    {
        "System".to_string()
    } else if categories.contains("AudioVideo") {
        "Media".to_string()
    } else if categories.contains("Graphics") {
        "Graphics".to_string()
    } else if categories.contains("Office") {
        "Office".to_string()
    } else if categories.contains("Game") {
        "Games".to_string()
    } else {
        "Native Apps".to_string()
    }
}

fn shell_meta_found(value: &str) -> bool {
    value.chars().any(|ch| {
        matches!(
            ch,
            ';' | '&' | '|' | '$' | '`' | '<' | '>' | '\\' | '\n' | '\r'
        )
    })
}

fn split_desktop_exec(exec: &str) -> Option<Vec<String>> {
    if shell_meta_found(exec) {
        return None;
    }

    let mut tokens = Vec::new();
    let mut current = String::new();
    let mut quote: Option<char> = None;

    for ch in exec.chars() {
        if let Some(active_quote) = quote {
            if ch == active_quote {
                quote = None;
            } else {
                current.push(ch);
            }
            continue;
        }

        if ch == '"' || ch == '\'' {
            quote = Some(ch);
        } else if ch.is_whitespace() {
            if !current.is_empty() {
                tokens.push(std::mem::take(&mut current));
            }
        } else {
            current.push(ch);
        }
    }

    if quote.is_some() {
        return None;
    }
    if !current.is_empty() {
        tokens.push(current);
    }

    if tokens.is_empty() {
        None
    } else {
        Some(tokens)
    }
}

fn blocked_desktop_executable(executable: &str) -> bool {
    let name = Path::new(executable)
        .file_name()
        .and_then(|value| value.to_str())
        .unwrap_or(executable);
    matches!(
        name,
        "sh" | "bash" | "dash" | "zsh" | "fish" | "pkexec" | "sudo" | "systemctl" | "nixos-rebuild"
    )
}

fn safe_desktop_exec_tokens(exec: &str) -> Option<Vec<String>> {
    let tokens = split_desktop_exec(exec.trim())?;
    let executable = tokens.first()?;
    if executable.contains('%') || blocked_desktop_executable(executable) {
        return None;
    }

    let filtered = tokens
        .into_iter()
        .filter(|token| !token.starts_with('%') && !token.contains('%'))
        .collect::<Vec<_>>();
    if filtered.is_empty() {
        None
    } else {
        Some(filtered)
    }
}

fn xdg_operator_app_from_desktop_entry(
    path: &Path,
    desktop_id: &str,
    contents: &str,
) -> Option<OperatorApp> {
    if !desktop_entry_is_visible_application(contents) {
        return None;
    }
    let name = parse_desktop_value(contents, "Name")?;
    let exec = parse_desktop_value(contents, "Exec")?;
    let tokens = safe_desktop_exec_tokens(&exec)?;
    let executable = tokens.first()?.clone();
    let description = parse_desktop_value(contents, "Comment")
        .or_else(|| parse_desktop_value(contents, "GenericName"))
        .unwrap_or_else(|| format!("Launch {name}."));
    let icon_name = parse_desktop_value(contents, "Icon");
    let icon_path = icon_name.as_deref().and_then(find_icon_path);

    Some(OperatorApp {
        id: format!("xdg:{desktop_id}"),
        label: name,
        description,
        executable,
        category: desktop_category(contents),
        icon_name,
        icon_path,
        desktop_file: Some(path.to_string_lossy().into_owned()),
        native: true,
    })
}

fn xdg_launch_plan_from_desktop_entry(
    app_id: &str,
    _path: &Path,
    contents: &str,
) -> Option<OperatorAppLaunchPlan> {
    if !desktop_entry_is_visible_application(contents) {
        return None;
    }
    let label = parse_desktop_value(contents, "Name")?;
    let exec = parse_desktop_value(contents, "Exec")?;
    let mut tokens = safe_desktop_exec_tokens(&exec)?;
    let executable = tokens.remove(0);

    Some(OperatorAppLaunchPlan {
        app_id: app_id.to_string(),
        label,
        executable,
        args: tokens,
    })
}

fn curated_desktop_ids() -> BTreeSet<&'static str> {
    OPERATOR_APPS
        .iter()
        .flat_map(|app| app.desktop_ids.iter().copied())
        .collect()
}

fn scan_operator_apps_from_dirs(dirs: &[PathBuf]) -> Vec<OperatorApp> {
    let mut apps = Vec::new();
    let mut seen = BTreeSet::new();

    for dir in dirs {
        let Ok(entries) = std::fs::read_dir(dir) else {
            continue;
        };
        let mut paths = entries
            .filter_map(Result::ok)
            .map(|entry| entry.path())
            .filter(|path| path.extension().and_then(|value| value.to_str()) == Some("desktop"))
            .collect::<Vec<_>>();
        paths.sort();

        for path in paths {
            let Some(desktop_id) = path.file_name().and_then(|value| value.to_str()) else {
                continue;
            };
            if !seen.insert(desktop_id.to_string()) {
                continue;
            }
            let Ok(contents) = std::fs::read_to_string(&path) else {
                continue;
            };
            if let Some(app) = xdg_operator_app_from_desktop_entry(&path, desktop_id, &contents) {
                apps.push(app);
            }
        }
    }

    apps
}

fn merge_operator_apps_with_scanned(scanned: Vec<OperatorApp>) -> Vec<OperatorApp> {
    let curated_desktop_ids = curated_desktop_ids();
    let mut apps = OPERATOR_APPS
        .iter()
        .map(enrich_operator_app)
        .collect::<Vec<_>>();
    let mut ids = apps
        .iter()
        .map(|app| app.id.clone())
        .collect::<BTreeSet<_>>();

    for app in scanned {
        let Some(desktop_id) = app.id.strip_prefix("xdg:") else {
            continue;
        };
        if curated_desktop_ids.contains(desktop_id) || !ids.insert(app.id.clone()) {
            continue;
        }
        apps.push(app);
    }

    apps
}

fn env_flag_enabled(value: Option<&str>) -> bool {
    value
        .map(str::trim)
        .map(|value| matches!(value, "1" | "true" | "TRUE" | "yes" | "YES"))
        .unwrap_or(false)
}

fn fullscreen_shell_policy(mode: Option<&str>, fullscreen: Option<&str>) -> bool {
    mode.map(str::trim) == Some("fullscreen-shell") && env_flag_enabled(fullscreen)
}

fn fullscreen_shell_enabled() -> bool {
    let mode = std::env::var("FORGE_SHELL_MODE").ok();
    let fullscreen = std::env::var("FORGE_SHELL_FULLSCREEN").ok();
    fullscreen_shell_policy(mode.as_deref(), fullscreen.as_deref())
}

fn operator_desktop_locked() -> bool {
    let locked = std::env::var("FORGE_OPERATOR_DESKTOP_LOCKED").ok();
    env_flag_enabled(locked.as_deref())
}

fn main_shell_window_locked() -> bool {
    fullscreen_shell_enabled() || operator_desktop_locked()
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

fn forge_api_token_file_from_env() -> Option<PathBuf> {
    if let Ok(value) = std::env::var("FORGE_API_TOKEN_FILE") {
        let trimmed = value.trim();
        if !trimmed.is_empty() {
            return Some(PathBuf::from(trimmed));
        }
    }
    None
}

fn forge_workspace_token_files_from_dir(cwd: &Path) -> Vec<PathBuf> {
    cwd.ancestors()
        .map(|dir| dir.join(".forge").join("docker-api-token"))
        .collect()
}

fn forge_api_token_files() -> Vec<PathBuf> {
    if let Some(path) = forge_api_token_file_from_env() {
        return vec![path];
    }

    let mut paths = Vec::new();
    if let Ok(cwd) = std::env::current_dir() {
        paths.extend(forge_workspace_token_files_from_dir(&cwd));
    }
    if let Some(path) = forge_data_dir().map(|dir| dir.join("auth").join("api_token")) {
        paths.push(path);
    }
    paths
}

#[tauri::command]
fn read_forge_api_token() -> Result<Option<String>, String> {
    if let Ok(value) = std::env::var("FORGE_API_TOKEN") {
        let token = value.trim();
        if !token.is_empty() {
            return Ok(Some(token.to_string()));
        }
    }
    for path in forge_api_token_files() {
        match std::fs::read_to_string(&path) {
            Ok(body) => {
                let token = body.trim();
                if !token.is_empty() {
                    return Ok(Some(token.to_string()));
                }
            }
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => continue,
            Err(err) => {
                return Err(format!(
                    "failed to read FORGE API token from {}: {}",
                    path.display(),
                    err
                ))
            }
        }
    }
    Ok(None)
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
    merge_operator_apps_with_scanned(scan_operator_apps_from_dirs(&application_dirs()))
}

fn resolve_operator_app(app_id: &str) -> Result<&'static OperatorAppDefinition, String> {
    OPERATOR_APPS
        .iter()
        .find(|candidate| candidate.id == app_id.trim())
        .ok_or_else(|| "operator app is not allowlisted".to_string())
}

#[tauri::command]
fn launch_operator_app(app_id: String) -> Result<OperatorAppLaunchResult, String> {
    let app_id = app_id.trim();
    let launch = if let Some(desktop_id) = app_id.strip_prefix("xdg:") {
        let (path, contents) = find_desktop_file_by_id(desktop_id)
            .ok_or_else(|| "operator app is not allowlisted".to_string())?;
        xdg_launch_plan_from_desktop_entry(app_id, &path, &contents)
            .ok_or_else(|| "operator app is not allowlisted".to_string())?
    } else {
        let app = resolve_operator_app(app_id)?;
        OperatorAppLaunchPlan {
            app_id: app.id.to_string(),
            label: app.label.to_string(),
            executable: app.executable.to_string(),
            args: app.launch_args.iter().map(|arg| arg.to_string()).collect(),
        }
    };

    let child = std::process::Command::new(&launch.executable)
        .args(&launch.args)
        .spawn()
        .map_err(|err| format!("failed to launch {}: {}", launch.label, err))?;

    Ok(OperatorAppLaunchResult {
        app_id: launch.app_id,
        label: launch.label.clone(),
        executable: launch.executable,
        launched: true,
        pid: Some(child.id()),
        message: format!("{} launch requested", launch.label),
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

fn read_host_power_policy_state(direct_system_control_enabled: bool) -> HostPowerPolicy {
    HostPowerPolicy {
        direct_system_control_enabled,
        message: if direct_system_control_enabled {
            "Host shutdown and reboot controls are enabled for this shell session".to_string()
        } else {
            "Host shutdown and reboot controls are disabled by FORGE_SHELL_DIRECT_SYSTEM_CONTROL policy"
                .to_string()
        },
    }
}

#[tauri::command]
fn read_host_power_policy() -> HostPowerPolicy {
    read_host_power_policy_state(direct_system_control_enabled())
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

fn shell_session_enabled() -> bool {
    std::env::var("FORGE_SHELL_SESSION_ENABLED")
        .map(|value| matches!(value.trim(), "1" | "true" | "TRUE" | "yes" | "YES"))
        .unwrap_or(false)
}

fn spawn_shell_restart_command() -> Result<(), String> {
    let executable = std::env::current_exe()
        .map_err(|err| format!("failed to resolve current shell executable: {err}"))?;
    let args = std::env::args_os().skip(1).collect::<Vec<_>>();
    Command::new(&executable)
        .args(args)
        .spawn()
        .map_err(|err| format!("failed to restart FORGE shell: {err}"))?;

    std::thread::spawn(|| {
        std::thread::sleep(std::time::Duration::from_millis(150));
        std::process::exit(0);
    });
    Ok(())
}

fn request_shell_session_action_with_policy<F>(
    action: String,
    shell_session_enabled: bool,
    mut runner: F,
) -> Result<ShellSessionActionResult, String>
where
    F: FnMut(&str) -> Result<(), String>,
{
    let normalized = action.trim().to_ascii_lowercase();
    if normalized != "restart_shell" {
        return Err("shell session action is not allowlisted".to_string());
    }

    if !shell_session_enabled {
        return Ok(ShellSessionActionResult {
            action: normalized,
            requested: false,
            message: "FORGE shell restart is available only inside forge-shell-session".to_string(),
        });
    }

    runner(&normalized)?;
    Ok(ShellSessionActionResult {
        action: normalized,
        requested: true,
        message: "FORGE shell restart requested".to_string(),
    })
}

#[tauri::command]
fn request_shell_session_action(action: String) -> Result<ShellSessionActionResult, String> {
    request_shell_session_action_with_policy(action, shell_session_enabled(), |_| {
        spawn_shell_restart_command()
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
            if main_shell_window_locked() && window.label() == "main" {
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
            if let Some(window) = app.get_webview_window("main") {
                if fullscreen_shell_enabled() {
                    eprintln!("[forge-shell] enforcing fullscreen main surface");
                    window.set_decorations(false)?;
                    window.set_resizable(false)?;
                    window.set_fullscreen(true)?;
                    window.set_focus()?;
                } else if operator_desktop_locked() {
                    window.set_decorations(false)?;
                    window.set_resizable(false)?;
                    fit_operator_desktop_window(&window);
                    window.set_focus()?;
                }
            }
            notifications::start_freedesktop_service(app.handle().clone());
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            read_forge_api_token,
            read_system_diagnostics,
            list_operator_apps,
            launch_operator_app,
            read_host_power_policy,
            request_host_power_action,
            request_shell_session_action,
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
    fn fullscreen_shell_policy_requires_mode_and_explicit_flag() {
        for enabled in ["1", "true", "TRUE", "yes", "YES", " true "] {
            assert!(fullscreen_shell_policy(
                Some("fullscreen-shell"),
                Some(enabled)
            ));
        }
        for disabled in [None, Some(""), Some("0"), Some("false"), Some("no")] {
            assert!(!fullscreen_shell_policy(Some("fullscreen-shell"), disabled));
        }
        assert!(!fullscreen_shell_policy(
            Some("operator-desktop"),
            Some("true")
        ));
        assert!(!fullscreen_shell_policy(None, Some("true")));
    }

    #[test]
    fn forge_workspace_token_lookup_walks_up_from_desktop_package() {
        let cwd = PathBuf::from("/repo/apps/desktop");
        let paths = forge_workspace_token_files_from_dir(&cwd);

        assert_eq!(
            paths,
            vec![
                PathBuf::from("/repo/apps/desktop/.forge/docker-api-token"),
                PathBuf::from("/repo/apps/.forge/docker-api-token"),
                PathBuf::from("/repo/.forge/docker-api-token"),
                PathBuf::from("/.forge/docker-api-token"),
            ]
        );
    }

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
    fn xdg_desktop_entries_become_native_operator_apps() {
        let path = PathBuf::from("/usr/share/applications/org.example.Tool.desktop");
        let app = xdg_operator_app_from_desktop_entry(
            &path,
            "org.example.Tool.desktop",
            "[Desktop Entry]\nType=Application\nName=Example Tool\nComment=Inspect example data\nExec=example-tool --new-window %U\nIcon=example-tool\nCategories=Development;Utility;\n",
        )
        .expect("safe application desktop entry should be listed");

        assert_eq!(app.id, "xdg:org.example.Tool.desktop");
        assert_eq!(app.label, "Example Tool");
        assert_eq!(app.description, "Inspect example data");
        assert_eq!(app.executable, "example-tool");
        assert_eq!(app.category, "Developer");
        assert_eq!(app.icon_name.as_deref(), Some("example-tool"));
        assert_eq!(app.desktop_file.as_deref(), Some(path.to_str().unwrap()));
        assert!(app.native);

        let launch = xdg_launch_plan_from_desktop_entry(
            "xdg:org.example.Tool.desktop",
            &path,
            "[Desktop Entry]\nType=Application\nName=Example Tool\nExec=example-tool --new-window %U\n",
        )
        .expect("safe application desktop entry should be launchable");
        assert_eq!(launch.app_id, "xdg:org.example.Tool.desktop");
        assert_eq!(launch.label, "Example Tool");
        assert_eq!(launch.executable, "example-tool");
        assert_eq!(launch.args, vec!["--new-window"]);
    }

    #[test]
    fn xdg_desktop_exec_rejects_shell_and_host_mutation_paths() {
        for exec in [
            "sh -c 'touch /tmp/nope'",
            "bash -lc firefox",
            "pkexec pcmanfm",
            "sudo reboot",
            "systemctl reboot",
            "nixos-rebuild switch",
            "firefox; touch /tmp/nope",
            "firefox && touch /tmp/nope",
            "firefox | tee /tmp/nope",
            "firefox $HOME",
            "firefox `whoami`",
        ] {
            assert!(
                safe_desktop_exec_tokens(exec).is_none(),
                "unsafe Exec line should be rejected: {exec}"
            );
        }
    }

    #[test]
    fn scanned_operator_apps_append_after_curated_apps_and_skip_curated_desktops() {
        let native = OperatorApp {
            id: "xdg:org.example.Tool.desktop".to_string(),
            label: "Example Tool".to_string(),
            description: "Inspect example data".to_string(),
            executable: "example-tool".to_string(),
            category: "Developer".to_string(),
            icon_name: None,
            icon_path: None,
            desktop_file: Some("/usr/share/applications/org.example.Tool.desktop".to_string()),
            native: true,
        };
        let duplicate_terminal = OperatorApp {
            id: "xdg:foot.desktop".to_string(),
            label: "Foot".to_string(),
            description: "Terminal".to_string(),
            executable: "foot".to_string(),
            category: "System".to_string(),
            icon_name: None,
            icon_path: None,
            desktop_file: Some("/usr/share/applications/foot.desktop".to_string()),
            native: true,
        };

        let apps = merge_operator_apps_with_scanned(vec![duplicate_terminal, native]);

        assert_eq!(
            apps.first().expect("curated apps should exist").id,
            "terminal"
        );
        assert!(apps
            .iter()
            .any(|app| app.id == "xdg:org.example.Tool.desktop"));
        assert!(!apps.iter().any(|app| app.id == "xdg:foot.desktop"));
        assert!(
            apps.iter()
                .position(|app| app.id == "xdg:org.example.Tool.desktop")
                .expect("scanned app should be present")
                > OPERATOR_APPS.len() - 1,
            "scanned apps should append after curated FORGE apps"
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
        let policy = read_host_power_policy_state(false);
        assert!(!policy.direct_system_control_enabled);
        assert!(policy.message.contains("disabled"));

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
        let policy = read_host_power_policy_state(true);
        assert!(policy.direct_system_control_enabled);
        assert!(policy.message.contains("enabled"));

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

    #[test]
    fn shell_session_restart_is_declined_outside_forge_shell_session() {
        let mut called = false;
        let result =
            request_shell_session_action_with_policy("restart_shell".to_string(), false, |_| {
                called = true;
                Ok(())
            })
            .expect("disabled shell session policy should decline cleanly");

        assert_eq!(result.action, "restart_shell");
        assert!(!result.requested);
        assert!(!called);
        assert!(result.message.contains("forge-shell-session"));
    }

    #[test]
    fn shell_session_restart_uses_runner_only_inside_forge_shell_session() {
        let mut requested = String::new();
        let result = request_shell_session_action_with_policy(
            " restart_shell ".to_string(),
            true,
            |action| {
                requested = action.to_string();
                Ok(())
            },
        )
        .expect("enabled shell session policy should call the runner");

        assert_eq!(requested, "restart_shell");
        assert_eq!(result.action, "restart_shell");
        assert!(result.requested);
        assert!(result.message.contains("restart"));
    }

    #[test]
    fn shell_session_action_rejects_unknown_actions() {
        let result = request_shell_session_action_with_policy(
            "systemctl restart forge-core".to_string(),
            true,
            |_| Ok(()),
        );

        assert_eq!(
            result.err().expect("unknown action should fail"),
            "shell session action is not allowlisted"
        );
    }
}
