package mkpkg

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// MSIConfig configures an MSI Package
type MSIConfig struct {
}

const wixBinaries = "https://storage.googleapis.com/go-builder-data/wix311-binaries.zip"
const wixSha256 = "da034c489bd1dd6d8e1623675bf5e899f32d74d6d8312f8dd125a084543193de"

func (p Package) windowsMSI() error {
	cwd, version, err := p.environ()
	if err != nil {
		return err
	}

	panic("windows MSI not yet finished")

	// Install Wix tools.
	wix := filepath.Join(cwd, "wix")
	defer os.RemoveAll(wix)
	if err := installWix(wix); err != nil {
		return err
	}

	windowsData, err := p.windowsData()
	if err != nil {
		return err
	}

	// Write out windows data that is used by the packaging process.
	win := filepath.Join(cwd, "windows")
	defer os.RemoveAll(win)
	if err := writeDataFiles(windowsData, win); err != nil {
		return err
	}

	// Gather files.
	goDir := filepath.Join(cwd, "go")
	appfiles := filepath.Join(win, "AppFiles.wxs")
	if err := runDir(win, filepath.Join(wix, "heat"),
		"dir", goDir,
		"-nologo",
		"-gg", "-g1", "-srd", "-sfrag",
		"-cg", "AppFiles",
		"-template", "fragment",
		"-dr", "INSTALLDIR",
		"-var", "var.SourceDir",
		"-out", appfiles,
	); err != nil {
		return err
	}

	msArch := func() string {
		switch runtime.GOARCH {
		default:
			panic("unknown arch for windows " + runtime.GOARCH)
		case "386":
			return "x86"
		case "amd64":
			return "x64"
		}
	}

	// Build package.
	verMajor, verMinor, verPatch := wixVersion(version)

	if err := runDir(win, filepath.Join(wix, "candle"),
		"-nologo",
		"-arch", msArch(),
		"-dGoVersion="+version,
		fmt.Sprintf("-dWixGoVersion=%v.%v.%v", verMajor, verMinor, verPatch),
		fmt.Sprintf("-dIsWinXPSupported=%v", wixIsWinXPSupported(version)),
		"-dArch="+runtime.GOARCH,
		"-dSourceDir="+goDir,
		filepath.Join(win, "installer.wxs"),
		appfiles,
	); err != nil {
		return err
	}

	msi := filepath.Join(cwd, "msi") // known to cmd/release
	if err := os.Mkdir(msi, 0755); err != nil {
		return err
	}
	return runDir(win, filepath.Join(wix, "light"),
		"-nologo",
		"-dcl:high",
		"-ext", "WixUIExtension",
		"-ext", "WixUtilExtension",
		"AppFiles.wixobj",
		"installer.wixobj",
		"-o", filepath.Join(msi, "go.msi"), // file name irrelevant
	)
}

var versionRe = regexp.MustCompile(`^v(\d+(\.\d+)*)`)

// wixVersion splits a Go version string such as "v1.9" or "v1.10.2" (as matched by versionRe)
// into its three parts: major, minor, and patch
// It's based on the Git tag.
func wixVersion(v string) (major, minor, patch int) {
	m := versionRe.FindStringSubmatch(v)
	if m == nil {
		return
	}
	parts := strings.Split(m[1], ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])

		if len(parts) >= 2 {
			minor, _ = strconv.Atoi(parts[1])

			if len(parts) >= 3 {
				patch, _ = strconv.Atoi(parts[2])
			}
		}
	}
	return
}

// wixIsWinXPSupported checks if Windows XP
// support is expected from the specified version.
// (WinXP is no longer supported starting Go v1.11)
func wixIsWinXPSupported(v string) bool {
	major, minor, _ := wixVersion(v)
	if major > 1 {
		return false
	}
	if minor >= 11 {
		return false
	}
	return true
}

// installWix fetches and installs the wix toolkit to the specified path.
func installWix(path string) error {
	// Fetch wix binary zip file.
	body, err := httpGet(wixBinaries)
	if err != nil {
		return err
	}

	// Verify sha256
	sum := sha256.Sum256(body)
	if fmt.Sprintf("%x", sum) != wixSha256 {
		return errors.New("sha256 mismatch for wix toolkit")
	}

	// Unzip to path.
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		name := filepath.FromSlash(f.Name)
		err := os.MkdirAll(filepath.Join(path, filepath.Dir(name)), 0755)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		b, err := ioutil.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(path, name), b, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p Package) windowsData() (map[string]string, error) {

	installerWxs, err := p.execTemplate(installerWxsTmpl)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"installer.wxs":         installerWxs,
		"LICENSE.rtf":           storageBase + "windows/LICENSE.rtf",
		"images/Banner.jpg":     storageBase + "windows/Banner.jpg",
		"images/Dialog.jpg":     storageBase + "windows/Dialog.jpg",
		"images/DialogLeft.jpg": storageBase + "windows/DialogLeft.jpg",
		"images/gopher.ico":     storageBase + "windows/gopher.ico",
	}, nil
}

var installerWxsTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
<!--
# Copyright 2010 The Go Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
-->

<?if $(var.Arch) = 386 ?>
  <?define ProdId = {FF5B30B2-08C2-11E1-85A2-6ACA4824019B} ?>
  <?define UpgradeCode = {1C3114EA-08C3-11E1-9095-7FCA4824019B} ?>
  <?define SysFolder=SystemFolder ?>
<?else?>
  <?define ProdId = {716c3eaa-9302-48d2-8e5e-5cfec5da2fab} ?>
  <?define UpgradeCode = {22ea7650-4ac6-4001-bf29-f4b8775db1c0} ?>
  <?define SysFolder=System64Folder ?>
<?endif?>

<Product
    Id="*"
    Name="Go Programming Language $(var.Arch) $(var.GoVersion)"
    Language="1033"
    Version="$(var.WixGoVersion)"
    Manufacturer="https://golang.org"
    UpgradeCode="$(var.UpgradeCode)" >

<Package
    Id='*'
    Keywords='Installer'
    Description="The Go Programming Language Installer"
    Comments="The Go programming language is an open source project to make programmers more productive."
    InstallerVersion="300"
    Compressed="yes"
    InstallScope="perMachine"
    Languages="1033" />

<Property Id="ARPCOMMENTS" Value="The Go programming language is a fast, statically typed, compiled language that feels like a dynamically typed, interpreted language." />
<Property Id="ARPCONTACT" Value="golang-nuts@googlegroups.com" />
<Property Id="ARPHELPLINK" Value="https://golang.org/help/" />
<Property Id="ARPREADME" Value="https://golang.org" />
<Property Id="ARPURLINFOABOUT" Value="https://golang.org" />
<Property Id="LicenseAccepted">1</Property>
<Icon Id="gopher.ico" SourceFile="images\gopher.ico"/>
<Property Id="ARPPRODUCTICON" Value="gopher.ico" />
<Property Id="EXISTING_GOLANG_INSTALLED">
  <RegistrySearch Id="installed" Type="raw" Root="HKCU" Key="Software\GoProgrammingLanguage" Name="installed" />
</Property>
<Media Id='1' Cabinet="go.cab" EmbedCab="yes" CompressionLevel="high" />
<?if $(var.IsWinXPSupported) = true ?>
    <Condition Message="Windows XP (with Service Pack 2) or greater required.">
        (VersionNT >= 501 AND (WindowsBuild > 2600 OR ServicePackLevel >= 2))
    </Condition>
<?else?>
    <Condition Message="Windows 7 (with Service Pack 1) or greater required.">
        ((VersionNT > 601) OR (VersionNT = 601 AND ServicePackLevel >= 1))
    </Condition>
<?endif?>
<MajorUpgrade AllowDowngrades="yes" />
<SetDirectory Id="INSTALLDIRROOT" Value="[%SYSTEMDRIVE]"/>

<CustomAction
    Id="SetApplicationRootDirectory"
    Property="ARPINSTALLLOCATION"
    Value="[INSTALLDIR]" />

<!-- Define the directory structure and environment variables -->
<Directory Id="TARGETDIR" Name="SourceDir">
  <Directory Id="INSTALLDIRROOT">
    <Directory Id="INSTALLDIR" Name="Go"/>
  </Directory>
  <Directory Id="ProgramMenuFolder">
    <Directory Id="GoProgramShortcutsDir" Name="Go Programming Language"/>
  </Directory>
  <Directory Id="EnvironmentEntries">
    <Directory Id="GoEnvironmentEntries" Name="Go Programming Language"/>
  </Directory>
</Directory>

<!-- Programs Menu Shortcuts -->
<DirectoryRef Id="GoProgramShortcutsDir">
  <Component Id="Component_GoProgramShortCuts" Guid="{f5fbfb5e-6c5c-423b-9298-21b0e3c98f4b}">
    <Shortcut
        Id="GoDocServerStartMenuShortcut"
        Name="GoDocServer"
        Description="Starts the Go documentation server (http://localhost:6060)"
        Show="minimized"
        Arguments='/c start "Godoc Server http://localhost:6060" "[INSTALLDIR]bin\godoc.exe" -http=localhost:6060 -goroot="[INSTALLDIR]." &amp;&amp; start http://localhost:6060'
        Icon="gopher.ico"
        Target="[%ComSpec]" />
    <Shortcut
        Id="UninstallShortcut"
        Name="Uninstall Go"
        Description="Uninstalls Go and all of its components"
        Target="[$(var.SysFolder)]msiexec.exe"
        Arguments="/x [ProductCode]" />
    <RemoveFolder
        Id="GoProgramShortcutsDir"
        On="uninstall" />
    <RegistryValue
        Root="HKCU"
        Key="Software\GoProgrammingLanguage"
        Name="ShortCuts"
        Type="integer"
        Value="1"
        KeyPath="yes" />
  </Component>
</DirectoryRef>

<!-- Registry & Environment Settings -->
<DirectoryRef Id="GoEnvironmentEntries">
  <Component Id="Component_GoEnvironment" Guid="{3ec7a4d5-eb08-4de7-9312-2df392c45993}">
    <RegistryKey
        Root="HKCU"
        Key="Software\GoProgrammingLanguage">
            <RegistryValue
                Name="installed"
                Type="integer"
                Value="1"
                KeyPath="yes" />
            <RegistryValue
                Name="installLocation"
                Type="string"
                Value="[INSTALLDIR]" />
    </RegistryKey>
    <Environment
        Id="GoPathEntry"
        Action="set"
        Part="last"
        Name="PATH"
        Permanent="no"
        System="yes"
        Value="[INSTALLDIR]bin" />
    <Environment
        Id="GoRoot"
        Action="set"
        Part="all"
        Name="GOROOT"
        Permanent="no"
        System="yes"
        Value="[INSTALLDIR]" />
    <Environment
        Id="UserGoPath"
        Action="create"
        Name="GOPATH"
        Permanent="no"
        Value="%USERPROFILE%\go" />
    <Environment
        Id="UserGoPathEntry"
        Action="set"
        Part="last"
        Name="PATH"
        Permanent="no"
        Value="%GOPATH%\bin" />
    <RemoveFolder
        Id="GoEnvironmentEntries"
        On="uninstall" />
  </Component>
</DirectoryRef>

<!-- Install the files -->
<Feature
    Id="GoTools"
    Title="Go"
    Level="1">
      <ComponentRef Id="Component_GoEnvironment" />
      <ComponentGroupRef Id="AppFiles" />
      <ComponentRef Id="Component_GoProgramShortCuts" />
</Feature>

<!-- Update the environment -->
<InstallExecuteSequence>
    <Custom Action="SetApplicationRootDirectory" Before="InstallFinalize" />
</InstallExecuteSequence>

<!-- Notify top level applications of the new PATH variable (golang.org/issue/18680)  -->
<CustomActionRef Id="WixBroadcastEnvironmentChange" />

<!-- Include the user interface -->
<WixVariable Id="WixUILicenseRtf" Value="LICENSE.rtf" />
<WixVariable Id="WixUIBannerBmp" Value="images\Banner.jpg" />
<WixVariable Id="WixUIDialogBmp" Value="images\Dialog.jpg" />
<Property Id="WIXUI_INSTALLDIR" Value="INSTALLDIR" />
<UIRef Id="Golang_InstallDir" />
<UIRef Id="WixUI_ErrorProgressText" />

</Product>
<Fragment>
  <!--
    The installer steps are modified so we can get user confirmation to uninstall an existing golang installation.

    WelcomeDlg  [not installed]  =>                  LicenseAgreementDlg => InstallDirDlg  ..
                [installed]      => OldVersionDlg => LicenseAgreementDlg => InstallDirDlg  ..
  -->
  <UI Id="Golang_InstallDir">
    <!-- style -->
    <TextStyle Id="WixUI_Font_Normal" FaceName="Tahoma" Size="8" />
    <TextStyle Id="WixUI_Font_Bigger" FaceName="Tahoma" Size="12" />
    <TextStyle Id="WixUI_Font_Title" FaceName="Tahoma" Size="9" Bold="yes" />

    <Property Id="DefaultUIFont" Value="WixUI_Font_Normal" />
    <Property Id="WixUI_Mode" Value="InstallDir" />

    <!-- dialogs -->
    <DialogRef Id="BrowseDlg" />
    <DialogRef Id="DiskCostDlg" />
    <DialogRef Id="ErrorDlg" />
    <DialogRef Id="FatalError" />
    <DialogRef Id="FilesInUse" />
    <DialogRef Id="MsiRMFilesInUse" />
    <DialogRef Id="PrepareDlg" />
    <DialogRef Id="ProgressDlg" />
    <DialogRef Id="ResumeDlg" />
    <DialogRef Id="UserExit" />
    <Dialog Id="OldVersionDlg" Width="240" Height="95" Title="[ProductName] Setup" NoMinimize="yes">
      <Control Id="Text" Type="Text" X="28" Y="15" Width="194" Height="50">
        <Text>A previous version of Go Programming Language is currently installed. By continuing the installation this version will be uninstalled. Do you want to continue?</Text>
      </Control>
      <Control Id="Exit" Type="PushButton" X="123" Y="67" Width="62" Height="17"
        Default="yes" Cancel="yes" Text="No, Exit">
        <Publish Event="EndDialog" Value="Exit">1</Publish>
      </Control>
      <Control Id="Next" Type="PushButton" X="55" Y="67" Width="62" Height="17" Text="Yes, Uninstall">
        <Publish Event="EndDialog" Value="Return">1</Publish>
      </Control>
    </Dialog>

    <!-- wizard steps -->
    <Publish Dialog="BrowseDlg" Control="OK" Event="DoAction" Value="WixUIValidatePath" Order="3">1</Publish>
    <Publish Dialog="BrowseDlg" Control="OK" Event="SpawnDialog" Value="InvalidDirDlg" Order="4"><![CDATA[NOT WIXUI_DONTVALIDATEPATH AND WIXUI_INSTALLDIR_VALID<>"1"]]></Publish>

    <Publish Dialog="ExitDialog" Control="Finish" Event="EndDialog" Value="Return" Order="999">1</Publish>

    <Publish Dialog="WelcomeDlg" Control="Next" Event="NewDialog" Value="OldVersionDlg"><![CDATA[EXISTING_GOLANG_INSTALLED << "#1"]]> </Publish>
    <Publish Dialog="WelcomeDlg" Control="Next" Event="NewDialog" Value="LicenseAgreementDlg"><![CDATA[NOT (EXISTING_GOLANG_INSTALLED << "#1")]]></Publish>

    <Publish Dialog="OldVersionDlg" Control="Next" Event="NewDialog" Value="LicenseAgreementDlg">1</Publish>

    <Publish Dialog="LicenseAgreementDlg" Control="Back" Event="NewDialog" Value="WelcomeDlg">1</Publish>
    <Publish Dialog="LicenseAgreementDlg" Control="Next" Event="NewDialog" Value="InstallDirDlg">LicenseAccepted = "1"</Publish>

    <Publish Dialog="InstallDirDlg" Control="Back" Event="NewDialog" Value="LicenseAgreementDlg">1</Publish>
    <Publish Dialog="InstallDirDlg" Control="Next" Event="SetTargetPath" Value="[WIXUI_INSTALLDIR]" Order="1">1</Publish>
    <Publish Dialog="InstallDirDlg" Control="Next" Event="DoAction" Value="WixUIValidatePath" Order="2">NOT WIXUI_DONTVALIDATEPATH</Publish>
    <Publish Dialog="InstallDirDlg" Control="Next" Event="SpawnDialog" Value="InvalidDirDlg" Order="3"><![CDATA[NOT WIXUI_DONTVALIDATEPATH AND WIXUI_INSTALLDIR_VALID<>"1"]]></Publish>
    <Publish Dialog="InstallDirDlg" Control="Next" Event="NewDialog" Value="VerifyReadyDlg" Order="4">WIXUI_DONTVALIDATEPATH OR WIXUI_INSTALLDIR_VALID="1"</Publish>
    <Publish Dialog="InstallDirDlg" Control="ChangeFolder" Property="_BrowseProperty" Value="[WIXUI_INSTALLDIR]" Order="1">1</Publish>
    <Publish Dialog="InstallDirDlg" Control="ChangeFolder" Event="SpawnDialog" Value="BrowseDlg" Order="2">1</Publish>

    <Publish Dialog="VerifyReadyDlg" Control="Back" Event="NewDialog" Value="InstallDirDlg" Order="1">NOT Installed</Publish>
    <Publish Dialog="VerifyReadyDlg" Control="Back" Event="NewDialog" Value="MaintenanceTypeDlg" Order="2">Installed AND NOT PATCH</Publish>
    <Publish Dialog="VerifyReadyDlg" Control="Back" Event="NewDialog" Value="WelcomeDlg" Order="2">Installed AND PATCH</Publish>

    <Publish Dialog="MaintenanceWelcomeDlg" Control="Next" Event="NewDialog" Value="MaintenanceTypeDlg">1</Publish>

    <Publish Dialog="MaintenanceTypeDlg" Control="RepairButton" Event="NewDialog" Value="VerifyReadyDlg">1</Publish>
    <Publish Dialog="MaintenanceTypeDlg" Control="RemoveButton" Event="NewDialog" Value="VerifyReadyDlg">1</Publish>
    <Publish Dialog="MaintenanceTypeDlg" Control="Back" Event="NewDialog" Value="MaintenanceWelcomeDlg">1</Publish>

    <Property Id="ARPNOMODIFY" Value="1" />
  </UI>

  <UIRef Id="WixUI_Common" />
</Fragment>
</Wix>
`
