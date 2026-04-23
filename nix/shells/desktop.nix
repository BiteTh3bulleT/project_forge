{ pkgs, lib, stdenv }:

# Shell for Tauri/desktop development.
# On Linux we include the WebKit/GTK stack required by Tauri.
# On Darwin we rely on system frameworks (installed separately).
let
  linuxDeps = with pkgs; [
    pkg-config
    openssl
    glib
    gtk3
    libsoup_3
    webkitgtk_4_1
    librsvg
    libayatana-appindicator
    patchelf
  ];

  commonDeps = with pkgs; [
    nodejs_20
    rustc
    cargo
    rustfmt
    clippy
    rust-analyzer
    git
    jq
    ripgrep
    fd
  ];
in
pkgs.mkShell {
  name = "forge-desktop-dev";

  packages = commonDeps ++ lib.optionals stdenv.isLinux linuxDeps;

  shellHook = ''
    echo "FORGE desktop shell. Useful commands:"
    echo "  npm install"
    echo "  npm -w @forge/desktop run dev"
    echo "  npm -w @forge/desktop run build"
    echo "  npm -w @forge/desktop run tauri -- dev"
    echo ""
    ${lib.optionalString stdenv.isLinux ''
      echo "Note: WebKit/GTK deps provided by this Nix shell."
    ''}
    ${lib.optionalString stdenv.isDarwin ''
      echo "Note: On macOS, Tauri uses system WebKit — ensure Xcode CLT is installed."
    ''}
  '';
}
