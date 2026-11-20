use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};

use crate::desktop_metadata::{find_desktop_file_by_id, find_icon_path, parse_desktop_value};

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
    lifecycle: String,
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

#[derive(Default)]
pub struct LinuxWindowRegistryState {
    registry: Mutex<LinuxWindowRegistry>,
}

#[derive(Default)]
struct LinuxWindowRegistry {
    records: BTreeMap<String, LinuxWindowSnapshot>,
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
        lifecycle: "active".to_string(),
        first_seen_ms: 0,
        last_seen_ms: 0,
    })
}

fn parse_linux_toplevels(raw: &str) -> Result<Vec<LswtToplevel>, String> {
    let output: LswtOutput =
        serde_json::from_str(raw).map_err(|err| format!("failed to parse lswt output: {err}"))?;
    Ok(output.toplevels)
}

fn linux_window_action_name(action: LinuxWindowAction) -> &'static str {
    match action {
        LinuxWindowAction::Focus => "focus",
        LinuxWindowAction::Minimize => "minimize",
        LinuxWindowAction::Maximize => "maximize",
        LinuxWindowAction::Fullscreen => "fullscreen",
        LinuxWindowAction::Close => "close",
    }
}

fn linux_window_match_args(
    observed: &[LswtToplevel],
    window_id: &str,
) -> Result<Vec<String>, String> {
    let mut targets = observed.iter().filter(|window| {
        !window.identifier.trim().is_empty() && window.identifier.trim() == window_id
    });
    let target = targets
        .next()
        .ok_or_else(|| "linux window is no longer active".to_string())?;
    if targets.next().is_some() || is_forge_shell_toplevel(&target.app_id, &target.title) {
        return Err("linux window is not an eligible native target".to_string());
    }

    let match_app = !target.app_id.trim().is_empty();
    let match_title = !target.title.trim().is_empty();
    if !match_app && !match_title {
        return Err("linux window has no usable compositor selector".to_string());
    }

    // wlrctl matches exact raw values, ANDing the supplied attributes. Check
    // all toplevels, including those filtered out of the taskbar.
    if observed.iter().any(|window| {
        !std::ptr::eq(window, target)
            && (!match_app || window.app_id == target.app_id)
            && (!match_title || window.title == target.title)
    }) {
        return Err("Multiple native windows match this action. Use the compositor to select the window directly.".to_string());
    }

    let mut args = Vec::new();
    if match_app {
        args.push(format!("app_id:{}", target.app_id));
    }
    if match_title {
        args.push(format!("title:{}", target.title));
    }
    Ok(args)
}

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_millis().min(u128::from(u64::MAX)) as u64)
        .unwrap_or(0)
}

impl LinuxWindowRegistry {
    fn refresh(&mut self, observed: Vec<LinuxWindowSnapshot>, observed_at_ms: u64) {
        let mut active_ids = BTreeSet::new();
        for mut window in observed {
            active_ids.insert(window.id.clone());
            let first_seen_ms = self
                .records
                .get(&window.id)
                .map(|record| record.first_seen_ms)
                .filter(|value| *value > 0)
                .unwrap_or(observed_at_ms);
            window.lifecycle = "active".to_string();
            window.first_seen_ms = first_seen_ms;
            window.last_seen_ms = observed_at_ms;
            self.records.insert(window.id.clone(), window);
        }

        for (id, record) in self.records.iter_mut() {
            if !active_ids.contains(id) && record.lifecycle == "active" {
                record.lifecycle = "closed".to_string();
                record.last_seen_ms = observed_at_ms;
                record.focused = false;
                record.minimized = false;
            }
        }
    }

    fn active_windows(&self) -> Vec<LinuxWindowSnapshot> {
        self.records
            .values()
            .filter(|window| window.lifecycle == "active")
            .cloned()
            .collect()
    }

    #[cfg(test)]
    fn active_window(&self, window_id: &str) -> Option<LinuxWindowSnapshot> {
        self.records
            .get(window_id)
            .filter(|window| window.lifecycle == "active")
            .cloned()
    }
}

fn read_compositor_linux_windows() -> Result<Vec<LswtToplevel>, String> {
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
    parse_linux_toplevels(&stdout)
}

fn refresh_linux_window_registry(
    registry_state: &LinuxWindowRegistryState,
) -> Result<Vec<LinuxWindowSnapshot>, String> {
    let observed = read_compositor_linux_windows()?
        .into_iter()
        .filter_map(linux_window_from_toplevel)
        .collect();
    let mut registry = registry_state
        .registry
        .lock()
        .map_err(|_| "linux window registry lock poisoned".to_string())?;
    registry.refresh(observed, now_ms());
    Ok(registry.active_windows())
}

#[tauri::command]
pub fn list_linux_windows(
    registry: tauri::State<'_, LinuxWindowRegistryState>,
) -> Result<Vec<LinuxWindowSnapshot>, String> {
    refresh_linux_window_registry(&registry)
}

#[tauri::command]
pub fn focus_linux_window(window_id: String) -> Result<bool, String> {
    control_linux_window(window_id, LinuxWindowAction::Focus)
}

#[tauri::command]
pub fn control_linux_window(window_id: String, action: LinuxWindowAction) -> Result<bool, String> {
    let observed = read_compositor_linux_windows()?;
    let args = linux_window_match_args(&observed, &window_id)?;

    let mut command = std::process::Command::new("wlrctl");
    command.args(["toplevel", linux_window_action_name(action)]);
    command.args(args);

    let output = command
        .output()
        .map_err(|err| format!("failed to run wlrctl: {err}"))?;
    if output.status.success() {
        Ok(true)
    } else {
        Err(String::from_utf8_lossy(&output.stderr).trim().to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

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

        let windows: Vec<_> = parse_linux_toplevels(raw)
            .expect("valid lswt json")
            .into_iter()
            .filter_map(linux_window_from_toplevel)
            .collect();

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
                lifecycle: "active".to_string(),
                first_seen_ms: 0,
                last_seen_ms: 0,
            }]
        );
    }

    #[test]
    fn linux_window_registry_preserves_lifecycle_timestamps() {
        let mut registry = LinuxWindowRegistry::default();
        registry.refresh(
            vec![LinuxWindowSnapshot {
                id: "terminal-window".to_string(),
                title: "Terminal".to_string(),
                app_id: "foot".to_string(),
                icon_name: None,
                icon_path: None,
                focused: true,
                minimized: false,
                native: true,
                lifecycle: "active".to_string(),
                first_seen_ms: 0,
                last_seen_ms: 0,
            }],
            100,
        );
        registry.refresh(
            vec![LinuxWindowSnapshot {
                id: "terminal-window".to_string(),
                title: "Terminal".to_string(),
                app_id: "foot".to_string(),
                icon_name: None,
                icon_path: None,
                focused: false,
                minimized: false,
                native: true,
                lifecycle: "active".to_string(),
                first_seen_ms: 0,
                last_seen_ms: 0,
            }],
            250,
        );

        let active = registry.active_window("terminal-window").unwrap();
        assert_eq!(active.first_seen_ms, 100);
        assert_eq!(active.last_seen_ms, 250);
        assert!(!active.focused);
    }

    #[test]
    fn linux_window_registry_marks_missing_windows_closed() {
        let mut registry = LinuxWindowRegistry::default();
        registry.refresh(
            vec![LinuxWindowSnapshot {
                id: "terminal-window".to_string(),
                title: "Terminal".to_string(),
                app_id: "foot".to_string(),
                icon_name: None,
                icon_path: None,
                focused: true,
                minimized: false,
                native: true,
                lifecycle: "active".to_string(),
                first_seen_ms: 0,
                last_seen_ms: 0,
            }],
            100,
        );
        registry.refresh(Vec::new(), 300);

        assert!(registry.active_window("terminal-window").is_none());
        let record = registry.records.get("terminal-window").unwrap();
        assert_eq!(record.lifecycle, "closed");
        assert_eq!(record.first_seen_ms, 100);
        assert_eq!(record.last_seen_ms, 300);
        assert!(!record.focused);
    }

    #[test]
    fn linux_window_actions_are_allowlisted() {
        assert_eq!(linux_window_action_name(LinuxWindowAction::Focus), "focus");
        assert_eq!(
            linux_window_action_name(LinuxWindowAction::Minimize),
            "minimize"
        );
        assert_eq!(
            linux_window_action_name(LinuxWindowAction::Maximize),
            "maximize"
        );
        assert_eq!(
            linux_window_action_name(LinuxWindowAction::Fullscreen),
            "fullscreen"
        );
        assert_eq!(linux_window_action_name(LinuxWindowAction::Close), "close");
    }

    #[test]
    fn linux_window_match_args_are_not_shell_expanded() {
        let windows = parse_linux_toplevels(
            r#"{"toplevels":[{"identifier":"terminal-window",
                "title":" Terminal: notes ","app-id":"foot"}]}"#,
        )
        .unwrap();

        assert_eq!(
            linux_window_match_args(&windows, "terminal-window").unwrap(),
            ["app_id:foot", "title: Terminal: notes "]
        );
    }

    #[test]
    fn linux_window_actions_refuse_duplicate_selectors() {
        let mut windows = parse_linux_toplevels(
            r#"{"toplevels":[
                {"identifier":"first","title":"Terminal","app-id":"foot"},
                {"identifier":"second","title":"Terminal","app-id":"foot"},
                {"identifier":"third","title":"Build log","app-id":"foot"}
            ]}"#,
        )
        .unwrap();

        for id in ["first", "second"] {
            assert!(linux_window_match_args(&windows, id)
                .unwrap_err()
                .contains("Multiple native windows"));
        }
        assert_eq!(
            linux_window_match_args(&windows, "third").unwrap(),
            ["app_id:foot", "title:Build log"]
        );
        assert!(linux_window_match_args(&windows, "closed").is_err());

        // An untitled window uses only the real app ID, including when the
        // competing window is hidden from the taskbar.
        windows[0].title.clear();
        windows[1].title = "FORGE Build".to_string();
        windows.pop();
        assert!(linux_window_match_args(&windows, "first").is_err());
        assert!(linux_window_match_args(&windows, "second").is_err());
        windows.pop();
        assert_eq!(
            linux_window_match_args(&windows, "first").unwrap(),
            ["app_id:foot"]
        );
    }
}
