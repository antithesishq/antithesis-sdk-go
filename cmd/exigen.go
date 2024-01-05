package main

import (
  "bytes"
  "flag"
  "fmt"
  "io"
  "os"
  "go/ast"
  "go/parser"
  "go/token"
  "go/types"
  "golang.org/x/tools/go/packages"
  "path"
  "strconv"
  "strings"
  "text/template"
)

type AntExpect struct {
  Text string
  Classname string
  Funcname string
  Receiver string
  Filename string
  Line int
}

type GenInfo struct {
  ExpectedVals []*AntExpect
  ExpectPackageName string
}

var fset *token.FileSet
var imports []string
var expects []*AntExpect
var verbose bool
var func_name string
var receiver string
var package_name string

const ANTILOG_PACKAGE = "github.com/antithesishq/antilog"
const NAME_NOT_AVAILABLE = "anonymous"
const GENERATED_SUFFIX = "_exigen.go"

func analyzed_expr(imports []string, expr ast.Expr) string {
  var expr_name string = ""
  if expr_id, ok := expr.(*ast.Ident); ok {
    expr_name = expr_id.Name
  }
  for _,import_name := range imports {
    if import_name == expr_name {
      return expr_name
    }
  }
  return ""
}

func arg_at_index(args []ast.Expr, idx int) string {
  if args == nil || idx < 0 || len(args) <= idx {
    return NAME_NOT_AVAILABLE
  }
  arg := args[idx]

  var basic_lit *ast.BasicLit
  var basic_lit2 *ast.BasicLit
  var ident *ast.Ident
  var value_spec *ast.ValueSpec
  var ok bool

  // A string literal was provided - nice
  if basic_lit, ok = arg.(*ast.BasicLit); ok {
    text,_ := strconv.Unquote(basic_lit.Value)
    return text
  }

  // Not so nice. 
  // A reference to a const or a var or an indexed value was provided
  //
  // Dig in and see if is resolvable at compile-time
  // When a const is declared in another file, it might not be available here
  if ident, ok = arg.(*ast.Ident); ok {
    if ident.Obj == nil || ident.Obj.Decl == nil {
      return ident.String()
    }
    if value_spec, ok = ident.Obj.Decl.(*ast.ValueSpec); ok {
      values := value_spec.Values
      if len(values) > 0 {
        this_value := values[0]
        if basic_lit2, ok = this_value.(*ast.BasicLit); ok {
          const_text,_ := strconv.Unquote(basic_lit2.Value)
          return const_text
        }
      }
    }
    return ident.String()
  }
  return NAME_NOT_AVAILABLE
}

func node_inspector(x ast.Node) bool {
  var call_expr *ast.CallExpr
  var func_decl *ast.FuncDecl
  var import_spec *ast.ImportSpec
  var fun_expr ast.Expr
  var call_args []ast.Expr
  var ok bool
  var path_name string

  if import_spec, ok = x.(*ast.ImportSpec); ok {
    path_name, _ = strconv.Unquote(import_spec.Path.Value)
    alias := ""
    if import_spec.Name != nil {
      alias = import_spec.Name.Name
    }
    if path_name == ANTILOG_PACKAGE {
      call_qualifier := path.Base(path_name)
      if (alias != "") {
        call_qualifier = alias
      }
      imports = append(imports, call_qualifier)
    }

    return true // you deal with this
  }


  // Track current func_name and receiver (type)
  if func_decl, ok = x.(*ast.FuncDecl); ok {
      func_name = "anonymous"
      if func_ident := func_decl.Name; func_ident != nil {
          func_name = func_ident.Name
      }
      receiver = ""
      if recv := func_decl.Recv; recv != nil {
          if num_fields := recv.NumFields(); num_fields > 0 {
            if field_list := recv.List; field_list != nil {
                if recv_type := field_list[0].Type; recv_type != nil {
                    receiver = types.ExprString(recv_type)
                }
            }
          }
      }
      fmt.Printf(">>       Func: %s %s\n", func_name, receiver)
  }

  if call_expr, ok = x.(*ast.CallExpr); ok {
    fun_expr = call_expr.Fun
    call_args = call_expr.Args

    // TODO Check the behavior when 'dot-import' is used to import 
    // a package directly into a source file's namespace
    //
    // All supported use cases are expected to be identified by
    // ast.SelectorExpr (which specifies an Expression 'X' and a 'Name')
    // For example, the SelectorExpr for strings.HasPrefix() 
    // sel_expr.X is "strings"
    // sel_expr.Name is "HasPrefix"
    if _, ok = fun_expr.(*ast.Ident); ok {
      return true // recurse further
    }

    var sel_expr *ast.SelectorExpr
    if sel_expr, ok = fun_expr.(*ast.SelectorExpr); ok {
      full_position := fset.Position(sel_expr.Pos())
      expr_text := analyzed_expr(imports, sel_expr.X)
      if sel_expr.Sel.Name == "Sometimes" && expr_text != "" {
        test_name := arg_at_index(call_args, 0)
        expect := AntExpect{test_name, package_name, func_name, receiver, full_position.Filename, full_position.Line}
        expects = append(expects, &expect)
      }
      return false
    }
    return false
  }
  return true
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
    for k,v := range pkg.Imports {
      if strings.HasPrefix(k, module_prefix) {
        all_pkg_names = append(all_pkg_names, package_list(module_name, v.PkgPath)...)
      }
    }
  }
  return all_pkg_names
}

func reset_for_module(module_name string) {
  if (verbose) {
    fmt.Printf(">> Module: %s\n", module_name)
  }
  fset = token.NewFileSet()
}

func reset_for_package(packagename string, package_path string) {
  if (verbose) {
    fmt.Printf(">>   Package: %s (%s)\n", packagename, package_path)
  }
  package_name = packagename
}

func reset_for_file(file_path string) {
  if (verbose) {
    fmt.Printf(">>     File: %s\n", file_path)
  }
  imports = []string{}
  func_name = ""
  receiver = ""
}

func investigate_file(file_path string) {
  var file *ast.File
  var err error

  reset_for_file(file_path)
  if file, err = parser.ParseFile(fset, file_path, nil, 0); err != nil {
    panic(err)
  }

  ast.Inspect(file, node_inspector)
}

func usage(gen_name string) string {

  const usage = `
{{.gen_name}} is a go generator to register all existential calls to the Assertions library

{{.gen_name}} [flags] module-name

{{.gen_name}} will scan for all existential calls in the module-name specified
This can be initiated using the 'go generate .' command line tool
after adding a line begininning in column 1 somewhere in the 'main'
package for the specified module-name

Example added to file main.go for module "{{.mod_name}}"

  //go:generate go run ../antilog/cmd/{{.gen_name}}.go -v {{.mod_name}}

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


func main() {

  flag.BoolVar(&verbose, "v", false, "verbose messages to stdout")
  flag.Usage = func() {
    var gen_name = path.Base(os.Args[0])
    usage_text := usage(gen_name)
    var out io.Writer = flag.CommandLine.Output()
    fmt.Fprintf(out, "%s", usage_text)
  }
  flag.Parse()

  var module_name = flag.Arg(0)
  reset_for_module(module_name)
  all_names := package_list(module_name, "")

  var all_pkgs []*packages.Package
  cfg := &packages.Config{
    Mode: packages.NeedModule | packages.NeedCompiledGoFiles | packages.NeedName,
  }

  // Load all the dependent modules we can find
  all_pkgs, _ = packages.Load(cfg, all_names...)
  for _, pkg := range all_pkgs {
    reset_for_package(pkg.Name, pkg.PkgPath)
    for _, file_path := range pkg.CompiledGoFiles {
      base_name := path.Base(file_path)
      if was_generated := strings.HasSuffix(base_name, GENERATED_SUFFIX); !was_generated {
        investigate_file(file_path)
      }
    }
  }

  gen_info := GenInfo{
    ExpectedVals: expects,
    ExpectPackageName: ANTILOG_PACKAGE,
  }

  generate_expects(&gen_info)
}

func expect_output_file() (*os.File, error) {
  from_file := os.Getenv("GOFILE")
  dir_name, file_name := path.Split(from_file)
  ext := path.Ext(file_name)
  if len(ext)> 0 {
    file_name = strings.TrimSuffix(file_name, ext)
  }
  generated_name := fmt.Sprintf("%s%s", file_name, GENERATED_SUFFIX)
  output_file_name := path.Join(dir_name, generated_name)
  fmt.Printf("File is: %q\n", output_file_name)

  var file *os.File
  var err error
  if file, err = os.OpenFile(output_file_name, os.O_RDWR | os.O_CREATE, 0644); err != nil {
    file = nil
  }
  if file != nil {
    if err = file.Truncate(0); err != nil {
      file = nil
    }
  }
  return file, err
}

func generate_expects(gen_info *GenInfo) {
  var tmpl *template.Template
  var err error

  const expector =
`package main

import (
  ant "{{.ExpectPackageName}}"
)

func init() {
   {{- range .ExpectedVals }}
     ant.Expect("{{.Text}}", "{{.Classname}}", "{{.Funcname}}", "{{.Filename}}", {{.Line}})
   {{- end}}
}
`

  if tmpl, err = template.New("expector").Parse(expector); err != nil {
    panic(err)
  }
 
  var out_file io.Writer
  if out_file, err = expect_output_file(); err != nil {
    panic(err)
  }

  if err = tmpl.Execute(out_file, gen_info); err != nil {
    panic(err)
  }
}
