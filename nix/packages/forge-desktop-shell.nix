{
  lib,
  stdenv,
  rustPlatform,
  buildNpmPackage,
  cargo-tauri,
  pkg-config,
  wrapGAppsHook4,
  dbus,
  openssl,
  glib,
  glib-networking,
  gtk3,
  libsoup_3,
  webkitgtk_4_1,
  librsvg,
  libayatana-appindicator,
}:

let
  version = "0.1.0";
  repoSrc = lib.cleanSourceWith {
    src = ../..;
    filter =
      path: type:
      let
        rel = lib.removePrefix ((toString ../..) + "/") (toString path);
        base = baseNameOf path;
      in
      !(lib.hasPrefix "node_modules/" rel)
      && !(lib.hasInfix "/node_modules/" rel)
      && !(lib.hasPrefix "apps/desktop/dist" rel)
      && !(lib.hasPrefix "apps/desktop/src-tauri/target/" rel)
      && !(lib.hasPrefix ".git/" rel)
      && base != "result";
  };

  desktopFrontend = buildNpmPackage {
    pname = "forge-desktop-frontend";
    inherit version;

    src = repoSrc;
    npmDepsHash = "sha256-Fpz1YjmIpjeLya7VEmCMY/eD+BmsWQz2bAPy2HvDoYs=";
    npmBuildScript = "build:desktop";
    VITE_FORGE_BOOT_LOGIN = "true";
    VITE_FORGE_LOGIN_USER = "operator";
    VITE_FORGE_LOGIN_PASSWORD = "forge";

    installPhase = ''
      runHook preInstall
      mkdir -p "$out"
      cp -r apps/desktop/dist "$out/dist"
      runHook postInstall
    '';
  };
in
rustPlatform.buildRustPackage rec {
  pname = "forge-desktop-shell";
  inherit version;

  src = repoSrc;
  cargoRoot = "apps/desktop/src-tauri";
  buildAndTestSubdir = cargoRoot;
  cargoHash = "sha256-QVzIw5HZswlYcQzLFcMh6WNTY2QldMEn4j7gB2+XSro=";

  nativeBuildInputs = [
    cargo-tauri.hook
    pkg-config
  ]
  ++ lib.optionals stdenv.hostPlatform.isLinux [
    wrapGAppsHook4
  ];

  buildInputs = lib.optionals stdenv.hostPlatform.isLinux [
    dbus
    openssl
    glib
    glib-networking
    gtk3
    libsoup_3
    webkitgtk_4_1
    librsvg
    libayatana-appindicator
  ];

  tauriNixConfig = builtins.toJSON {
    build = {
      beforeBuildCommand = "";
      frontendDist = "../dist";
    };
  };

  preBuild = ''
    mkdir -p apps/desktop/dist
    cp -R ${desktopFrontend}/dist/. apps/desktop/dist/
    printf '%s' '${tauriNixConfig}' > apps/desktop/src-tauri/tauri.nix.conf.json
    tauriBuildFlags+=(
      "--config"
      "tauri.nix.conf.json"
    )
  '';

  doCheck = false;

  postInstall = ''
    runHook prePostInstall

    tauriBinary="$out/bin/forge_desktop"
    if [ ! -x "$tauriBinary" ]; then
      echo "expected Tauri binary was not installed at $tauriBinary" >&2
      echo "installed files:" >&2
      find "$out" -maxdepth 4 \( -type f -o -type l \) >&2
      exit 1
    fi

    printf '%s\n' \
      '#!@shell@' \
      'set -euo pipefail' \
      "" \
      'export FORGE_SHELL_SESSION_ENABLED="''${FORGE_SHELL_SESSION_ENABLED:-true}"' \
      'export FORGE_SHELL_MODE="''${FORGE_SHELL_MODE:-fullscreen-shell}"' \
      'export FORGE_CORE_URL="''${FORGE_CORE_URL:-http://127.0.0.1:18492}"' \
      'export VITE_FORGE_API_URL="''${VITE_FORGE_API_URL:-$FORGE_CORE_URL}"' \
      'export FORGE_SHELL_SAFE_MODE="''${FORGE_SHELL_SAFE_MODE:-true}"' \
      'export FORGE_SHELL_FULLSCREEN="''${FORGE_SHELL_FULLSCREEN:-true}"' \
      'export FORGE_SHELL_HOST_MUTATION="''${FORGE_SHELL_HOST_MUTATION:-false}"' \
      'export FORGE_SHELL_DIRECT_SYSTEM_CONTROL="''${FORGE_SHELL_DIRECT_SYSTEM_CONTROL:-false}"' \
      'export FORGE_SHELL_MODEL_MUTATION="''${FORGE_SHELL_MODEL_MUTATION:-false}"' \
      'export FORGE_SHELL_SEMANTIC_MEMORY_WRITE="''${FORGE_SHELL_SEMANTIC_MEMORY_WRITE:-false}"' \
      'export FORGE_SHELL_FORGE_K_LIVE_AUTHORITY="''${FORGE_SHELL_FORGE_K_LIVE_AUTHORITY:-false}"' \
      "" \
      'if [ -n "''${FORGE_DESKTOP_SHELL_BINARY:-}" ]; then' \
      '  if [ -x "$FORGE_DESKTOP_SHELL_BINARY" ]; then' \
      '    exec "$FORGE_DESKTOP_SHELL_BINARY" "$@"' \
      '  fi' \
      '  echo "FORGE_DESKTOP_SHELL_BINARY is set but is not executable: $FORGE_DESKTOP_SHELL_BINARY" >&2' \
      '  exit 1' \
      'fi' \
      "" \
      'exec "@tauriBinary@" "$@"' \
      > "$out/bin/forge-desktop-shell"
    substituteInPlace "$out/bin/forge-desktop-shell" \
      --replace-fail "@shell@" "${stdenv.shell}" \
      --replace-fail "@tauriBinary@" "$tauriBinary"
    chmod +x "$out/bin/forge-desktop-shell"

    runHook postPostInstall
  '';

  passthru = {
    containsTauriBinary = true;
  };

  meta = with lib; {
    description = "FORGE Tauri desktop shell packaged for the graphical shell session";
    license = licenses.mit;
    mainProgram = "forge-desktop-shell";
    platforms = platforms.linux;
  };
}
