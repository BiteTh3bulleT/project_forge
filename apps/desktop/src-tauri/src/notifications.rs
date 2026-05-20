use serde::Serialize;

const DEFAULT_NOTIFICATION_ID: u32 = 1;
const MAX_FIELD_CHARS: usize = 2048;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NotificationServerInfo {
    pub name: &'static str,
    pub vendor: &'static str,
    pub version: &'static str,
    pub spec_version: &'static str,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct NotificationRequest {
    pub id: u32,
    pub app_name: String,
    pub app_icon: String,
    pub summary: String,
    pub body: String,
    pub expire_timeout_ms: Option<i32>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct NotificationClosed {
    pub id: u32,
    pub reason: u32,
}

pub fn server_info() -> NotificationServerInfo {
    NotificationServerInfo {
        name: "FORGE",
        vendor: "FORGE",
        version: "1.0",
        spec_version: "1.2",
    }
}

pub fn capabilities() -> [&'static str; 3] {
    ["body", "body-markup", "persistence"]
}

pub fn normalize_notification_request(
    app_name: &str,
    replaces_id: u32,
    app_icon: &str,
    summary: &str,
    body: &str,
    expire_timeout: i32,
) -> NotificationRequest {
    let id = if replaces_id > 0 {
        replaces_id
    } else {
        DEFAULT_NOTIFICATION_ID
    };
    normalize_notification_request_with_id(app_name, id, app_icon, summary, body, expire_timeout)
}

fn normalize_notification_request_with_id(
    app_name: &str,
    id: u32,
    app_icon: &str,
    summary: &str,
    body: &str,
    expire_timeout: i32,
) -> NotificationRequest {
    NotificationRequest {
        id,
        app_name: clean_field(app_name, "Unknown"),
        app_icon: clean_field(app_icon, ""),
        summary: clean_field(summary, "Notification"),
        body: clean_field(body, ""),
        expire_timeout_ms: if expire_timeout >= 0 {
            Some(expire_timeout)
        } else {
            None
        },
    }
}

fn clean_field(value: &str, fallback: &str) -> String {
    let trimmed = value.trim();
    let source = if trimmed.is_empty() {
        fallback
    } else {
        trimmed
    };
    source.chars().take(MAX_FIELD_CHARS).collect()
}

pub fn start_freedesktop_service(app: tauri::AppHandle) {
    #[cfg(target_os = "linux")]
    {
        tauri::async_runtime::spawn(async move {
            if let Err(err) = run_freedesktop_service(app).await {
                eprintln!("FORGE notification service unavailable: {err}");
            }
        });
    }

    #[cfg(not(target_os = "linux"))]
    {
        let _ = app;
    }
}

#[cfg(target_os = "linux")]
#[derive(Clone)]
struct NotificationService {
    app: tauri::AppHandle,
    next_id: std::sync::Arc<std::sync::atomic::AtomicU32>,
}

#[cfg(target_os = "linux")]
impl NotificationService {
    fn new(app: tauri::AppHandle) -> Self {
        Self {
            app,
            next_id: std::sync::Arc::new(std::sync::atomic::AtomicU32::new(1)),
        }
    }

    fn allocate_id(&self) -> u32 {
        self.next_id
            .fetch_add(1, std::sync::atomic::Ordering::Relaxed)
    }
}

#[cfg(target_os = "linux")]
#[zbus::interface(name = "org.freedesktop.Notifications")]
impl NotificationService {
    #[zbus(name = "GetCapabilities")]
    fn get_capabilities(&self) -> Vec<&'static str> {
        capabilities().to_vec()
    }

    #[zbus(name = "GetServerInformation")]
    fn get_server_information(&self) -> (&'static str, &'static str, &'static str, &'static str) {
        let info = server_info();
        (info.name, info.vendor, info.version, info.spec_version)
    }

    #[zbus(name = "Notify")]
    fn notify(
        &self,
        app_name: &str,
        replaces_id: u32,
        app_icon: &str,
        summary: &str,
        body: &str,
        _actions: Vec<String>,
        _hints: std::collections::HashMap<String, zbus::zvariant::OwnedValue>,
        expire_timeout: i32,
    ) -> u32 {
        let id = if replaces_id > 0 {
            replaces_id
        } else {
            self.allocate_id()
        };
        let request = normalize_notification_request_with_id(
            app_name,
            id,
            app_icon,
            summary,
            body,
            expire_timeout,
        );
        use tauri::Emitter;
        let _ = self.app.emit("forge://notification", &request);
        request.id
    }

    #[zbus(name = "CloseNotification")]
    async fn close_notification(
        &self,
        id: u32,
        #[zbus(signal_emitter)] emitter: zbus::object_server::SignalEmitter<'_>,
    ) -> zbus::fdo::Result<()> {
        let closed = NotificationClosed { id, reason: 2 };
        use tauri::Emitter;
        let _ = self.app.emit("forge://notification-closed", &closed);
        emitter.notification_closed(id, closed.reason).await?;
        Ok(())
    }

    #[zbus(signal, name = "NotificationClosed")]
    async fn notification_closed(
        signal_emitter: &zbus::object_server::SignalEmitter<'_>,
        id: u32,
        reason: u32,
    ) -> zbus::Result<()>;

    #[zbus(signal, name = "ActionInvoked")]
    async fn action_invoked(
        signal_emitter: &zbus::object_server::SignalEmitter<'_>,
        id: u32,
        action_key: &str,
    ) -> zbus::Result<()>;
}

#[cfg(target_os = "linux")]
async fn run_freedesktop_service(app: tauri::AppHandle) -> zbus::Result<()> {
    let service = NotificationService::new(app);
    let _connection = zbus::connection::Builder::session()?
        .name("org.freedesktop.Notifications")?
        .serve_at("/org/freedesktop/Notifications", service)?
        .build()
        .await?;

    std::future::pending::<()>().await;
    #[allow(unreachable_code)]
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn server_info_uses_freedesktop_notification_contract() {
        let info = server_info();

        assert_eq!(info.name, "FORGE");
        assert_eq!(info.vendor, "FORGE");
        assert_eq!(info.version, "1.0");
        assert_eq!(info.spec_version, "1.2");
        assert_eq!(capabilities(), ["body", "body-markup", "persistence"]);
    }

    #[test]
    fn notification_request_trims_and_limits_payloads() {
        let request = normalize_notification_request(
            "  Native App  ",
            0,
            "  forge-icon  ",
            "  Build finished  ",
            "  The desktop shell build completed successfully.  ",
            30_000,
        );

        assert_eq!(request.app_name, "Native App");
        assert_eq!(request.app_icon, "forge-icon");
        assert_eq!(request.summary, "Build finished");
        assert_eq!(
            request.body,
            "The desktop shell build completed successfully."
        );
        assert_eq!(request.expire_timeout_ms, Some(30_000));
        assert_eq!(request.id, 1);
    }

    #[test]
    fn notification_replacement_id_is_preserved_when_supplied() {
        let request = normalize_notification_request("", 42, "", "", "", -1);

        assert_eq!(request.id, 42);
        assert_eq!(request.app_name, "Unknown");
        assert_eq!(request.summary, "Notification");
        assert_eq!(request.body, "");
        assert_eq!(request.expire_timeout_ms, None);
    }
}
