use serde::{Deserialize, Serialize};

#[cfg(target_os = "linux")]
mod wayland;

#[cfg(target_os = "linux")]
pub use wayland::{start, LinuxWindowRegistryState};

#[cfg(not(target_os = "linux"))]
#[derive(Default)]
pub struct LinuxWindowRegistryState;

#[derive(Serialize, Clone, Debug, PartialEq, Eq)]
pub struct LinuxWindowSnapshot {
    id: String,
    title: String,
    app_id: String,
    icon_name: Option<String>,
    icon_path: Option<String>,
    focused: bool,
    minimized: bool,
    native: bool,
    lifecycle: &'static str,
    first_seen_ms: u64,
    last_seen_ms: u64,
}

#[derive(Deserialize, Clone, Copy, Debug, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum LinuxWindowAction {
    Focus,
    Minimize,
    Maximize,
    Fullscreen,
    Close,
}

#[tauri::command]
pub fn list_linux_windows(
    registry: tauri::State<'_, LinuxWindowRegistryState>,
) -> Result<Vec<LinuxWindowSnapshot>, String> {
    #[cfg(target_os = "linux")]
    return registry.list();
    #[cfg(not(target_os = "linux"))]
    {
        let _ = registry;
        Ok(Vec::new())
    }
}

#[tauri::command]
pub fn focus_linux_window(
    registry: tauri::State<'_, LinuxWindowRegistryState>,
    window_id: String,
) -> Result<bool, String> {
    control_linux_window(registry, window_id, LinuxWindowAction::Focus)
}

#[tauri::command]
pub fn control_linux_window(
    registry: tauri::State<'_, LinuxWindowRegistryState>,
    window_id: String,
    action: LinuxWindowAction,
) -> Result<bool, String> {
    #[cfg(target_os = "linux")]
    return registry.request(&window_id, action);
    #[cfg(not(target_os = "linux"))]
    {
        let _ = (registry, window_id, action);
        Ok(false)
    }
}
