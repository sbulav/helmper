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

        helmperPkg = pkgs.buildGoModule {
          pname = "helmper";
          version = "0.1.0";
          src = ./.;

          # Has external dependencies (Helm SDK, Cosign, etc.)
          # To update: set to null, run `nix build`, then copy the got hash
          vendorHash = "sha256-/BwXWvumPR9j/hCqoGX5xIYdjQvZ7DyxgNqla9bXOBQ=";

          # Skip tests that require network access (Nix sandbox blocks external calls)
          doCheck = false;

          env = {
            CGO_ENABLED = "0";
          };
          ldflags = [
            "-s"
            "-w"
            "-X github.com/ChristofferNissen/helmper/internal.version=0.1.0"
            "-X github.com/ChristofferNissen/helmper/internal.commit=dev"
            "-X github.com/ChristofferNissen/helmper/internal.date=unknown"
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
