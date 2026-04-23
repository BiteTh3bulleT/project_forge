{ pkgs }:

# FORGE default dev shell — broad enough for most repo work:
# Go core, Node workspace, sqlite, scripts, and Nix formatter.
# Heavy Tauri/desktop extras live in nix/shells/desktop.nix to keep
# this shell fast to enter on non-desktop machines.
pkgs.mkShell {
  name = "forge-dev";

  packages = with pkgs; [
    # Go core
    go
    gopls
    gotools

    # Node workspace
    nodejs_20

    # Data / scripts
    sqlite
    jq
    ripgrep
    fd
    git

    # Nix tooling
    nixfmt
  ];

  shellHook = ''
    echo "FORGE dev shell (default). Useful commands:"
    echo "  npm install                     # install workspace deps"
    echo "  npm run build:core              # build forge-core"
    echo "  npm run build:desktop           # build desktop UI"
    echo "  cd services/core && go test ./... "
    echo ""
    echo "For desktop/Tauri work: nix develop .#desktop"
    echo "For Go-only core work:  nix develop .#core"
  '';
}
