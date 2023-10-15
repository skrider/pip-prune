{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
    buildInputs = with pkgs; [
        gnumake
        go
        terraform
    ];
    shellHook = ''
echo nixpgks version: ${pkgs.lib.version}
export NIX_LD_LIB=${pkgs.python310.stdenv.cc.cc.lib}/lib
    '';
}

