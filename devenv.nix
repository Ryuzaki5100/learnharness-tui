{ pkgs, ... }:

{
  languages.go.enable = true;

  packages = with pkgs; [
    golangci-lint
    gotools
    gopls
    delve
    just
  ];

  env.CGO_ENABLED = "0";

  scripts.run.exec = "go run ./cmd/lhtui \"$@\"";

  enterTest = ''
    go vet ./...
    go test ./...
  '';
}
