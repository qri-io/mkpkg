// Package mkpkg is a
package mkpkg

import (
	"bytes"
	"os"
	"text/template"
)

// Package contains state for package construction
type Package struct {
	// Human-friendly name of the project, eg: "The Go Programming Language"
	Name string
	// name of the binary. eg "go"
	BinName string
	// short description. eg: "The Go programming language is a fast, statically typed, compiled language that feels like a dynamically typed, interpreted language."
	Description string
	// url for the project eg: "https://golang.org"
	SiteURL string
	// app identifier in reverse-domain notation, eg: com.googlecode.go
	Identifier string
	// semantic version identifier with "v" prefix. eg: v1.0.0
	Version string
	// Darwin-Specific Configuration Details
	Darwin DarwinConfig
	// MSI-Specific Configuration Details
	MSI MSIConfig
}

// MakeDarwin creates an os X .pkg
func (p Package) MakeDarwin() error {
	return p.darwinPKG()
}

// environ returns commonly required details for the environment mkpkg is operating in
func (p Package) environ() (cwd, version string, err error) {
	cwd, err = os.Getwd()
	if err != nil {
		return
	}
	version = p.Version
	return
}

// execTemplate executes a template string against package info
func (p Package) execTemplate(tmpl string) (string, error) {
	buf := &bytes.Buffer{}
	t, err := template.New("template").Parse(tmpl)
	if err != nil {
		return "", err
	}
	if err = t.Execute(buf, p); err != nil {
		return "", err
	}
	return buf.String(), nil
}
