{
  pkgs ? import <nixpkgs> {}, 
  doc2go ? pkgs.doc2go,
  lib ? pkgs.lib,
  go ? pkgs.go,
  stdenv ? pkgs.stdenv,
}:

let
  docs = stdenv.mkDerivation {
    name = "build_docs";
    src = ./.; # TODO: filter
    buildInputs = [go];
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
      licenses = lib.licenses.mit;
    };
  };
in

{
  inherit docs go_sdk;
}