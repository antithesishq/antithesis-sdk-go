package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

const ANTITHESIS_SDK_PACKAGE = "github.com/antithesishq/antithesis-sdk-go/assert"

func usage(gen_name string) string {

	const usage = `
{{.gen_name}} generates code to integrate module references to the Antithesis SDK

{{.gen_name}} [flags] module-name

{{.gen_name}} will scan for all Antithesis assertions used in the module-name specified
This can be initiated using the 'go generate .' command line tool
after adding a line begininning in column 1 somewhere in the 'main'
package for the specified module-name

Example added to file main.go for module "{{.mod_name}}"

  //go:generate go run ../antithesis-sdk-go/antithesis-go-generator/{{.gen_name}}.go -v {{.mod_name}}

Supported flags:
  -v    verbose messages to stdout
  -h    show this help text

`

	usage_vals := make(map[string]interface{})
	usage_vals["gen_name"] = gen_name
	usage_vals["mod_name"] = "github.com/somebody/calctool"

	var tmpl *template.Template
	var err error
	var buf bytes.Buffer

	if tmpl, err = template.New("usage").Parse(usage); err != nil {
		panic(err)
	}

	if err = tmpl.Execute(&buf, usage_vals); err != nil {
		panic(err)
	}
	return buf.String()
}

func package_list(module_name string, root string) []string {
	cfg := &packages.Config{
		Mode: packages.NeedModule | packages.NeedName | packages.NeedImports | packages.NeedDeps,
	}
	var all_pkg_names []string = []string{}
	var all_pkgs []*packages.Package
	var start_name = module_name
	if len(root) > 0 {
		start_name = root
	}
	var module_prefix = module_name + "/"

	all_pkg_names = append(all_pkg_names, start_name)
	all_pkgs, _ = packages.Load(cfg, start_name)
	for _, pkg := range all_pkgs {
		for k, v := range pkg.Imports {
			if strings.HasPrefix(k, module_prefix) {
				all_pkg_names = append(all_pkg_names, package_list(module_name, v.PkgPath)...)
			}
		}
	}
	return all_pkg_names
}

func main() {

    verbose := false
	flag.BoolVar(&verbose, "v", false, "verbose messages to stdout")
	flag.Usage = func() {
		var gen_name = path.Base(os.Args[0])
		usage_text := usage(gen_name)
		var out io.Writer = flag.CommandLine.Output()
		fmt.Fprintf(out, "%s", usage_text)
	}
	flag.Parse()

	var module_name = flag.Arg(0)
    aSI := NewScanningInfo(verbose, module_name)
	all_names := package_list(module_name, "")

	var all_pkgs []*packages.Package
	cfg := &packages.Config{
		Mode: packages.NeedModule | packages.NeedCompiledGoFiles | packages.NeedName,
	}

	// Load all the dependent modules we can find
	all_pkgs, _ = packages.Load(cfg, all_names...)
	for _, pkg := range all_pkgs {
		aSI.reset_for_package(pkg.Name, pkg.PkgPath)
		for _, file_path := range pkg.CompiledGoFiles {
			base_name := path.Base(file_path)
            // dont investigate the init file generated from a previous run
			if was_generated := strings.HasSuffix(base_name, GENERATED_SUFFIX); !was_generated {
                aSI.ScanFile(file_path)
			}
		}
	}
    
    aSI.WriteAssertionCatalog(ANTITHESIS_SDK_PACKAGE)
}
