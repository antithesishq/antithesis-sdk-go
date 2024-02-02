
# Antithesis Go SDK

Provides functions for Go programs to configure the [Antithesis](https://antithesis.com) platform. Functionality is grouped into the packages `assert` for Assertions, `random` for Antithesis input, and `lifecycle` for controlling Antithesis simulation.

For general usage guidance see [Antithesis SDK Documentation](https://antithesis.com/docs/using_antithesis/sdk/overview.html)

## package `assert`
Developers can use the functions in the `assert` package to declare properties of their system for Antithesis to check. This includes conventional assertions, and also [Sometimes Assertions](https://antithesis.com/docs/best_practices/sometimes_assertions.html) which can help you assess the quality of your testing or check for unreachable code.

Visit the Antithesis documentation for more [details about Assertions](https://antithesis.com/docs/using_antithesis/properties.html).

## package `random`
Developers can get input from the Antithesis platform by calling functions from the `random` package. Getting input from Antithesis allows it to guide your workload to find more bugs faster. For information on how this works, and should be used, see the documentation on [workload basics](https://antithesis.com/docs/getting_started/workload.html)

## package `lifecycle`
Antithesis waits to start injecting faults until the software under test indicates
that it is booted and ready. Use the lifecycle call `SetupComplete()` to indicate
that your system is ready for testing.

## Assertion Indexer
If an Assertions such as `assert.Always()` is not reached during a test, Antithesis will raise a warning. In order to warn about these unseen calls, Antithesis needs to know what assertions you have defined in your code. The included tool, `antithesis-go-generator`, does this job. `antithesis-go-generator` scans your code and generates new code to register your Assertions with Antithesis.

### Using `antithesis-go-generator`

Add a Go directive somewhere in the main package for a module that has Antithesis assertions added to it.  The general form is:

`//go:generate antithesis-go-generator my-module-name`

For example:

`//go:generate antithesis-go-generator antithesis.com/go/sample-project`

Next, install the generator tool in your development environment:

```
go get github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-generator
go install github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-generator
```

Finally, add a build step to invoke `go generate <path>` prior to the typical compile/link build step of `go build <path>`.  The `antithesis-go-generator` will write a file named after the source file containing the `//go:generate` directive. For example, If the file containing the `//go:generate` directive is `main.go` then the derived file will be named `main_antithesis.go`.

After running `go generate <path>` run `go build <path>` and the module source, along with the newly generated `main_antithesis.go` file will be compiled and linked.

### Building

```
CC=clang go build ./assert ./random ./internal ./lifecycle 
```

