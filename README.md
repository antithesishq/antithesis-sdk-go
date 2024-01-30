
# Antithesis Go SDK

Provides functions for Go programs to configure [Antithesis testing](https://antithesis.com).

## Assertions
Developers can use the functions in the `assert` package to declare properties
of their system for Antithesis to check. This includes conventional assertions,
and also [Sometimes Assertions](https://antithesis.com/docs/best_practices/sometimes_assertions.html)
which can help you assess the quality of your testing or check for unreachable
code.

Visit the Antithesis documentation for more [details about Assertions](https://antithesis.com/docs/using_antithesis/properties.html).

## Lifecycle
Antithesis waits to start injecting faults until the software under test indicates
that it is booted and ready. Use the lifecycle call `SetupComplete()` to indicate
that your system is ready for testing.

### Building

```
CC=clang go build ./assert ./random ./internal ./lifecycle 
```

### Running Benchmarks and Tests

```
CC=clang go test -bench=. ./internal ./assert
```

Results from 30-Jan-2024

```
goos: linux
goarch: amd64
pkg: github.com/antithesishq/antithesis-sdk-go/internal
cpu: Intel(R) Xeon(R) E-2224G CPU @ 3.50GHz
BenchmarkNoEmitWithLocalEmitDisabled-4          43285894                25.65 ns/op
BenchmarkNoEmitWithLocalEmitEnabled-4           34070612                34.24 ns/op
PASS
ok      github.com/antithesishq/antithesis-sdk-go/internal      2.345s
goos: linux
goarch: amd64
pkg: github.com/antithesishq/antithesis-sdk-go/assert
cpu: Intel(R) Xeon(R) E-2224G CPU @ 3.50GHz
BenchmarkAlways-4        1330213               897.3 ns/op
PASS
ok      github.com/antithesishq/antithesis-sdk-go/assert        2.110s
```


### Exigen
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
