{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  nativeBuildInputs = with pkgs; [
    (sqlite.override {
      interactive = true;
    })
    duckdb
    pkg-config
    gcc
    openssl
    curl
  ];

  shellHook = ''
    export CGO_ENABLED=1
  '';
}
