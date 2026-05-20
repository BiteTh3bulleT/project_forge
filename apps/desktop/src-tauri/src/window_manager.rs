use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};
use tauri::{
    AppHandle, Emitter, LogicalPosition, LogicalSize, Manager, WebviewUrl, WebviewWindow,
    WebviewWindowBuilder, WindowEvent,
};

const REGISTRY_UPDATED_EVENT: &str = "forge://window/registry-updated";
const OPENED_EVENT: &str = "forge://window/opened";
const CLOSED_EVENT: &str = "forge://window/closed";
const FOCUSED_EVENT: &str = "forge://window/focused";
const HIDDEN_EVENT: &str = "forge://window/hidden";
const SHOWN_EVENT: &str = "forge://window/shown";
const LAYOUT_RESTORED_EVENT: &str = "forge://layout/restored";
const LAYOUT_FILE: &str = "forge-window-layout.json";

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum WindowKind {
    MainShell,
    Workspace,
    Terminal,
    MemoryPanel,
    TaskPanel,
    SystemPanel,
    Settings,
    Inspector,
    ArtifactViewer,
    DebugConsole,
    ShellHost,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "camelCase")]
pub struct WindowBounds {
    pub x: f64,
    pub y: f64,
    pub width: f64,
    pub height: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WindowDescriptor {
    pub kind: WindowKind,
    pub label: String,
    pub route: String,
    pub title: String,
    pub singleton: bool,
    pub bounds: WindowBounds,
    pub workspace_id: Option<String>,
    pub artifact_id: Option<String>,
    pub session_id: Option<String>,
    pub host_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WindowOpenRequest {
    pub kind: WindowKind,
    pub workspace_id: Option<String>,
    pub artifact_id: Option<String>,
    pub session_id: Option<String>,
    pub host_id: Option<String>,
    pub route: Option<String>,
    pub title: Option<String>,
    pub bounds: Option<WindowBounds>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "camelCase")]
pub struct WindowRegistryEntry {
    pub label: String,
    pub kind: WindowKind,
    pub route: String,
    pub title: String,
    pub visible: bool,
    pub focused: bool,
    pub minimized: bool,
    pub singleton: bool,
    pub bounds: Option<WindowBounds>,
    pub workspace_id: Option<String>,
    pub artifact_id: Option<String>,
    pub session_id: Option<String>,
    pub host_id: Option<String>,
    pub created_at_ms: u64,
    pub updated_at_ms: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct WindowRestoreFailure {
    pub label: Option<String>,
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default, PartialEq)]
#[serde(rename_all = "camelCase")]
pub struct WindowRegistrySnapshot {
    pub windows: Vec<WindowRegistryEntry>,
    pub timestamp_ms: u64,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub restore_failures: Vec<WindowRestoreFailure>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct WindowEventPayload {
    label: String,
    kind: WindowKind,
    route: String,
    visible: bool,
    focused: bool,
    timestamp_ms: u64,
    workspace_id: Option<String>,
    artifact_id: Option<String>,
    host_id: Option<String>,
}

#[derive(Default)]
pub struct WindowManagerState {
    registry: Mutex<WindowRegistry>,
}

#[derive(Default)]
struct WindowRegistry {
    entries: BTreeMap<String, WindowRegistryEntry>,
}

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_millis().min(u128::from(u64::MAX)) as u64)
        .unwrap_or(0)
}

fn default_bounds(kind: WindowKind) -> WindowBounds {
    match kind {
        WindowKind::Settings => WindowBounds {
            x: 180.0,
            y: 120.0,
            width: 980.0,
            height: 680.0,
        },
        WindowKind::DebugConsole => WindowBounds {
            x: 220.0,
            y: 140.0,
            width: 1120.0,
            height: 720.0,
        },
        WindowKind::ArtifactViewer => WindowBounds {
            x: 160.0,
            y: 110.0,
            width: 1180.0,
            height: 760.0,
        },
        WindowKind::Terminal => WindowBounds {
            x: 140.0,
            y: 100.0,
            width: 1100.0,
            height: 720.0,
        },
        _ => WindowBounds {
            x: 120.0,
            y: 92.0,
            width: 1040.0,
            height: 680.0,
        },
    }
}

fn normalize_bounds(bounds: Option<WindowBounds>, kind: WindowKind) -> WindowBounds {
    let bounds = bounds.unwrap_or_else(|| default_bounds(kind));
    let fallback = default_bounds(kind);
    if !bounds.x.is_finite()
        || !bounds.y.is_finite()
        || !bounds.width.is_finite()
        || !bounds.height.is_finite()
        || bounds.width < 320.0
        || bounds.height < 240.0
        || bounds.width > 8000.0
        || bounds.height > 8000.0
    {
        return fallback;
    }
    WindowBounds {
        x: bounds.x,
        y: bounds.y,
        width: bounds.width,
        height: bounds.height,
    }
}

fn singleton_kind(kind: WindowKind) -> bool {
    matches!(
        kind,
        WindowKind::MainShell
            | WindowKind::MemoryPanel
            | WindowKind::TaskPanel
            | WindowKind::SystemPanel
            | WindowKind::Settings
            | WindowKind::Inspector
            | WindowKind::DebugConsole
    )
}

#[cfg(not(test))]
fn debug_console_enabled() -> bool {
    std::env::var("FORGE_SHELL_DEBUG_CONSOLE")
        .map(|value| {
            matches!(
                value.trim().to_ascii_lowercase().as_str(),
                "1" | "true" | "yes" | "on"
            )
        })
        .unwrap_or(false)
}

#[cfg(test)]
fn debug_console_enabled() -> bool {
    false
}

fn clean_dynamic_id(value: &str) -> Result<String, String> {
    let clean = value.trim();
    if clean.is_empty() || clean == "." || clean == ".." || clean.contains("..") {
        return Err("dynamic window id is invalid".to_string());
    }
    if clean.len() > 64 {
        return Err("dynamic window id is too long".to_string());
    }
    if !clean
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_' | '.'))
    {
        return Err("dynamic window id contains unsupported characters".to_string());
    }
    Ok(clean.to_string())
}

fn shell_host_label_from_id(host_id: &str) -> Result<String, String> {
    let clean = clean_dynamic_id(host_id)?;
    let label = clean
        .strip_prefix("forge-monitor-")
        .map(|_| clean.clone())
        .unwrap_or_else(|| format!("forge-monitor-{clean}"));
    if label == "main" || label == "forge-monitor-main" {
        return Err("shell host id is reserved".to_string());
    }
    validate_window_label(&label)?;
    Ok(label)
}

pub fn validate_window_label(label: &str) -> Result<(), String> {
    let clean = label.trim();
    if clean.is_empty() || clean.len() > 96 || clean != label {
        return Err("window label is invalid".to_string());
    }
    if clean.contains("..") {
        return Err("window label must not contain traversal segments".to_string());
    }
    if !clean
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_' | '.'))
    {
        return Err("window label contains unsupported characters".to_string());
    }
    Ok(())
}

fn normalize_route(route: Option<String>, fallback: String) -> Result<String, String> {
    let route = route.unwrap_or(fallback);
    let clean = route.trim();
    if clean.is_empty()
        || !clean.starts_with('/')
        || clean.contains("://")
        || clean.contains("..")
        || clean.chars().any(|ch| ch.is_control())
        || clean.len() > 512
    {
        return Err("window route is invalid".to_string());
    }
    Ok(clean.to_string())
}

fn safe_title(title: Option<String>, fallback: &str) -> String {
    let clean = title
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty() && value.len() <= 120)
        .unwrap_or(fallback);
    clean
        .chars()
        .filter(|ch| !ch.is_control())
        .collect::<String>()
}

fn descriptor_from_request(request: WindowOpenRequest) -> Result<WindowDescriptor, String> {
    let singleton = singleton_kind(request.kind);
    let (label, fallback_route, fallback_title) = match request.kind {
        WindowKind::MainShell => ("main".to_string(), "/".to_string(), "FORGE".to_string()),
        WindowKind::Workspace => {
            let workspace_id = request
                .workspace_id
                .as_deref()
                .map(clean_dynamic_id)
                .transpose()?;
            let label = workspace_id
                .as_deref()
                .map(|id| format!("workspace-{id}"))
                .unwrap_or_else(|| "workspace-main".to_string());
            let route = workspace_id
                .as_deref()
                .map(|id| format!("/workspaces/{id}"))
                .unwrap_or_else(|| "/".to_string());
            (label, route, "FORGE Workspace".to_string())
        }
        WindowKind::Terminal => {
            let session_id = request
                .session_id
                .as_deref()
                .map(clean_dynamic_id)
                .transpose()?;
            let label = session_id
                .as_deref()
                .map(|id| format!("terminal-{id}"))
                .unwrap_or_else(|| "terminal-panel".to_string());
            (
                label,
                "/operator-apps".to_string(),
                "FORGE Terminal".to_string(),
            )
        }
        WindowKind::ArtifactViewer => {
            let artifact_id = request
                .artifact_id
                .as_deref()
                .ok_or_else(|| "artifact viewer requires artifactId".to_string())
                .and_then(clean_dynamic_id)?;
            (
                format!("artifact-{artifact_id}"),
                format!("/artifacts/{artifact_id}"),
                "FORGE Artifact".to_string(),
            )
        }
        WindowKind::Settings => (
            "settings".to_string(),
            "/settings".to_string(),
            "FORGE Settings".to_string(),
        ),
        WindowKind::MemoryPanel => (
            "memory-panel".to_string(),
            "/memory".to_string(),
            "FORGE Memory".to_string(),
        ),
        WindowKind::TaskPanel => (
            "task-panel".to_string(),
            "/jobs".to_string(),
            "FORGE Tasks".to_string(),
        ),
        WindowKind::SystemPanel => (
            "system-panel".to_string(),
            "/system".to_string(),
            "FORGE System".to_string(),
        ),
        WindowKind::Inspector => (
            "inspector".to_string(),
            "/inspectors".to_string(),
            "FORGE Inspector".to_string(),
        ),
        WindowKind::DebugConsole => {
            if !debug_console_enabled() {
                return Err("debug console window is disabled by default".to_string());
            }
            (
                "debug-console".to_string(),
                "/system?surface=debug-console".to_string(),
                "FORGE Debug Console".to_string(),
            )
        }
        WindowKind::ShellHost => {
            let host_id = request
                .host_id
                .as_deref()
                .ok_or_else(|| "shell host requires hostId".to_string())?;
            let label = shell_host_label_from_id(host_id)?;
            let title_suffix = label
                .strip_prefix("forge-monitor-")
                .filter(|value| !value.is_empty())
                .unwrap_or(&label);
            (
                label.clone(),
                format!("/?host={label}"),
                format!("FORGE Monitor {title_suffix}"),
            )
        }
    };

    validate_window_label(&label)?;
    let route = normalize_route(request.route, fallback_route)?;
    let title = safe_title(request.title, &fallback_title);
    Ok(WindowDescriptor {
        kind: request.kind,
        label,
        route,
        title,
        singleton,
        bounds: normalize_bounds(request.bounds, request.kind),
        workspace_id: request.workspace_id,
        artifact_id: request.artifact_id,
        session_id: request.session_id,
        host_id: request.host_id,
    })
}

fn entry_from_descriptor(descriptor: &WindowDescriptor, timestamp_ms: u64) -> WindowRegistryEntry {
    WindowRegistryEntry {
        label: descriptor.label.clone(),
        kind: descriptor.kind,
        route: descriptor.route.clone(),
        title: descriptor.title.clone(),
        visible: true,
        focused: false,
        minimized: false,
        singleton: descriptor.singleton,
        bounds: Some(descriptor.bounds.clone()),
        workspace_id: descriptor.workspace_id.clone(),
        artifact_id: descriptor.artifact_id.clone(),
        session_id: descriptor.session_id.clone(),
        host_id: descriptor.host_id.clone(),
        created_at_ms: timestamp_ms,
        updated_at_ms: timestamp_ms,
    }
}

impl WindowRegistry {
    fn upsert(&mut self, descriptor: &WindowDescriptor, timestamp_ms: u64) -> WindowRegistryEntry {
        if let Some(existing) = self.entries.get_mut(&descriptor.label) {
            existing.route = descriptor.route.clone();
            existing.title = descriptor.title.clone();
            existing.visible = true;
            existing.minimized = false;
            existing.bounds = Some(descriptor.bounds.clone());
            existing.workspace_id = descriptor.workspace_id.clone();
            existing.artifact_id = descriptor.artifact_id.clone();
            existing.session_id = descriptor.session_id.clone();
            existing.host_id = descriptor.host_id.clone();
            existing.updated_at_ms = timestamp_ms;
            return existing.clone();
        }
        let entry = entry_from_descriptor(descriptor, timestamp_ms);
        self.entries.insert(entry.label.clone(), entry.clone());
        entry
    }

    fn mark_visible(
        &mut self,
        label: &str,
        visible: bool,
        timestamp_ms: u64,
    ) -> Option<WindowRegistryEntry> {
        let entry = self.entries.get_mut(label)?;
        entry.visible = visible;
        entry.minimized = false;
        entry.updated_at_ms = timestamp_ms;
        Some(entry.clone())
    }

    fn mark_focused(
        &mut self,
        label: &str,
        focused: bool,
        timestamp_ms: u64,
    ) -> Option<WindowRegistryEntry> {
        if focused {
            for entry in self.entries.values_mut() {
                entry.focused = false;
            }
        }
        let entry = self.entries.get_mut(label)?;
        entry.focused = focused;
        entry.visible = true;
        entry.minimized = false;
        entry.updated_at_ms = timestamp_ms;
        Some(entry.clone())
    }

    fn mark_minimized(&mut self, label: &str, timestamp_ms: u64) -> Option<WindowRegistryEntry> {
        let entry = self.entries.get_mut(label)?;
        entry.focused = false;
        entry.minimized = true;
        entry.visible = true;
        entry.updated_at_ms = timestamp_ms;
        Some(entry.clone())
    }

    fn remove(&mut self, label: &str) -> Option<WindowRegistryEntry> {
        self.entries.remove(label)
    }

    fn snapshot(&self, timestamp_ms: u64) -> WindowRegistrySnapshot {
        WindowRegistrySnapshot {
            windows: self.entries.values().cloned().collect(),
            timestamp_ms,
            restore_failures: Vec::new(),
        }
    }
}

fn registry_lock(
    state: &WindowManagerState,
) -> Result<std::sync::MutexGuard<'_, WindowRegistry>, String> {
    state
        .registry
        .lock()
        .map_err(|_| "window manager registry lock poisoned".to_string())
}

fn event_payload(entry: &WindowRegistryEntry, timestamp_ms: u64) -> WindowEventPayload {
    WindowEventPayload {
        label: entry.label.clone(),
        kind: entry.kind,
        route: entry.route.clone(),
        visible: entry.visible,
        focused: entry.focused,
        timestamp_ms,
        workspace_id: entry.workspace_id.clone(),
        artifact_id: entry.artifact_id.clone(),
        host_id: entry.host_id.clone(),
    }
}

fn emit_window_event(app: &AppHandle, event: &str, entry: &WindowRegistryEntry) {
    let timestamp_ms = now_ms();
    let payload = event_payload(entry, timestamp_ms);
    let _ = app.emit(event, payload);
}

fn emit_registry_updated(app: &AppHandle, snapshot: WindowRegistrySnapshot) {
    let _ = app.emit(REGISTRY_UPDATED_EVENT, snapshot);
}

fn snapshot_state(state: &WindowManagerState) -> Result<WindowRegistrySnapshot, String> {
    let registry = registry_lock(state)?;
    Ok(registry.snapshot(now_ms()))
}

fn layout_path(app: &AppHandle) -> Result<PathBuf, String> {
    let mut dir = app
        .path()
        .app_config_dir()
        .map_err(|err| format!("failed to resolve app config dir: {err}"))?;
    dir.push(LAYOUT_FILE);
    Ok(dir)
}

fn persist_layout(app: &AppHandle, snapshot: &WindowRegistrySnapshot) -> Result<(), String> {
    let path = layout_path(app)?;
    persist_layout_to_path(&path, snapshot)
}

fn persisted_snapshot(snapshot: &WindowRegistrySnapshot) -> WindowRegistrySnapshot {
    let mut persisted = snapshot.clone();
    persisted.restore_failures.clear();
    persisted
}

fn persist_layout_to_path(path: &Path, snapshot: &WindowRegistrySnapshot) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)
            .map_err(|err| format!("failed to create window layout dir: {err}"))?;
    }
    let data = serde_json::to_vec_pretty(&persisted_snapshot(snapshot))
        .map_err(|err| format!("failed to encode window layout: {err}"))?;
    let file_name = path
        .file_name()
        .ok_or_else(|| "window layout path has no file name".to_string())?
        .to_string_lossy();
    let tmp_name = format!(".{file_name}.tmp-{}-{}", std::process::id(), now_ms());
    let tmp_path = path.with_file_name(tmp_name);
    let write_result = (|| -> Result<(), String> {
        let mut file = std::fs::File::create(&tmp_path)
            .map_err(|err| format!("failed to create temp window layout: {err}"))?;
        use std::io::Write;
        file.write_all(&data)
            .map_err(|err| format!("failed to write temp window layout: {err}"))?;
        file.sync_all()
            .map_err(|err| format!("failed to sync temp window layout: {err}"))?;
        std::fs::rename(&tmp_path, path)
            .map_err(|err| format!("failed to replace window layout: {err}"))?;
        if let Some(parent) = path.parent() {
            if let Ok(dir) = std::fs::File::open(parent) {
                let _ = dir.sync_all();
            }
        }
        Ok(())
    })();
    if write_result.is_err() {
        let _ = std::fs::remove_file(&tmp_path);
    }
    write_result
}

fn load_layout(app: &AppHandle) -> Result<WindowRegistrySnapshot, String> {
    let path = layout_path(app)?;
    let data = std::fs::read(path).map_err(|err| format!("failed to read window layout: {err}"))?;
    serde_json::from_slice(&data).map_err(|err| format!("failed to parse window layout: {err}"))
}

fn window_url(route: &str) -> WebviewUrl {
    let clean = route.trim_start_matches('/');
    WebviewUrl::App(format!("index.html#/{clean}").into())
}

fn apply_entry_to_window(window: &WebviewWindow, entry: &WindowRegistryEntry) {
    if let Some(bounds) = &entry.bounds {
        let _ = window.set_position(LogicalPosition::new(bounds.x, bounds.y));
        let _ = window.set_size(LogicalSize::new(bounds.width, bounds.height));
    }
    if entry.visible {
        let _ = window.show();
    }
    if entry.focused {
        let _ = window.set_focus();
    }
}

fn create_or_focus_window(
    app: &AppHandle,
    descriptor: &WindowDescriptor,
) -> Result<WebviewWindow, String> {
    if let Some(existing) = app.get_webview_window(&descriptor.label) {
        let _ = existing.unminimize();
        let _ = existing.show();
        let _ = existing.set_focus();
        return Ok(existing);
    }

    let bounds = &descriptor.bounds;
    let window = WebviewWindowBuilder::new(app, &descriptor.label, window_url(&descriptor.route))
        .title(&descriptor.title)
        .position(bounds.x, bounds.y)
        .inner_size(bounds.width, bounds.height)
        .resizable(true)
        .background_color(tauri::webview::Color(3, 3, 3, 255))
        .visible(false)
        .focused(false)
        .build()
        .map_err(|err| format!("failed to build window {}: {err}", descriptor.label))?;

    let _ = window.show();
    let _ = window.set_focus();
    Ok(window)
}

fn entry_for_label(state: &WindowManagerState, label: &str) -> Result<WindowRegistryEntry, String> {
    validate_window_label(label)?;
    let registry = registry_lock(state)?;
    registry
        .entries
        .get(label)
        .cloned()
        .ok_or_else(|| "window is not registered".to_string())
}

struct WindowRestorePlan {
    entries: Vec<WindowRegistryEntry>,
    descriptors: Vec<WindowDescriptor>,
    failures: Vec<WindowRestoreFailure>,
}

fn label_failure(label: &str, reason: String) -> WindowRestoreFailure {
    WindowRestoreFailure {
        label: if label.trim().is_empty() {
            None
        } else {
            Some(label.to_string())
        },
        reason,
    }
}

fn request_from_restored_entry(entry: &WindowRegistryEntry) -> Result<WindowOpenRequest, String> {
    validate_window_label(&entry.label)?;
    let mut request = WindowOpenRequest {
        kind: entry.kind,
        workspace_id: None,
        artifact_id: None,
        session_id: None,
        host_id: None,
        route: Some(entry.route.clone()),
        title: Some(entry.title.clone()),
        bounds: entry.bounds.clone(),
    };
    match entry.kind {
        WindowKind::MainShell => {
            if entry.label != "main" {
                return Err("main shell restore label mismatch".to_string());
            }
        }
        WindowKind::Workspace => {
            if entry.label == "workspace-main" {
                request.workspace_id = None;
            } else if let Some(id) = entry.label.strip_prefix("workspace-") {
                request.workspace_id = Some(id.to_string());
            } else {
                return Err("workspace restore label mismatch".to_string());
            }
        }
        WindowKind::Terminal => {
            if entry.label == "terminal-panel" {
                request.session_id = None;
            } else if let Some(id) = entry.label.strip_prefix("terminal-") {
                request.session_id = Some(id.to_string());
            } else {
                return Err("terminal restore label mismatch".to_string());
            }
        }
        WindowKind::ArtifactViewer => {
            let id = entry
                .label
                .strip_prefix("artifact-")
                .ok_or_else(|| "artifact restore label mismatch".to_string())?;
            request.artifact_id = Some(id.to_string());
        }
        WindowKind::Settings => {
            if entry.label != "settings" {
                return Err("settings restore label mismatch".to_string());
            }
        }
        WindowKind::MemoryPanel => {
            if entry.label != "memory-panel" {
                return Err("memory panel restore label mismatch".to_string());
            }
        }
        WindowKind::TaskPanel => {
            if entry.label != "task-panel" {
                return Err("task panel restore label mismatch".to_string());
            }
        }
        WindowKind::SystemPanel => {
            if entry.label != "system-panel" {
                return Err("system panel restore label mismatch".to_string());
            }
        }
        WindowKind::Inspector => {
            if entry.label != "inspector" {
                return Err("inspector restore label mismatch".to_string());
            }
        }
        WindowKind::DebugConsole => {
            if !debug_console_enabled() {
                return Err("debug console restore is disabled by default".to_string());
            }
            if entry.label != "debug-console" {
                return Err("debug console restore label mismatch".to_string());
            }
        }
        WindowKind::ShellHost => {
            let host_id = entry
                .host_id
                .clone()
                .or_else(|| {
                    entry
                        .label
                        .strip_prefix("forge-monitor-")
                        .map(str::to_string)
                })
                .ok_or_else(|| "shell host restore label mismatch".to_string())?;
            request.host_id = Some(host_id);
        }
    }
    Ok(request)
}

fn restored_entry_from_descriptor(
    descriptor: &WindowDescriptor,
    source: &WindowRegistryEntry,
    timestamp_ms: u64,
) -> WindowRegistryEntry {
    let mut entry = entry_from_descriptor(descriptor, timestamp_ms);
    entry.visible = source.visible;
    entry.focused = source.focused;
    entry.minimized = source.minimized;
    entry.created_at_ms = if source.created_at_ms > 0 {
        source.created_at_ms
    } else {
        timestamp_ms
    };
    entry.updated_at_ms = timestamp_ms;
    entry
}

fn restored_descriptor(entry: &WindowRegistryEntry) -> Result<WindowDescriptor, String> {
    let descriptor = descriptor_from_request(request_from_restored_entry(entry)?)?;
    if descriptor.label != entry.label {
        return Err("restored label does not match backend label policy".to_string());
    }
    Ok(descriptor)
}

fn restore_plan_from_snapshot(
    snapshot: WindowRegistrySnapshot,
    timestamp_ms: u64,
) -> WindowRestorePlan {
    let mut plan = WindowRestorePlan {
        entries: Vec::new(),
        descriptors: Vec::new(),
        failures: Vec::new(),
    };
    for entry in snapshot.windows {
        match restored_descriptor(&entry) {
            Ok(descriptor) => {
                plan.entries.push(restored_entry_from_descriptor(
                    &descriptor,
                    &entry,
                    timestamp_ms,
                ));
                plan.descriptors.push(descriptor);
            }
            Err(reason) => plan.failures.push(label_failure(&entry.label, reason)),
        }
    }
    plan
}

pub fn register_main_window(app: &AppHandle, state: &WindowManagerState) -> Result<(), String> {
    let descriptor = descriptor_from_request(WindowOpenRequest {
        kind: WindowKind::MainShell,
        workspace_id: None,
        artifact_id: None,
        session_id: None,
        host_id: None,
        route: Some("/".to_string()),
        title: Some("FORGE".to_string()),
        bounds: None,
    })?;
    let timestamp_ms = now_ms();
    let snapshot = {
        let mut registry = registry_lock(state)?;
        let mut entry = registry.upsert(&descriptor, timestamp_ms);
        entry.focused = true;
        registry.entries.insert(entry.label.clone(), entry);
        registry.snapshot(timestamp_ms)
    };
    emit_registry_updated(app, snapshot);
    Ok(())
}

pub fn handle_window_event(
    app: &AppHandle,
    state: &WindowManagerState,
    label: &str,
    event: &WindowEvent,
) {
    if validate_window_label(label).is_err() {
        return;
    }
    let timestamp_ms = now_ms();
    let mut event_to_emit: Option<(&str, WindowRegistryEntry)> = None;
    let snapshot = {
        let Ok(mut registry) = registry_lock(state) else {
            return;
        };
        match event {
            WindowEvent::Focused(focused) => {
                if let Some(entry) = registry.mark_focused(label, *focused, timestamp_ms) {
                    if *focused {
                        event_to_emit = Some((FOCUSED_EVENT, entry));
                    }
                }
            }
            WindowEvent::Destroyed => {
                if let Some(entry) = registry.remove(label) {
                    event_to_emit = Some((CLOSED_EVENT, entry));
                }
            }
            WindowEvent::Moved(position) => {
                if let Some(entry) = registry.entries.get_mut(label) {
                    let mut bounds = entry
                        .bounds
                        .clone()
                        .unwrap_or_else(|| default_bounds(entry.kind));
                    bounds.x = position.x as f64;
                    bounds.y = position.y as f64;
                    entry.bounds = Some(bounds);
                    entry.updated_at_ms = timestamp_ms;
                }
            }
            WindowEvent::Resized(size) => {
                if let Some(entry) = registry.entries.get_mut(label) {
                    let mut bounds = entry
                        .bounds
                        .clone()
                        .unwrap_or_else(|| default_bounds(entry.kind));
                    bounds.width = size.width as f64;
                    bounds.height = size.height as f64;
                    entry.bounds = Some(bounds);
                    entry.updated_at_ms = timestamp_ms;
                }
            }
            _ => {}
        }
        registry.snapshot(timestamp_ms)
    };
    if let Some((event_name, entry)) = event_to_emit {
        emit_window_event(app, event_name, &entry);
    }
    emit_registry_updated(app, snapshot.clone());
    let _ = persist_layout(app, &snapshot);
}

#[tauri::command]
pub async fn forge_window_open(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    request: WindowOpenRequest,
) -> Result<WindowRegistryEntry, String> {
    let descriptor = descriptor_from_request(request)?;
    let window = create_or_focus_window(&app, &descriptor)?;
    let timestamp_ms = now_ms();
    let (entry, snapshot) = {
        let mut registry = registry_lock(&state)?;
        let mut entry = registry.upsert(&descriptor, timestamp_ms);
        entry.focused = true;
        entry.visible = true;
        entry.minimized = false;
        registry.entries.insert(entry.label.clone(), entry.clone());
        (entry, registry.snapshot(timestamp_ms))
    };
    let _ = window.set_focus();
    emit_window_event(&app, OPENED_EVENT, &entry);
    emit_window_event(&app, FOCUSED_EVENT, &entry);
    emit_registry_updated(&app, snapshot.clone());
    persist_layout(&app, &snapshot)?;
    Ok(entry)
}

#[tauri::command]
pub fn forge_window_close(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    label: String,
) -> Result<bool, String> {
    validate_window_label(&label)?;
    if label == "main" {
        return Err("main shell window cannot be closed through window manager".to_string());
    }
    let entry = entry_for_label(&state, &label)?;
    if let Some(window) = app.get_webview_window(&label) {
        window
            .close()
            .map_err(|err| format!("failed to close window {label}: {err}"))?;
    }
    let snapshot = {
        let mut registry = registry_lock(&state)?;
        registry.remove(&label);
        registry.snapshot(now_ms())
    };
    emit_window_event(&app, CLOSED_EVENT, &entry);
    emit_registry_updated(&app, snapshot.clone());
    persist_layout(&app, &snapshot)?;
    Ok(true)
}

#[tauri::command]
pub fn forge_window_focus(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    label: String,
) -> Result<bool, String> {
    validate_window_label(&label)?;
    let window = app
        .get_webview_window(&label)
        .ok_or_else(|| "window is missing".to_string())?;
    let _ = window.unminimize();
    let _ = window.show();
    window
        .set_focus()
        .map_err(|err| format!("failed to focus window {label}: {err}"))?;
    let timestamp_ms = now_ms();
    let (entry, snapshot) = {
        let mut registry = registry_lock(&state)?;
        let entry = registry
            .mark_focused(&label, true, timestamp_ms)
            .ok_or_else(|| "window is not registered".to_string())?;
        (entry, registry.snapshot(timestamp_ms))
    };
    emit_window_event(&app, FOCUSED_EVENT, &entry);
    emit_registry_updated(&app, snapshot.clone());
    persist_layout(&app, &snapshot)?;
    Ok(true)
}

#[tauri::command]
pub fn forge_window_show(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    label: String,
) -> Result<bool, String> {
    validate_window_label(&label)?;
    let window = app
        .get_webview_window(&label)
        .ok_or_else(|| "window is missing".to_string())?;
    let _ = window.unminimize();
    window
        .show()
        .map_err(|err| format!("failed to show window {label}: {err}"))?;
    let timestamp_ms = now_ms();
    let (entry, snapshot) = {
        let mut registry = registry_lock(&state)?;
        let entry = registry
            .mark_visible(&label, true, timestamp_ms)
            .ok_or_else(|| "window is not registered".to_string())?;
        (entry, registry.snapshot(timestamp_ms))
    };
    emit_window_event(&app, SHOWN_EVENT, &entry);
    emit_registry_updated(&app, snapshot.clone());
    persist_layout(&app, &snapshot)?;
    Ok(true)
}

#[tauri::command]
pub fn forge_window_hide(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    label: String,
) -> Result<bool, String> {
    validate_window_label(&label)?;
    if label == "main" {
        return Err("main shell window cannot be hidden through window manager".to_string());
    }
    let window = app
        .get_webview_window(&label)
        .ok_or_else(|| "window is missing".to_string())?;
    window
        .hide()
        .map_err(|err| format!("failed to hide window {label}: {err}"))?;
    let timestamp_ms = now_ms();
    let (entry, snapshot) = {
        let mut registry = registry_lock(&state)?;
        let entry = registry
            .mark_visible(&label, false, timestamp_ms)
            .ok_or_else(|| "window is not registered".to_string())?;
        (entry, registry.snapshot(timestamp_ms))
    };
    emit_window_event(&app, HIDDEN_EVENT, &entry);
    emit_registry_updated(&app, snapshot.clone());
    persist_layout(&app, &snapshot)?;
    Ok(true)
}

#[tauri::command]
pub fn forge_window_toggle(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    label: String,
) -> Result<bool, String> {
    let entry = entry_for_label(&state, &label)?;
    if entry.visible && !entry.minimized {
        forge_window_hide(app, state, label)
    } else {
        forge_window_show(app, state, label)
    }
}

#[tauri::command]
pub fn forge_window_minimize(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    label: String,
) -> Result<bool, String> {
    validate_window_label(&label)?;
    if label == "main" {
        return Err("main shell window cannot be minimized through window manager".to_string());
    }
    let window = app
        .get_webview_window(&label)
        .ok_or_else(|| "window is missing".to_string())?;
    window
        .minimize()
        .map_err(|err| format!("failed to minimize window {label}: {err}"))?;
    let timestamp_ms = now_ms();
    let (entry, snapshot) = {
        let mut registry = registry_lock(&state)?;
        let entry = registry
            .mark_minimized(&label, timestamp_ms)
            .ok_or_else(|| "window is not registered".to_string())?;
        (entry, registry.snapshot(timestamp_ms))
    };
    emit_window_event(&app, HIDDEN_EVENT, &entry);
    emit_registry_updated(&app, snapshot.clone());
    persist_layout(&app, &snapshot)?;
    Ok(true)
}

#[tauri::command]
pub fn forge_window_list(
    state: tauri::State<'_, WindowManagerState>,
) -> Result<Vec<WindowRegistryEntry>, String> {
    Ok(snapshot_state(&state)?.windows)
}

#[tauri::command]
pub fn forge_window_snapshot(
    state: tauri::State<'_, WindowManagerState>,
) -> Result<WindowRegistrySnapshot, String> {
    snapshot_state(&state)
}

#[tauri::command]
pub async fn forge_window_restore_layout(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
    layout_id: Option<String>,
) -> Result<WindowRegistrySnapshot, String> {
    if layout_id
        .as_deref()
        .map(|value| value != "default")
        .unwrap_or(false)
    {
        return Err("only the default window layout is supported".to_string());
    }
    let loaded = load_layout(&app)?;
    let mut plan = restore_plan_from_snapshot(loaded, now_ms());
    let mut restored_entries = Vec::new();
    for (descriptor, entry) in plan.descriptors.iter().zip(plan.entries.iter()) {
        if entry.label == "main" {
            restored_entries.push(entry.clone());
            continue;
        }
        match create_or_focus_window(&app, descriptor) {
            Ok(window) => {
                apply_entry_to_window(&window, entry);
                restored_entries.push(entry.clone());
            }
            Err(reason) => plan.failures.push(label_failure(&entry.label, reason)),
        }
    }
    if restored_entries.is_empty() && !plan.failures.is_empty() {
        return Err(format!(
            "window layout restore failed for {} window(s)",
            plan.failures.len()
        ));
    }
    {
        let mut registry = registry_lock(&state)?;
        registry.entries.clear();
        for entry in restored_entries {
            registry.entries.insert(entry.label.clone(), entry);
        }
    }
    let mut snapshot = snapshot_state(&state)?;
    snapshot.restore_failures = plan.failures;
    let _ = app.emit(LAYOUT_RESTORED_EVENT, snapshot.clone());
    emit_registry_updated(&app, snapshot.clone());
    persist_layout(&app, &snapshot)?;
    Ok(snapshot)
}

#[tauri::command]
pub fn forge_window_sync_state(
    app: AppHandle,
    state: tauri::State<'_, WindowManagerState>,
) -> Result<WindowRegistrySnapshot, String> {
    let snapshot = snapshot_state(&state)?;
    emit_registry_updated(&app, snapshot.clone());
    Ok(snapshot)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn open_request(kind: WindowKind) -> WindowOpenRequest {
        WindowOpenRequest {
            kind,
            workspace_id: None,
            artifact_id: None,
            session_id: None,
            host_id: None,
            route: None,
            title: None,
            bounds: None,
        }
    }

    #[test]
    fn static_kind_labels_are_stable() {
        let settings = descriptor_from_request(open_request(WindowKind::Settings)).unwrap();
        assert_eq!(settings.label, "settings");
        assert_eq!(settings.route, "/settings");
        assert!(settings.singleton);
    }

    #[test]
    fn debug_console_is_disabled_by_default() {
        let debug = descriptor_from_request(open_request(WindowKind::DebugConsole));
        assert!(debug.is_err());
        assert!(debug
            .unwrap_err()
            .contains("debug console window is disabled by default"));
    }

    #[test]
    fn dynamic_labels_are_sanitized() {
        let artifact = descriptor_from_request(WindowOpenRequest {
            kind: WindowKind::ArtifactViewer,
            artifact_id: Some("abc-123_def.4".to_string()),
            workspace_id: None,
            session_id: None,
            host_id: None,
            route: None,
            title: None,
            bounds: None,
        })
        .unwrap();
        assert_eq!(artifact.label, "artifact-abc-123_def.4");

        let bad = descriptor_from_request(WindowOpenRequest {
            kind: WindowKind::ArtifactViewer,
            artifact_id: Some("../secret".to_string()),
            workspace_id: None,
            session_id: None,
            host_id: None,
            route: None,
            title: None,
            bounds: None,
        });
        assert!(bad.is_err());
    }

    #[test]
    fn invalid_routes_are_rejected() {
        let route = descriptor_from_request(WindowOpenRequest {
            route: Some("https://example.invalid".to_string()),
            ..open_request(WindowKind::Settings)
        });
        assert!(route.is_err());

        let traversal = descriptor_from_request(WindowOpenRequest {
            route: Some("/../../secret".to_string()),
            ..open_request(WindowKind::Settings)
        });
        assert!(traversal.is_err());
    }

    #[test]
    fn registry_prevents_singleton_duplicate_entries() {
        let descriptor = descriptor_from_request(open_request(WindowKind::Settings)).unwrap();
        let mut registry = WindowRegistry::default();
        registry.upsert(&descriptor, 100);
        registry.upsert(&descriptor, 200);

        let snapshot = registry.snapshot(250);
        assert_eq!(snapshot.windows.len(), 1);
        assert_eq!(snapshot.windows[0].label, "settings");
        assert_eq!(snapshot.windows[0].created_at_ms, 100);
        assert_eq!(snapshot.windows[0].updated_at_ms, 200);
    }

    #[test]
    fn snapshot_serializes_safe_metadata_only() {
        let descriptor = descriptor_from_request(WindowOpenRequest {
            kind: WindowKind::Terminal,
            session_id: Some("session-1".to_string()),
            workspace_id: None,
            artifact_id: None,
            host_id: None,
            route: None,
            title: Some("Terminal".to_string()),
            bounds: None,
        })
        .unwrap();
        let mut registry = WindowRegistry::default();
        registry.upsert(&descriptor, 100);
        let encoded = serde_json::to_string(&registry.snapshot(200)).unwrap();

        assert!(encoded.contains("terminal-session-1"));
        assert!(encoded.contains("Terminal"));
        assert!(!encoded.contains("scrollback"));
        assert!(!encoded.contains("secret"));
    }

    #[test]
    fn focus_is_global_and_singular() {
        let settings = descriptor_from_request(open_request(WindowKind::Settings)).unwrap();
        let inspector = descriptor_from_request(open_request(WindowKind::Inspector)).unwrap();
        let mut registry = WindowRegistry::default();
        registry.upsert(&settings, 100);
        registry.upsert(&inspector, 100);
        registry.mark_focused("settings", true, 200);
        registry.mark_focused("inspector", true, 300);

        let snapshot = registry.snapshot(350);
        let focused: Vec<_> = snapshot
            .windows
            .iter()
            .filter(|entry| entry.focused)
            .map(|entry| entry.label.as_str())
            .collect();
        assert_eq!(focused, vec!["inspector"]);
    }

    #[test]
    fn labels_reject_shell_and_path_characters() {
        assert!(validate_window_label("settings").is_ok());
        assert!(validate_window_label("artifact-safe_1.2").is_ok());
        assert!(validate_window_label("bad/label").is_err());
        assert!(validate_window_label("bad;label").is_err());
        assert!(validate_window_label("bad label").is_err());
        assert!(validate_window_label("bad..label").is_err());
    }

    #[test]
    fn shell_host_labels_are_backend_derived_from_sanitized_host_ids() {
        let host = descriptor_from_request(WindowOpenRequest {
            kind: WindowKind::ShellHost,
            host_id: Some("2".to_string()),
            route: None,
            title: None,
            ..open_request(WindowKind::ShellHost)
        })
        .unwrap();

        assert_eq!(host.label, "forge-monitor-2");
        assert_eq!(host.route, "/?host=forge-monitor-2");
        assert_eq!(host.host_id.as_deref(), Some("2"));
        assert!(!host.singleton);

        let existing_label = descriptor_from_request(WindowOpenRequest {
            kind: WindowKind::ShellHost,
            host_id: Some("forge-monitor-3".to_string()),
            route: None,
            title: None,
            ..open_request(WindowKind::ShellHost)
        })
        .unwrap();
        assert_eq!(existing_label.label, "forge-monitor-3");

        let bad = descriptor_from_request(WindowOpenRequest {
            kind: WindowKind::ShellHost,
            host_id: Some("../../forge-monitor-2".to_string()),
            route: None,
            title: None,
            ..open_request(WindowKind::ShellHost)
        });
        assert!(bad.is_err());
    }

    #[test]
    fn persist_layout_writes_json_through_temp_file_then_rename() {
        let mut path = std::env::temp_dir();
        path.push(format!(
            "forge-window-layout-test-{}-{}.json",
            std::process::id(),
            now_ms()
        ));
        std::fs::write(&path, br#"{"windows":[],"timestampMs":1}"#).unwrap();

        let descriptor = descriptor_from_request(open_request(WindowKind::Settings)).unwrap();
        let mut registry = WindowRegistry::default();
        registry.upsert(&descriptor, 100);
        let snapshot = registry.snapshot(200);

        persist_layout_to_path(&path, &snapshot).unwrap();

        let encoded = std::fs::read_to_string(&path).unwrap();
        assert!(encoded.contains("settings"));
        assert!(encoded.contains("timestampMs"));

        let parent = path.parent().unwrap();
        let tmp_prefix = format!(
            ".{}.tmp-{}-",
            path.file_name().unwrap().to_string_lossy(),
            std::process::id()
        );
        let leaked_temp = std::fs::read_dir(parent)
            .unwrap()
            .filter_map(Result::ok)
            .any(|entry| entry.file_name().to_string_lossy().starts_with(&tmp_prefix));
        assert!(!leaked_temp);

        let _ = std::fs::remove_file(path);
    }

    #[test]
    fn restore_plan_reports_bad_entries_without_dropping_valid_entries() {
        let good = entry_from_descriptor(
            &descriptor_from_request(open_request(WindowKind::Settings)).unwrap(),
            100,
        );
        let mut bad = good.clone();
        bad.label = "bad/label".to_string();
        bad.route = "/settings".to_string();

        let snapshot = WindowRegistrySnapshot {
            windows: vec![bad, good],
            timestamp_ms: 200,
            restore_failures: Vec::new(),
        };
        let plan = restore_plan_from_snapshot(snapshot, 300);

        assert_eq!(plan.entries.len(), 1);
        assert_eq!(plan.entries[0].label, "settings");
        assert_eq!(plan.failures.len(), 1);
        assert_eq!(plan.failures[0].label.as_deref(), Some("bad/label"));
        assert!(plan.failures[0].reason.contains("label"));
    }

    #[test]
    fn restore_plan_rejects_mismatched_kind_label_and_unsafe_route() {
        let mut mismatched = entry_from_descriptor(
            &descriptor_from_request(open_request(WindowKind::Settings)).unwrap(),
            100,
        );
        mismatched.kind = WindowKind::ArtifactViewer;
        mismatched.artifact_id = Some("report".to_string());

        let mut unsafe_route = entry_from_descriptor(
            &descriptor_from_request(WindowOpenRequest {
                kind: WindowKind::ShellHost,
                host_id: Some("2".to_string()),
                ..open_request(WindowKind::ShellHost)
            })
            .unwrap(),
            100,
        );
        unsafe_route.route = "https://example.invalid".to_string();

        let plan = restore_plan_from_snapshot(
            WindowRegistrySnapshot {
                windows: vec![mismatched, unsafe_route],
                timestamp_ms: 200,
                restore_failures: Vec::new(),
            },
            300,
        );

        assert!(plan.entries.is_empty());
        assert_eq!(plan.failures.len(), 2);
    }

    #[test]
    fn restore_plan_rejects_debug_console_by_default() {
        let debug = WindowRegistryEntry {
            label: "debug-console".to_string(),
            kind: WindowKind::DebugConsole,
            route: "/system?surface=debug-console".to_string(),
            title: "FORGE Debug Console".to_string(),
            visible: true,
            focused: false,
            minimized: false,
            singleton: true,
            bounds: Some(default_bounds(WindowKind::DebugConsole)),
            workspace_id: None,
            artifact_id: None,
            session_id: None,
            host_id: None,
            created_at_ms: 100,
            updated_at_ms: 100,
        };

        let plan = restore_plan_from_snapshot(
            WindowRegistrySnapshot {
                windows: vec![debug],
                timestamp_ms: 200,
                restore_failures: Vec::new(),
            },
            300,
        );

        assert!(plan.entries.is_empty());
        assert_eq!(plan.failures.len(), 1);
        assert_eq!(plan.failures[0].label.as_deref(), Some("debug-console"));
        assert!(plan.failures[0].reason.contains("disabled by default"));
    }
}
