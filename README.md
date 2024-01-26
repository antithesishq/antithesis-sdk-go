
# antithesis-sdk-go

Provides functions enabling Go programs to include non-fatal Assertions and structured IO.

### Running Benchmarks and Tests

```
CC=clang go test -bench=. ./assert 
```

Results from 26-Jan-2024 v0.1.13

```
goos: linux
goarch: amd64
pkg: github.com/antithesishq/antithesis-sdk-go/assert
cpu: AMD Ryzen 9 5900X 12-Core Processor            
BenchmarkIsTrue-24                              374603544                3.178 ns/op
BenchmarkCanEmitWithLocalEmitDisabled-24        684141135                1.692 ns/op
BenchmarkNoEmitWithLocalEmitDisabled-24         690226068                1.686 ns/op
BenchmarkCanEmitWithLocalEmitEnabled-24         790518410                1.584 ns/op
BenchmarkNoEmitWithLocalEmitEnabled-24          748888886                1.553 ns/op
PASS
ok      github.com/antithesishq/antithesis-sdk-go/assert        6.927s
```

### Building

```
CC=clang go build ./assert ./io ./local ./internal ./lifecycle 
```


### Using exigen
The antithesis-sdk-go module also contains the exigen command, used to 
identify assertions that were added to a Go module, and to generate
corresponding function calls in an init() function to register these
assertions.  exigen should be installed in a dev environment.  

```
go get github.com/antithesishq/antithesis-sdk-go/tools/exigen
go install github.com/antithesishq/antithesis-sdk-go/tools/exigen
```

This will install exigen to the $GOPATH/bin folder so it can be used
to register all antithesis assertions used.  Prior to every `go build` step,
use `go generate` so that any and all `//go:generate ...` directives found in
the source being built, can be evaluated and executed.


Add a Go directive somewhere in the main package for
a module that has Antithesis assertions added to it.  The
general form is like this:

`//go:generate exigen my-module-name`

Example:

`//go:generate exigen antithesis.com/go/sample-project`

A good place to add this directive is in the top-level
driver for an executable (often this is `main.go`)

With the directive in place, add a `go generate` build step, prior to 
the typical compile/link build step `go build <path>`.  The `go generate`
step will scan Golang source code files.  Any generate commands 
encountered will be executed as part of the `go generate`
process.  When exigen runs, it will scan for packages in the specified
module indicated in the corresponding //go:generate directive.  

All of the files in each package of the module will be scanned for 
assertions, and result in a regsitration call to be created
for each assertion that was scanned.  These registrations 
will be written to a new file whose name is derived from the name of the file that
contains the //go:generate directive.  

If the file containing the //go:generate directive is `main.go`  then the 
derived file will be `main_exigen.go` and will include an init() function 
containing the registration calls that were created.

After running `go generate <path>` run `go build <path>` and the
module source, along with the newly generated `main_exigen.go` file
will be compiled and linked.

### Building antithesis-sdk-go
When building antithesis-sdk-go, make sure to specify clang and to set CGO_ENABLED
  
Example:

    CC=clang CGO_ENABLED=1 go build


