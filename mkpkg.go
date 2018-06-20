package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/qri-io/mkpkg/mkpkg"
)

var (
	cfg   string
	os    string
	blank bool
)

func init() {
	flag.BoolVar(&blank, "blank", false, "print blank YAML configuration file")
	flag.StringVar(&cfg, "config", "", "path to config.yaml file")
	flag.StringVar(&os, "os", "", "operating system to create package for. One of: darwin,windows,linux")
}

const helpText = `
mkpkg creates installer packages for a distributable binary
`

func main() {
	flag.Parse()
	if blank {
		fmt.Print(blankFile)
		return
	}

	if cfg == "" {
		fmt.Println(helpText)
		flag.PrintDefaults()
		return
	}

	data, err := ioutil.ReadFile(cfg)
	if err != nil {
		fmt.Printf("error reading config file: %s\n", err.Error())
		return
	}

	r := mkpkg.Package{}
	if err := yaml.Unmarshal(data, &r); err != nil {
		fmt.Printf("error decoding yaml file: %s\n", err.Error())
		return
	}

	switch os {
	case "darwin":
		if err := r.MakeDarwin(); err != nil {
			fmt.Printf("error creating darwin package: %s\n", err.Error())
			return
		}
	case "linux":
		fmt.Printf("linux packages not yet supported.\n")
	case "windows":
		fmt.Printf("windows packages not yet supported.\n")
	}
}

const blankFile = `Name: "Qri CLI"
BinName: "qri"
Identifier: "io.qri.cli"
Version: "v0.5.0"
Description: "qri is a web of datasets"
Darwin:
  WelcomeMsg: |
    The following steps will guide you to installing the qri command line client. Once installed you'll have access to qri from the command line.

    Please note that this is not the qri desktop app, for more info on the desktop app check https://qri.io/downloads
  ConclusionMsg: |
    Thanks for installing qri CLI. For documentation and tutorials be sure to check out https://docs.qri.io
  MinOSXVersion: "10.6.0"
  BgPngPath: assets/darwin/bg.png
  BinPath: /go/bin/qri
`
