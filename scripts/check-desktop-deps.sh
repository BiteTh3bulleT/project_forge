#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Linux" ]]; then
  exit 0
fi

if ! command -v pkg-config >/dev/null 2>&1; then
  cat <<'EOF'
[forge desktop] Missing required tool: pkg-config
Install pkg-config, then rerun `npm run desktop`.
EOF
  exit 1
fi

missing=()
for dep in webkit2gtk-4.1 javascriptcoregtk-4.1 gtk+-3.0; do
  if ! pkg-config --exists "$dep"; then
    missing+=("$dep")
  fi
done

if [[ ${#missing[@]} -eq 0 ]]; then
  exit 0
fi

echo "[forge desktop] Missing required Linux desktop libraries:"
for dep in "${missing[@]}"; do
  echo "  - $dep"
done
echo

os_id=""
if [[ -r /etc/os-release ]]; then
  # shellcheck disable=SC1091
  source /etc/os-release
  os_id="${ID:-}"
fi

case "$os_id" in
  opensuse*|sles|suse)
    cat <<'EOF'
OpenSUSE install hint:
  sudo zypper install -y webkitgtk3-devel gtk3-devel

If package names differ by repo snapshot, locate providers:
  zypper search --provides 'pkgconfig(webkit2gtk-4.1)'
  zypper search --provides 'pkgconfig(javascriptcoregtk-4.1)'
EOF
    ;;
  ubuntu|debian|linuxmint|pop)
    cat <<'EOF'
Debian/Ubuntu install hint:
  sudo apt-get update
  sudo apt-get install -y libwebkit2gtk-4.1-dev libjavascriptcoregtk-4.1-dev libgtk-3-dev
EOF
    ;;
  fedora)
    cat <<'EOF'
Fedora install hint:
  sudo dnf install -y webkit2gtk4.1-devel gtk3-devel
EOF
    ;;
  arch)
    cat <<'EOF'
Arch install hint:
  sudo pacman -S --needed webkit2gtk gtk3
EOF
    ;;
  *)
    cat <<'EOF'
Install packages that provide these pkg-config entries:
  pkgconfig(webkit2gtk-4.1)
  pkgconfig(javascriptcoregtk-4.1)
  pkgconfig(gtk+-3.0)
EOF
    ;;
esac

exit 1
