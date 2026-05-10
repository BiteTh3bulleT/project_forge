{ runCommandNoCC }:

runCommandNoCC "forge-core-bind-host-check"
  {
    src = ../..;
  }
  ''
    module="$src/nix/nixos/modules/forge-services.nix"
    config="$src/services/core/internal/config/config.go"
    main="$src/services/core/main.go"
    compose="$src/docker-compose.yml"
    dockerfile="$src/services/core/Dockerfile"

    grep -F 'default = "127.0.0.1";' "$module"
    grep -F 'FORGE_CORE_BIND_HOST = toString cfg.bindHost;' "$module"
    grep -F 'envStringDefault("FORGE_CORE_BIND_HOST", "127.0.0.1")' "$config"
    grep -F 'net.JoinHostPort(host, strconv.Itoa(cfg.Port))' "$main"
    if grep -E 'addr := ":".*strconv\.Itoa\(cfg\.Port\)' "$main"; then
      echo "forge-core must not bind all interfaces by default" >&2
      exit 1
    fi
    grep -F 'FORGE_CORE_BIND_HOST: "''${FORGE_CORE_BIND_HOST:-0.0.0.0}"' "$compose"
    grep -F 'FORGE_CORE_BIND_HOST=0.0.0.0' "$dockerfile"

    touch "$out"
  ''
