{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  nativeBuildInputs = with pkgs; [
    go
    sqlite
    pkg-config
    gcc
  ];

  shellHook = ''
    export CGO_ENABLED=1
  '';
}
