
# antilog

Provides Always() Somtimes() and Assert() functions used to
specify Antithesis assertions used in a go module.  

### Using exigen
The antilog module also contains the exigen command, used to 
discover existential assertions in a Go module, and to generate
Assert() calls in an init() function for that module.  exigen
can run from source, or it can be installed in a dev 
environment.  The notes here describe a run-from-source
use case.

Add a Go directive somewhere in the main package for
a module that has Antithesis assertions added to it.  The
general form is like this:

`//go:generate go run build-time-path-to-exigen my-module-name`

Example:

`//go:generate go run /home/adk/go/tools/antilog/cmd/exigen.go github.com/synadia/nats-cluster`

A good place to add this directive is in the top-level
driver for an executable (often this is `main.go`)

With the directive in place, add a `go generate <path>` build step, prior to 
the typical compile/link build step `go build <path>`.  The `go generate <path>`
step will scan the provided path, looking for Golang source code files.
Any generate commands encountered will be executed as part of the `go generate`
process.  When exigen runs, it will scan for packages in the specified
module indicated in the corresponding //go:generate directive.  

All of the files in each package of the module will be scanned for 
existential assertions, and result in one Expect() call to be created
for each existential assertion that was scanned.  The Expect() calls 
will be written to a new file whose name is derived from the name of the file that
contains the //go:generate directive.  

If the file containing the //go:generate directive is `main.go`  then the 
derived file will be `main_exigen.go` and will include an init() function 
containing the Expect() calls that were created.

After running `go generate <path>` run `go build <path>` and the
module source, along with the newly generated `main_exigen.go` file
will be compiled and linked.

### Building antilog
When building antilog, make sure to specify clang and to set CGO_ENABLED
  
Example:

    CC=clang CGO_ENABLED=1 go build


