{ lib, buildGoModule, sqlite }:

# FORGE core service — Go module at services/core.
#
# vendorHash was captured on x86_64-linux via `nix build .#forge-core`.
# If services/core/go.sum changes, Nix will refuse to build and print
# the new expected hash — update vendorHash accordingly.
buildGoModule rec {
  pname = "forge-core";
  version = "0.1.0";

  # Source is the whole repo; modRoot scopes the Go build to services/core.
  src = lib.cleanSource ../..;
  modRoot = "services/core";

  vendorHash = "sha256-ycmhyDcdzpcyK4V0SaTdM0ClyzCJHKIirXNK4uO12dM=";

  # Core imports modernc.org/sqlite (pure Go). sqlite CLI is handy at
  # runtime for inspection/tooling but not strictly required.
  buildInputs = [ sqlite ];

  # The go-tests check runs tests separately; keep the package build lean.
  doCheck = false;

  # services/core has a single main package at its root.
  subPackages = [ "." ];

  ldflags = [ "-s" "-w" ];

  meta = with lib; {
    description = "FORGE AI-OS core service (HTTP API, jobs, approvals, context)";
    license = licenses.mit;
    mainProgram = "core";
    platforms = platforms.unix;
  };
}
