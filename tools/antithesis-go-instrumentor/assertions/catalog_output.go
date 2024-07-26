package assertions

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/antithesishq/antithesis-sdk-go/assert"
	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

type GenInfo struct {
	ConstMap            map[string]bool
	logWriter           *common.LogWriter
	AssertPackageName   string
	VersionText         string
	CreateDate          string
	ExpectedVals        []*AntExpect
	NumericGuidanceVals []*AntGuidance
	BooleanGuidanceVals []*AntGuidance
	HasAssertions       bool
	HasNumericGuidance  bool
	HasBooleanGuidance  bool
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

func numericGuidanceNameRepr(s string) string {
	return fmt.Sprintf("%s(left, right, message, details)", s)
}

func booleanGuidanceNameRepr(s string) string {
	return fmt.Sprintf("%s(pairs, message, details)", s)
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

func textRepr(s string) string {
	return strconv.Quote(s)
}

func guidanceFnRepr(n assert.GuidanceFnType) string {
	gp := ""
	switch n {
	case assert.GuidanceFnMaximize:
		gp = "assert.GuidanceFnMaximize"
	case assert.GuidanceFnMinimize:
		gp = "assert.GuidanceFnMinimize"
	// case assert.GuidanceFnExplore:
	// 	gp = "assert.GuidanceFnExplore"
	case assert.GuidanceFnWantAll:
		gp = "assert.GuidanceFnWantAll"
	case assert.GuidanceFnWantNone:
		gp = "assert.GuidanceFnWantNone"
	}
	return gp
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
		"hitRepr":                 hitRepr,
		"condRepr":                condRepr,
		"mustHitRepr":             mustHitRepr,
		"assertTypeRepr":          assertTypeRepr,
		"assertionNameRepr":       assertionNameRepr,
		"usesConst":               usesConst,
		"textRepr":                textRepr,
		"numericGuidanceNameRepr": numericGuidanceNameRepr,
		"booleanGuidanceNameRepr": booleanGuidanceNameRepr,
		"guidanceFnRepr":          guidanceFnRepr,
	})

	all_template_text := getExpectorText() + getNumericGuidanceText() + getBooleanGuidanceText()
	if tmpl, err = tmpl.Parse(all_template_text); err != nil {
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
{{if usesConst .ConstMap "universalTest"}}  const universalTest = "always" {{- end}}
{{if usesConst .ConstMap "existentialTest"}}  const existentialTest = "sometimes" {{- end}}
{{if usesConst .ConstMap "reachabilityTest"}}  const reachabilityTest = "reachability" {{- end}}

  var noDetails map[string]any = nil
	
	{{- range .ExpectedVals }}
	{{- $cond := condRepr .AssertionFuncInfo.Condition -}}
	{{- $didHit := hitRepr false -}}
	{{- $mustHit := mustHitRepr .AssertionFuncInfo.MustHit -}}
	{{- $assertionName := assertionNameRepr .Assertion -}}
	{{- $assertType := assertTypeRepr .AssertionFuncInfo.AssertType -}}
	{{- $message := textRepr .Message -}}
	{{- $classname := textRepr .Classname -}}
	{{- $funcname := textRepr .Funcname -}}
	{{- $filename := textRepr .Filename -}}
	{{- $displayname := textRepr .Assertion}}

  // {{$assertionName}}
  assert.AssertRaw({{$cond}}, {{$message}}, noDetails, {{$classname}}, {{$funcname}}, {{$filename}}, {{.Line}}, {{$didHit}}, {{$mustHit}}, {{$assertType}}, {{$displayname}}, {{$message}})
	{{- end}}
}
{{- end}}
`

	return text
}

func getNumericGuidanceText() string {
	const text = `

{{if .HasNumericGuidance -}}
func init() {

  const notHit = false
  const left = 0
  const right = 0

  {{- range .NumericGuidanceVals }}
  {{- $guidanceName := numericGuidanceNameRepr .Assertion -}}
	{{- $message := textRepr .Message -}}
	{{- $classname := textRepr .Classname -}}
	{{- $funcname := textRepr .Funcname -}}
	{{- $filename := textRepr .Filename -}}
  {{- $guidanceFn := guidanceFnRepr .GuidanceFuncInfo.GuidanceFn}}

  // {{$guidanceName}}
  assert.NumericGuidanceRaw(left, right, {{$message}}, {{$message}}, {{$classname}}, {{$funcname}}, {{$filename}}, {{.Line}}, {{$guidanceFn}}, notHit)
  {{- end}}
}
{{- end}}
`

	return text
}

func getBooleanGuidanceText() string {
	const text = `

{{if .HasBooleanGuidance -}}
func init() {

  const notHit = false
  var named_bools = []assert.NamedBool{}

  {{- range .BooleanGuidanceVals }}
  {{- $guidanceName := booleanGuidanceNameRepr .Assertion -}}
	{{- $message := textRepr .Message -}}
	{{- $classname := textRepr .Classname -}}
	{{- $funcname := textRepr .Funcname -}}
	{{- $filename := textRepr .Filename -}}
  {{- $guidanceFn := guidanceFnRepr .GuidanceFuncInfo.GuidanceFn}}

  // {{$guidanceName}}
  assert.BooleanGuidanceRaw(named_bools, {{$message}}, {{$message}}, {{$classname}}, {{$funcname}}, {{$filename}}, {{.Line}}, {{$guidanceFn}}, notHit)
  {{- end}}
}
{{- end}}
`

	return text
}
