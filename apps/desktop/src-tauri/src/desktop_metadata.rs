use std::path::{Path, PathBuf};

pub fn application_dirs() -> Vec<PathBuf> {
    vec![
        PathBuf::from("/run/current-system/sw/share/applications"),
        PathBuf::from("/usr/share/applications"),
        PathBuf::from("/usr/local/share/applications"),
    ]
}

pub fn icon_dirs() -> Vec<PathBuf> {
    vec![
        PathBuf::from("/run/current-system/sw/share/icons/hicolor"),
        PathBuf::from("/run/current-system/sw/share/icons/Adwaita"),
        PathBuf::from("/run/current-system/sw/share/pixmaps"),
        PathBuf::from("/usr/share/icons/hicolor"),
        PathBuf::from("/usr/share/icons/Adwaita"),
        PathBuf::from("/usr/share/pixmaps"),
    ]
}

pub fn parse_desktop_value(contents: &str, key: &str) -> Option<String> {
    let prefix = format!("{key}=");
    contents.lines().find_map(|line| {
        let trimmed = line.trim();
        trimmed
            .strip_prefix(&prefix)
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty())
    })
}

pub fn find_desktop_file_for_ids(desktop_ids: &[&str]) -> Option<(PathBuf, String)> {
    for dir in application_dirs() {
        for desktop_id in desktop_ids {
            let candidate = dir.join(desktop_id);
            if let Ok(contents) = std::fs::read_to_string(&candidate) {
                return Some((candidate, contents));
            }
        }
    }
    None
}

pub fn find_desktop_file_by_id(desktop_id: &str) -> Option<(PathBuf, String)> {
    for dir in application_dirs() {
        let candidate = dir.join(desktop_id);
        if let Ok(contents) = std::fs::read_to_string(&candidate) {
            return Some((candidate, contents));
        }
    }
    None
}

pub fn find_icon_path(icon_name: &str) -> Option<String> {
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
