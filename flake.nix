{
  description = "Dev environment for Helmper (Go) Helm chart and image management tool";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;

        # Get git commit and build date dynamically (similar to Makefile)
        gitCommit = lib.substring 0 7 (self.rev or self.dirtyRev or "unknown");
        buildDate = builtins.readFile (
          pkgs.runCommand "build-date" { } ''
            date -u +%Y-%m-%dT%H:%M:%SZ > $out
          ''
        );

        helmperPkg = pkgs.buildGoModule {
          pname = "helmper";
          version = "dev";
          src = ./.;

          # Build from cmd/helmper directory like Makefile
          subPackages = [ "cmd/helmper" ];

          # Has external dependencies (Helm SDK, Cosign, etc.)
          # To update: set to null, run `nix build`, then copy the got hash
          vendorHash = "sha256-OqEVEfN3uodltMZoclYBVXmE70s1oALZKI4rslUKbX8=";

          # Skip tests that require network access (Nix sandbox blocks external calls)
          doCheck = false;

          env = {
            CGO_ENABLED = "0";
          };

          # Match Makefile LD_FLAGS behavior
          ldflags = [
            "-s"
            "-w"
            "-X github.com/ChristofferNissen/helmper/internal.version=dev"
            "-X github.com/ChristofferNissen/helmper/internal.commit=${gitCommit}"
            "-X github.com/ChristofferNissen/helmper/internal.date=${buildDate}"
          ];
        };

        baseTools = with pkgs; [
          go
          gopls
          golangci-lint
          delve
          git
          jq
          nixpkgs-fmt
        ];

        extraTools = lib.optionals (lib.hasAttr "staticcheck" pkgs) [ pkgs.staticcheck ];
      in
      {
        devShells.default = pkgs.mkShell {
          name = "helmper-go-devshell";
          buildInputs = baseTools ++ extraTools;
          shellHook = ''
            export CGO_ENABLED=0
            export GOFLAGS="-mod=readonly"
            echo "⚓ Helmper dev environment ready."
            echo "Go: $(go version)"
            echo "Run: golangci-lint run  |  dlv debug  |  go test -v ./..."
          '';
        };

        packages.helmper = helmperPkg;
        packages.default = helmperPkg;

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
