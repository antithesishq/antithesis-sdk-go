package assertions

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

type GenInfo struct {
	ConstMap          map[string]bool
	logWriter         *common.LogWriter
	AssertPackageName string
	VersionText       string
	CreateDate        string
	ExpectedVals      []*AntExpect
	HasAssertions     bool
}

func IsGeneratedFile(file_name string) bool {
	base_name := filepath.Base(file_name)
	return strings.HasSuffix(base_name, common.GENERATED_SUFFIX)
}

// --------------------------------------------------------------------------------
// dest_path is always structured as a file path
// that ends with the module name being instrumented
// the module name has been mangled by replacing path
// separators "/" and "\" with "_V_"
//
// Example:
// If the 'dest_path' argument is:
//
//	"/home/johndoe/output/customer/nice.example.com_V_my_V_thing"
//
// Then 'dest_path' was composed of
//
//	customerOutputDir: "/home/johndoe/output/customer"
//	moduleName: "nice.example.com/my/thing"
//
// This gets split to:
//
//	dir_name: "/home/johndoe/output/customer"
//	file_name: "nice.example.com_V_my_V_thing"
//	ext: ""
//
// And finally:
//
//	generated_name: "nice.example.com_V_my_V_thing_antithesis_catalog.go"
//	output_file_name: "/home/johndoe/output/customer/nice.example.com_V_my_V_thing_antithesis_catalog.go"
//
// --------------------------------------------------------------------------------
func expectOutputFile(dest_path string, logWriter *common.LogWriter) (*os.File, error) {
	dir_name, file_name := path.Split(dest_path)
	generated_name := fmt.Sprintf("%s%s", file_name, common.GENERATED_SUFFIX)
	output_file_name := path.Join(dir_name, generated_name)

	var file *os.File
	var err error
	if file, err = os.OpenFile(output_file_name, os.O_RDWR|os.O_CREATE, 0644); err != nil {
		file = nil
	}
	if file != nil {
		if err = file.Truncate(0); err != nil {
			file = nil
		}
	}
	if err == nil {
		logWriter.Printf("Assertion Catalog: %q\n", output_file_name)
	} else {
		logWriter.Printf("Unable to generate Assertion Catalog: %q\n", output_file_name)
	}
	return file, err
}

func assertionNameRepr(s string) string {
	if s == "Reachable" || s == "Unreachable" {
		return fmt.Sprintf("%s(message, details)", s)
	}
	return fmt.Sprintf("%s(cond, message, details)", s)
}

func hitRepr(b bool) string {
	if !b {
		return "notHit"
	}
	return "wasHit"
}

func condRepr(b bool) string {
	if b {
		return "condTrue"
	}
	return "condFalse"
}

func mustHitRepr(b bool) string {
	if b {
		return "mustBeHit"
	}
	return "optionallyHit"
}

func assertTypeRepr(s string) string {
	reprText := "reachabilityTest"

	switch s {
	case "always":
		reprText = "universalTest"
	case "sometimes":
		reprText = "existentialTest"
	case "reachability":
		reprText = "reachabilityTest"
	}
	return reprText
}

func usesConst(cm map[string]bool, c string) bool {
	return cm[c]
}

func GenerateAssertionsCatalog(moduleName string, genInfo *GenInfo) {
	var tmpl *template.Template
	var err error

	tmpl = template.New("expector")

	tmpl = tmpl.Funcs(template.FuncMap{
		"hitRepr":           hitRepr,
		"condRepr":          condRepr,
		"mustHitRepr":       mustHitRepr,
		"assertTypeRepr":    assertTypeRepr,
		"assertionNameRepr": assertionNameRepr,
		"usesConst":         usesConst,
	})

	if tmpl, err = tmpl.Parse(getExpectorText()); err != nil {
		panic(err)
	}

	var outFile io.Writer
	if outFile, err = expectOutputFile(moduleName, genInfo.logWriter); err != nil {
		panic(err)
	}

	if err = tmpl.Execute(outFile, genInfo); err != nil {
		panic(err)
	}
}

func getExpectorText() string {
	const text = `package main

// ----------------------------------------------------
// {{.VersionText}}
// 
// Assertion Catalog - Do not modify
// 
// Generated on {{.CreateDate}} 
// ----------------------------------------------------

{{if .HasAssertions -}}import "{{.AssertPackageName}}"{{- end}}

{{if .HasAssertions -}}
func init() {

{{if usesConst .ConstMap "condFalse"}}  const condFalse = false{{- end}}
{{if usesConst .ConstMap "condTrue"}}  const condTrue = true {{- end}}
  const wasHit = true
{{if usesConst .ConstMap "notHit"}}  const notHit = !wasHit {{- end}}
{{if usesConst .ConstMap "mustBeHit"}}  const mustBeHit = true {{- end}}
{{if usesConst .ConstMap "optionallyHit"}}  const optionallyHit = false {{- end}}
{{if usesConst .ConstMap "expectingTrue"}}  const expectingTrue = true {{- end}}
{{if usesConst .ConstMap "expectingFalse"}} const expectingFalse = false {{- end}}
{{if usesConst .ConstMap "universalTest"}}  const universalTest = "every" {{- end}}
{{if usesConst .ConstMap "existentialTest"}}  const existentialTest = "some" {{- end}}
{{if usesConst .ConstMap "reachabilityTest"}}  const reachabilityTest = "none" {{- end}}

  var noDetails map[string]any = nil
	
	{{- range .ExpectedVals }}
	{{- $cond := condRepr .AssertionFuncInfo.Condition -}}
	{{- $didHit := hitRepr false -}}
	{{- $mustHit := mustHitRepr .AssertionFuncInfo.MustHit -}}
	{{- $assertionName := assertionNameRepr .Assertion -}}
	{{- $assertType := assertTypeRepr .AssertionFuncInfo.AssertType -}}

  // {{$assertionName}}
  assert.AssertRaw({{$cond}}, "{{.Message}}", noDetails, "{{.Classname}}", "{{.Funcname}}", "{{.Filename}}", {{.Line}}, {{$didHit}}, {{$mustHit}}, {{$assertType}}, "{{.Assertion}}", "{{.Message}}")
	{{- end}}
}
{{- end}}
`

	return text
}
