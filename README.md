
# antithesis-sdk-go

Provides functions enabling Go programs to include non-fatal Assertions and structured IO.

### Using exigen
The antithesis-sdk-go module also contains the exigen command, used to 
identify assertions that were added to a Go module, and to generate
corresponding function calls in an init() function to register these
assertions.  exigen can run from source, or it can be installed in a dev 
environment.  The notes here describe a run-from-source
use case.

Add a Go directive somewhere in the main package for
a module that has Antithesis assertions added to it.  The
general form is like this:

`//go:generate go run build-time-path-to-exigen my-module-name`

Example:

`//go:generate go run /home/src/antithesis-sdk-go/cmd/exigen.go github.com/synadia/nats-cluster`

A good place to add this directive is in the top-level
driver for an executable (often this is `main.go`)

With the directive in place, add a `go generate <path>` build step, prior to 
the typical compile/link build step `go build <path>`.  The `go generate <path>`
step will scan the provided path, looking for Golang source code files.
Any generate commands encountered will be executed as part of the `go generate`
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


