#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::Serialize;
use std::path::{Path, PathBuf};
use sysinfo::{Disks, Pid, System};
use tauri::Manager;

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
        id: "browser",
        label: "Browser",
        description: "Open Firefox for local docs, web consoles, and model tooling.",
        executable: "firefox",
        category: "Internet",
        desktop_ids: &["firefox.desktop"],
        launch_args: &[],
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

#[tauri::command]
fn launch_operator_app(app_id: String) -> Result<OperatorAppLaunchResult, String> {
    let app = OPERATOR_APPS
        .iter()
        .find(|candidate| candidate.id == app_id.trim())
        .ok_or_else(|| "operator app is not allowlisted".to_string())?;

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
                    let _ = window.maximize();
                    let _ = window.set_resizable(false);
                }
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            read_system_diagnostics,
            list_operator_apps,
            launch_operator_app
        ])
        .run(tauri::generate_context!())
        .expect("error while running FORGE desktop");
}
