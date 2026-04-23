{ pkgs }:

# Shell for AI-OS architecture/doctrine work: Go core + Node +
# light docs/diagram tooling. Intentionally does NOT include Tauri
# or heavy desktop deps.
pkgs.mkShell {
  name = "forge-aios-dev";

  packages = with pkgs; [
    go
    gopls
    gotools
    nodejs_20
    sqlite
    jq
    ripgrep
    fd
    git
    graphviz
  ];

  shellHook = ''
    echo "FORGE AI-OS shell. Useful commands:"
    echo "  cd services/core && go test ./..."
    echo "  npm install && npm run build:core"
    echo ""
    echo "Docs: docs/architecture/forge_ai_os.md"
    echo "      docs/architecture/nix_substrate.md"
  '';
}
