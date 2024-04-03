{
  pkgs ? import <nixpkgs> {}, 
  doc2go ? pkgs.doc2go,
  lib ? pkgs.lib,
  go ? pkgs.go,
  pandoc ? pkgs.pandoc,
  stdenv ? pkgs.stdenv,
}:

let
  index_template = pkgs.writeText "doc_template.html" ''
    <!doctype html>
    <html lang="en">
      <head>
        <meta charset="utf-8">
      </head>
      <body>
      $body$

      <script>
        function replaceWithLink(packageName) {
          const id = '#package-' + packageName
          const selector = id + ' code'
          console.log(selector)
          const codeElem = document.querySelector(selector); 
          const linkElem = document.createElement("a");
          const linkText = document.createTextNode(packageName)
          linkElem.appendChild(linkText);
    
          const linkUrl = './docs/' + packageName + '/'
    
          linkElem.href = linkUrl
          codeElem.replaceWith(linkElem)
        }
        replaceWithLink('assert')
        replaceWithLink('random')
        replaceWithLink('lifecycle')
      </script>
      
      </body>
    </html>
  '';


  docs = stdenv.mkDerivation {
    name = "go_sdk_docs";
    src = ./.; # TODO: filter
    nativeBuildInputs = [ go pandoc doc2go ];
    buildPhase = ''
      export HOME=$TMPDIR
      mkdir -p $out/docs
      # TODO: can add `-emded` to generate basic stubs for the docs with no styling to customize our own
      doc2go -home github.com/antithesishq/antithesis-sdk-go -out $out/docs ./assert ./random ./lifecycle
      pandoc --template ${index_template} -o $out/index.html README.md
    '';
  };

  go_sdk = pkgs.buildGoModule {
    pname = "antithesis-go-sdk";
    version = "v0.3.3";

    src = ./.;

    vendorHash = "sha256-+TgeewwzcAgvj0tTcb25pegmJ/IhuQJkXiLrHMfLpDY=";

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
