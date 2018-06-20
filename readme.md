# mkpkg

Ever make a binary you just want to get onto someone's computer without them having to jump through all sorts of hoops? Yeah us too. We built this to scratch that itch, starting with an OS X installer.

mkpkg ("make package") creates installer packages for a distributable binary. It's based on the golang installer process, the [go build](https://github.com/golang/build) tooling in particular. It comes with a command-line utility, and a golang package that you can import into your toolchain if that's more your style.

### Project Status: :construction:
This is currently just a proof-of-concept we use to build the qri installer for os x. Next steps include building a wix MSI installer for windows, and some nice .tar.gz porcelain for linux.


### Getting started
```shell
$ go get github.com/qri-io/mkpkg
$ mkpkg -blank > config.yaml

# edit that file to taste. Then, on a mac:
$ mkpkg -config config.yaml -os darwin

# if it works, package will output to ./pkg
```

docs on what each field does are always available at https://godoc.org/github.com/qri-io/mkpkg/mkpkg


### Code-Signing for OS X
Mac OS X doesn't just let you cut installers all willy-nilly. So you'll need to _sign_ the resulting package, or else users will get a big security warning they can only get around by digging in system preferences security settings. You'll need a "Developer ID Installer"-type certificate for the next part, which you can only get if you're registered with apple's developer program. If you're a registered mac developer, you can [generate one using xcode](https://help.apple.com/developer-account/#/deveedc0daa0).

If any of these commands (like productsign) are missing, make sure you have xcode installed.

```shell
# first, let's list available identities...
$ security find-identity -v

# find a valid ID that starts with "Developer ID Installer:", copy the contents of that string for the --sign flag
# then sign the package:
$ productsign --sign 'Developer ID Installer: qri, inc.' qri_os_x_cli_darwin_amd64_unsigned.pkg qri_os_x_cli_darwin_amd64_signed.pkg

# voila, no more awful security messages
```