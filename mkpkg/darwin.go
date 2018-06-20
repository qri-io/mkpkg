package mkpkg

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

// DarwinConfig encapsulates configuration details for creating a darwin PKG
type DarwinConfig struct {
	// welcome message to show in the installer when installer is launched
	WelcomeMsg string
	// conclusion message to show in the installer when installation is complete
	ConclusionMsg string
	// path to a 140x370 png file to use as the installer background
	BgPngPath string
	// Minimum os x version. Default is 10.6.0
	MinOSXVersion string
	// Path to compatible darwin binary executable to install
	// name of binary must
	BinPath string
}

func (p Package) darwinPKG() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("can only build darwin installer pkg on darwin (Mac) Operating system")
	}

	cwd, version, err := p.environ()
	if err != nil {
		return err
	}

	darwinData, err := p.darwinData()
	if err != nil {
		return err
	}
	// Write out darwin data that is used by the packaging process.
	defer os.RemoveAll("darwin")
	if err := writeDataFiles(darwinData, "darwin"); err != nil {
		return err
	}

	// Create a work directory and place inside the files as they should
	// be on the destination file system.
	work := filepath.Join(cwd, "darwinpkg")
	if err := os.MkdirAll(work, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(work)

	// Write out /etc/paths.d/[p.BinName]
	pathsBody := fmt.Sprintf("/usr/local/%s/bin", p.BinName)
	pathsDir := filepath.Join(work, "etc/paths.d")
	pathsFile := filepath.Join(pathsDir, p.BinName)
	if err := os.MkdirAll(pathsDir, 0755); err != nil {
		return err
	}
	if err = ioutil.WriteFile(pathsFile, []byte(pathsBody), 0644); err != nil {
		return err
	}

	// Copy installation to /usr/local/[p.BinName]
	binDir := filepath.Join(work, "usr/local", p.BinName, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	if err := cp(filepath.Join(binDir, p.BinName), p.Darwin.BinPath); err != nil {
		return err
	}

	// Build the package file.
	dest := "package"
	if err := os.Mkdir(dest, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(dest)

	// run pkbuild tool
	if err := run("pkgbuild",
		"--identifier", p.Identifier,
		"--version", version,
		"--scripts", "darwin/scripts",
		"--root", work,
		filepath.Join(dest, fmt.Sprintf("%s.pkg", p.Identifier)),
	); err != nil {
		return err
	}

	const pkg = "pkg" // known to cmd/release
	if err := os.Mkdir(pkg, 0755); err != nil {
		return err
	}

	return run("productbuild",
		"--distribution", "darwin/Distribution",
		"--resources", "darwin/Resources",
		"--package-path", dest,
		filepath.Join(cwd, pkg, fmt.Sprintf("%s.pkg", p.Name)), // file name irrelevant
	)
}

func (p Package) darwinData() (map[string]string, error) {
	// moar info on this: https://developer.apple.com/library/archive/documentation/DeveloperTools/Reference/DistributionDefinitionRef/Chapters/Introduction.html#//apple_ref/doc/uid/TP40005370-CH1-SW1
	// (docs are apparently out of date, but seem to work ok...)
	distTmpl := `<?xml version="1.0" encoding="utf-8" standalone="no"?>
<installer-script minSpecVersion="1.000000">
    <title>{{ .Name }}</title>
    {{ if .Darwin.BgPngPath }}
    <background mime-type="image/png" file="bg.png"/>
    {{ end }}
    <options customize="never" allow-external-scripts="no"/>
    <domains enable_localSystem="true" />
    <installation-check script="installCheck();"/>
    {{ if .Darwin.WelcomeMsg }}
    <welcome mime-type="text/plain" file="welcome.txt"/>
    {{ end}}
    {{ if .Darwin.MinOSXVersion }}
    <allowed-os-versions>
      <os-version min="{{ .Darwin.MinOSXVersion }}" />
    </allowed-os-versions>
    {{ end }}
    <script>
function installCheck() {
    if(system.files.fileExistsAtPath('/usr/local/{{ .BinName }}/bin/{{ .BinName }}')) {
      my.result.title = 'Previous Installation Detected';
      my.result.message = 'A previous installation of {{ .Name }} exists at /usr/local/{{ .BinName }}. This installer will remove the previous installation prior to installing. Please back up any data before proceeding.';
      my.result.type = 'Warning';
      return false;
  }
    return true;
}
    </script>
    <choices-outline>
        <line choice="{{ .Identifier }}.choice"/>
    </choices-outline>
    <choice id="{{ .Identifier }}.choice" title="{{ .Name }}">
        <pkg-ref id="{{ .Identifier }}.pkg"/>
    </choice>
    <pkg-ref id="{{ .Identifier }}.pkg" auth="Root">{{ .Identifier }}.pkg</pkg-ref>
    {{ if .Darwin.ConclusionMsg }}
    <conclusion mime-type="text/plain" file="conclusion.txt"/>
    {{ end }}
</installer-script>
`
	dist, err := p.execTemplate(distTmpl)
	if err != nil {
		return nil, err
	}

	preInstallTmpl := `#!/bin/bash
PROJROOT=/usr/local/{{ .BinName }}
echo "Removing previous installation"
if [ -d $PROJROOT ]; then
  rm -r $PROJROOT
fi
`
	preinstall, err := p.execTemplate(preInstallTmpl)
	if err != nil {
		return nil, err
	}

	postInstallTmpl := `#!/bin/bash
PROJROOT=/usr/local/{{ .BinName }}
echo "Fixing permissions"
cd $PROJROOT
find . -exec chmod ugo+r \{\} \;
find bin -exec chmod ugo+rx \{\} \;
find . -type d -exec chmod ugo+rx \{\} \;
chmod o-w .
`
	postinstall, err := p.execTemplate(postInstallTmpl)
	if err != nil {
		return nil, err
	}

	var bgPngStr string
	if p.Darwin.BgPngPath != "" {
		b, err := ioutil.ReadFile(p.Darwin.BgPngPath)
		if err != nil {
			return nil, err
		}
		bgPngStr = string(b)
	}

	return map[string]string{
		"scripts/preinstall":       preinstall,
		"scripts/postinstall":      postinstall,
		"Distribution":             dist,
		"Resources/welcome.txt":    p.Darwin.WelcomeMsg,
		"Resources/conclusion.txt": p.Darwin.ConclusionMsg,
		"Resources/bg.png":         bgPngStr,
	}, nil
}
