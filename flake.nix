{
  description = "lhtui: read a LearnHarness learning record and run opencode study sessions";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      version = "0.1.1";

      forAll = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      package =
        pkgs:
        pkgs.buildGoModule {
          pname = "lhtui";
          inherit version;
          src = ./.;
          # Dependencies are vendored, so the build needs no network.
          vendorHash = null;
          subPackages = [ "cmd/lhtui" ];
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];
          meta = {
            description = "Terminal companion for LearnHarness learning records";
            homepage = "https://github.com/Ryuzaki5100/learnharness-tui";
            license = pkgs.lib.licenses.agpl3Only;
            mainProgram = "lhtui";
          };
        };
    in
    {
      packages = forAll (pkgs: {
        default = package pkgs;
        lhtui = package pkgs;
      });

      apps = forAll (pkgs: {
        default = {
          type = "app";
          program = nixpkgs.lib.getExe self.packages.${pkgs.stdenv.hostPlatform.system}.default;
        };
      });

      devShells = forAll (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            golangci-lint
            just
          ];
          env.CGO_ENABLED = "0";
        };
      });
    };
}
