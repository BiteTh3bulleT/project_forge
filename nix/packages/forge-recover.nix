{ lib, buildGoModule, sqlite }:

# FORGE offline recovery utility — restores a full FORGE backup through the
# daemon-stopped whole-store recovery boundary.
buildGoModule rec {
  pname = "forge-recover";
  version = "0.1.0";

  # Source is the whole repo; modRoot scopes the Go module to services/core.
  src = lib.cleanSource ../..;
  modRoot = "services/core";

  # This shares services/core/go.sum with the core daemon and should be kept
  # in sync when dependencies change.
  vendorHash = "sha256-JjVms5MeQbCt69/H0b9L0SVO8KZPPNdv2Kp3EnivXug=";

  # Runtime has no external Go runtime dependency beyond Go-standard and project
  # dependencies, but sqlite is kept for support tools in transitive callers.
  buildInputs = [ sqlite ];

  # Keep tests off for package builds to keep local Nix usage fast.
  doCheck = false;

  # `cmd/forge-recover` is a standalone, daemon-stopped restore utility.
  subPackages = [ "cmd/forge-recover" ];

  ldflags = [ "-s" "-w" ];

  meta = with lib; {
    description = "FORGE offline whole-store recovery utility for daemon-stopped restores";
    license = licenses.mit;
    mainProgram = "forge-recover";
    platforms = platforms.unix;
  };
}
