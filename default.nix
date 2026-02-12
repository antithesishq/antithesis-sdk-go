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


  docPages = {
    "assert" = {
      title = "package assert";
      desc = "Docs page for antithesis sdk golang assert package";
    };
    random = {
      title = "package random";
      desc = "Docs page for antithesis sdk golang random package";
    };
    lifecycle = {
      title = "package lifecycle";
      desc = "Docs page for antithesis sdk golang lifecycle package";
    };
  };

  docs = stdenv.mkDerivation {
    name = "go_sdk_docs";
    src = ./.; # TODO: filter
    nativeBuildInputs = [ go pandoc (doc2go.overrideAttrs (old: {
      patches = (old.patches or [ ]) ++ [
        ./doc2go-headers.patch
        ./doc2go-title.patch
        ./doc2go-meta-desc.patch
      ];
      doCheck = false;
    })) ];
    buildPhase = ''
      export HOME=$TMPDIR
      mkdir -p $out/docs
      # TODO: can add `-emded` to generate basic stubs for the docs with no styling to customize our own
      doc2go -home github.com/antithesishq/antithesis-sdk-go -out $out/docs ${lib.concatMapStringsSep " " (p: "./${p}") (lib.attrNames docPages)}

      ${lib.concatMapStringsSep "\n" ({ name, value }:
      let
        title = if value ? title then value.title else "package ${name}";
      in
      ''
        substituteInPlace $out/docs/${name}/index.html --replace-fail "%META_DESCRIPTION%" "${value.desc}"
        substituteInPlace $out/docs/${name}/index.html --replace-fail "%TITLE%" "${title}"
      '') (lib.attrsToList docPages)}

      pandoc --template ${index_template} -o $out/index.html README.md
    '';
  };

  go_sdk = pkgs.buildGoModule {
    pname = "antithesis-go-sdk";
    version = "v0.3.4";

    src = ./.;

    vendorHash = "sha256-+TgeewwzcAgvj0tTcb25pegmJ/IhuQJkXiLrHMfLpDY=";

    meta = {
      description = "Antithesis go sdk.";
      homepage = "https://github.com/antithesishq/antithesis-sdk-go";
      platforms = lib.platforms.linux;
      licenses = lib.licenses.mit;
    };
  };

  go_sdk_no_antithesis = go_sdk.overrideAttrs (old: {
    pname = "antithesis-go-sdk-no-antithesis";
    checkFlags = (old.checkFlags or []) ++ ["-tags=no_antithesis_sdk"];
  });
in

{
  inherit docs go_sdk go_sdk_no_antithesis;
}
