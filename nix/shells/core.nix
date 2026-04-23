{ pkgs }:

# Focused shell for forge-core Go work.
pkgs.mkShell {
  name = "forge-core-dev";

  packages = with pkgs; [
    go
    gopls
    gotools
    sqlite
    jq
    ripgrep
    fd
    git
  ];

  shellHook = ''
    echo "FORGE core shell. Useful commands:"
    echo "  cd services/core && go build ./..."
    echo "  cd services/core && go test ./..."
    echo "  cd services/core && go vet ./..."
  '';
}
