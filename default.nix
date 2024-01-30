{pkgs ? import <nixpkgs> {}}:

let
  lib = pkgs.lib;
  doc2go = pkgs.doc2go;

  docs = pkgs.stdenv.mkDerivation {
    name = "build_docs";
    src = ./.; # TODO: filter
    buildInputs = [pkgs.go];
    buildPhase = ''
      export HOME=$TMPDIR
      mkdir -p $out/docs
      ${doc2go}/bin/doc2go -out $out/docs ./assert ./io ./lifecycle
    '';
  };

  go_sdk = pkgs.buildGoModule {
    pname = "antithesis-go-sdk";
    version = "0.0.0";

    src = ./.;

    vendorHash = "sha256-ZgekPCbKEzorA6S7Y9Gp/FX8IvuMzSFpwjZYOtXGNJc=";

    meta = {
      description = "Antithesis go sdk.";
      homepage = "https://github.com/antithesishq/antithesis-sdk-go";
      platforms = lib.platforms.linux;
      licenses = lib.icenses.mit;
    };
  };
in

{
  inherit docs go_sdk;
}