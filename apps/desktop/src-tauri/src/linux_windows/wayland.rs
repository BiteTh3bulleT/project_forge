use std::collections::{BTreeMap, HashMap};
use std::sync::{Arc, Mutex};
use std::time::{SystemTime, UNIX_EPOCH};

use tauri::{Emitter, Manager};
use wayland_client::{
    backend::ObjectId,
    event_created_child,
    globals::{registry_queue_init, GlobalListContents},
    protocol::{wl_registry, wl_seat::WlSeat},
    Connection, Dispatch, Proxy, QueueHandle,
};
use wayland_protocols_wlr::foreign_toplevel::v1::client::{
    zwlr_foreign_toplevel_handle_v1::{self as handle, ZwlrForeignToplevelHandleV1},
    zwlr_foreign_toplevel_manager_v1::{self as manager, ZwlrForeignToplevelManagerV1},
};

use super::{LinuxWindowAction, LinuxWindowSnapshot};
use crate::desktop_metadata::{find_desktop_file_by_id, find_icon_path, parse_desktop_value};

#[derive(Default)]
pub struct LinuxWindowRegistryState(Arc<Mutex<Registry>>);

#[derive(Default)]
struct Registry {
    connection: Option<Connection>,
    seat: Option<WlSeat>,
    windows: BTreeMap<String, (ZwlrForeignToplevelHandleV1, LinuxWindowSnapshot)>,
}

impl LinuxWindowRegistryState {
    pub(super) fn list(&self) -> Result<Vec<LinuxWindowSnapshot>, String> {
        let registry = self.0.lock().map_err(|err| err.to_string())?;
        Ok(registry
            .windows
            .values()
            .map(|(_, snapshot)| snapshot.clone())
            .collect())
    }

    pub(super) fn request(&self, id: &str, action: LinuxWindowAction) -> Result<bool, String> {
        let registry = self.0.lock().map_err(|err| err.to_string())?;
        let connection = registry
            .connection
            .as_ref()
            .ok_or("Wayland window controls are unavailable.")?;
        let (window, _) = registry
            .windows
            .get(id)
            .ok_or("This native window has closed.")?;
        let request = match action {
            LinuxWindowAction::Focus => {
                let seat = registry.seat.clone().ok_or("Wayland has no input seat.")?;
                handle::Request::Activate { seat }
            }
            LinuxWindowAction::Minimize => handle::Request::SetMinimized,
            LinuxWindowAction::Maximize => handle::Request::SetMaximized,
            LinuxWindowAction::Fullscreen if window.version() >= 2 => {
                handle::Request::SetFullscreen { output: None }
            }
            LinuxWindowAction::Fullscreen => {
                return Err("This compositor does not support fullscreen requests.".into())
            }
            LinuxWindowAction::Close => handle::Request::Close,
        };
        window
            .send_request(request)
            .map_err(|err| err.to_string())?;
        connection.flush().map_err(|err| err.to_string())?;
        Ok(true)
    }
}

pub fn start(app: tauri::AppHandle) {
    if std::env::var_os("WAYLAND_DISPLAY").is_none() && std::env::var_os("WAYLAND_SOCKET").is_none()
    {
        return;
    }
    let registry = Arc::clone(&app.state::<LinuxWindowRegistryState>().0);
    let worker = std::thread::Builder::new()
        .name("forge-wayland-windows".into())
        .spawn(move || {
            if let Err(err) = observe(Arc::clone(&registry), || {
                let _ = app.emit_to("main", "forge://linux-windows-changed", ());
            }) {
                eprintln!("[forge-shell] Wayland window tracking unavailable: {err}");
            }
        });
    if let Err(err) = worker {
        eprintln!("[forge-shell] Could not start Wayland window tracking: {err}");
    }
}

fn observe(
    registry: Arc<Mutex<Registry>>,
    notify: impl Fn(),
) -> Result<(), Box<dyn std::error::Error>> {
    let connection = Connection::connect_to_env()?;
    let (globals, mut queue) = registry_queue_init::<Observer>(&connection)?;
    let qh = queue.handle();
    let _manager: ZwlrForeignToplevelManagerV1 = globals.bind(&qh, 1..=3, ())?;
    let seat = globals.bind(&qh, 1..=1, ()).ok();
    {
        let mut shared = registry.lock().unwrap();
        shared.connection = Some(connection);
        shared.seat = seat;
    }
    let mut observer = Observer {
        registry,
        pending: HashMap::new(),
        next_id: 0,
        changed: false,
        finished: false,
    };
    let result = loop {
        if let Err(err) = queue.blocking_dispatch(&mut observer) {
            break Err(err.into());
        }
        if observer.finished {
            break Ok(());
        }
        if std::mem::take(&mut observer.changed) {
            notify();
        }
    };
    *observer.registry.lock().unwrap() = Registry::default();
    notify();
    result
}

struct Observer {
    registry: Arc<Mutex<Registry>>,
    pending: HashMap<ObjectId, PendingWindow>,
    next_id: u64,
    changed: bool,
    finished: bool,
}

struct PendingWindow {
    id: String,
    title: String,
    app_id: String,
    focused: bool,
    minimized: bool,
    first_seen_ms: u64,
}

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_millis() as u64)
        .unwrap_or_default()
}

fn visible(app_id: &str, title: &str) -> bool {
    let app_id = app_id.to_ascii_lowercase();
    let title = title.to_ascii_lowercase();
    (!app_id.is_empty() || !title.is_empty())
        && !app_id.contains("forge_desktop")
        && app_id != "dev.forge.workshop"
        && title != "forge"
        && !title.starts_with("forge build")
}

fn resolve_icon(app_id: &str) -> (Option<String>, Option<String>) {
    if app_id.is_empty() {
        return (None, None);
    }
    for desktop_id in [
        format!("{app_id}.desktop"),
        format!("{}.desktop", app_id.to_ascii_lowercase()),
    ] {
        if let Some((_, contents)) = find_desktop_file_by_id(&desktop_id) {
            if let Some(icon) = parse_desktop_value(&contents, "Icon") {
                let path = find_icon_path(&icon);
                return (Some(icon), path);
            }
        }
    }
    (Some(app_id.to_string()), find_icon_path(app_id))
}

impl Observer {
    fn publish(&mut self, window: &ZwlrForeignToplevelHandleV1) {
        let Some(pending) = self.pending.get(&window.id()) else {
            return;
        };
        let app_id = pending.app_id.trim();
        let title = pending.title.trim();
        let mut registry = self.registry.lock().unwrap();
        if !visible(app_id, title) {
            self.changed |= registry.windows.remove(&pending.id).is_some();
            return;
        }
        let title = if title.is_empty() { app_id } else { title };
        let previous = registry
            .windows
            .get(&pending.id)
            .map(|(_, snapshot)| snapshot);
        let same_app = previous.filter(|snapshot| snapshot.app_id == app_id);
        if same_app.is_some_and(|snapshot| {
            snapshot.title == title
                && snapshot.focused == pending.focused
                && snapshot.minimized == pending.minimized
        }) {
            return;
        }
        let (icon_name, icon_path) = same_app
            .map(|snapshot| (snapshot.icon_name.clone(), snapshot.icon_path.clone()))
            .unwrap_or_else(|| resolve_icon(app_id));
        registry.windows.insert(
            pending.id.clone(),
            (
                window.clone(),
                LinuxWindowSnapshot {
                    id: pending.id.clone(),
                    title: title.into(),
                    app_id: app_id.into(),
                    icon_name,
                    icon_path,
                    focused: pending.focused,
                    minimized: pending.minimized,
                    native: true,
                    lifecycle: "active",
                    first_seen_ms: pending.first_seen_ms,
                    last_seen_ms: now_ms(),
                },
            ),
        );
        self.changed = true;
    }
}

impl Dispatch<ZwlrForeignToplevelManagerV1, ()> for Observer {
    fn event(
        state: &mut Self,
        _: &ZwlrForeignToplevelManagerV1,
        event: manager::Event,
        _: &(),
        _: &Connection,
        _: &QueueHandle<Self>,
    ) {
        match event {
            manager::Event::Toplevel { toplevel } => {
                state.next_id += 1;
                state.pending.insert(
                    toplevel.id(),
                    PendingWindow {
                        id: format!("wayland-{}", state.next_id),
                        title: String::new(),
                        app_id: String::new(),
                        focused: false,
                        minimized: false,
                        first_seen_ms: now_ms(),
                    },
                );
            }
            manager::Event::Finished => state.finished = true,
            _ => {}
        }
    }

    event_created_child!(Observer, ZwlrForeignToplevelManagerV1, [
        manager::EVT_TOPLEVEL_OPCODE => (ZwlrForeignToplevelHandleV1, ())
    ]);
}

impl Dispatch<ZwlrForeignToplevelHandleV1, ()> for Observer {
    fn event(
        state: &mut Self,
        window: &ZwlrForeignToplevelHandleV1,
        event: handle::Event,
        _: &(),
        _: &Connection,
        _: &QueueHandle<Self>,
    ) {
        let Some(pending) = state.pending.get_mut(&window.id()) else {
            return;
        };
        match event {
            handle::Event::Title { title } => pending.title = title,
            handle::Event::AppId { app_id } => pending.app_id = app_id,
            handle::Event::State { state: flags } => {
                let has = |value: handle::State| {
                    flags
                        .chunks_exact(4)
                        .any(|flag| flag == (value as u32).to_ne_bytes())
                };
                pending.focused = has(handle::State::Activated);
                pending.minimized = has(handle::State::Minimized);
            }
            handle::Event::Done => state.publish(window),
            handle::Event::Closed => {
                if let Some(closed) = state.pending.remove(&window.id()) {
                    state.changed |= state
                        .registry
                        .lock()
                        .unwrap()
                        .windows
                        .remove(&closed.id)
                        .is_some();
                }
                window.destroy();
            }
            _ => {}
        }
    }
}

impl Dispatch<wl_registry::WlRegistry, GlobalListContents> for Observer {
    fn event(
        _: &mut Self,
        _: &wl_registry::WlRegistry,
        _: wl_registry::Event,
        _: &GlobalListContents,
        _: &Connection,
        _: &QueueHandle<Self>,
    ) {
    }
}

wayland_client::delegate_noop!(Observer: ignore WlSeat);
