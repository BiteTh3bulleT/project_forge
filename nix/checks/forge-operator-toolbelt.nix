{
  forgeOperatorToolbelt,
  runCommand,
}:

runCommand "forge-operator-toolbelt-check" { } ''
  set -euo pipefail

  toolbelt="${forgeOperatorToolbelt}"

  for bin in \
    foot pcmanfm firefox ollama btop htop sqlitebrowser jq yq rg fd bat eza tree git lazygit \
    curl wget nmap dig ip ss lspci lsusb lsof strace micro hx xarchiver \
    forge-operator-btop forge-operator-lazygit forge-operator-editor forge-operator-ollama-status \
    forge-operator-models forge-operator-core-status forge-operator-core-logs \
    forge-operator-network-diagnostics forge-operator-hardware-diagnostics
  do
    if [ ! -x "$toolbelt/bin/$bin" ]; then
      echo "missing executable in forge-operator-toolbelt closure: $bin" >&2
      exit 1
    fi
  done

  touch "$out"
''
