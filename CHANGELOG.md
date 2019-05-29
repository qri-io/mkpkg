#  (2019-05-23)

mkpkg ("make package") creates installer packages for a distributable binary. It's based on the golang installer process, the [go build](https://github.com/golang/build) tooling in particular. It comes with a command-line utility, and a golang package that you can import into your toolchain if that's more your style.

This is the first proper release of `mkpkg`. In preparation for go 1.13, in which `go.mod` files and go modules are the primary way to handle go dependencies, we are going to do an official release of all our modules. This will be version v0.1.0 of `mkpkg`.