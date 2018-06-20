package mkpkg

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// TODO - remove this stuff
const storageBase = "https://storage.googleapis.com/go-builder-data/release/"

// writeDataFiles writes the files in the provided map to the provided base
// directory. If the map value is a URL it fetches the data at that URL and
// uses it as the file contents.
func writeDataFiles(data map[string]string, base string) error {
	for name, body := range data {
		dst := filepath.Join(base, name)
		err := os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			return err
		}
		b := []byte(body)
		if strings.HasPrefix(body, storageBase) {
			b, err = httpGet(body)
			if err != nil {
				return err
			}
		}
		// (We really mean 0755 on the next line; some of these files
		// are executable, and there's no harm in making them all so.)
		if err := ioutil.WriteFile(dst, b, 0755); err != nil {
			return err
		}
	}
	return nil
}

func httpGet(url string) ([]byte, error) {
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, errors.New(r.Status)
	}
	return body, nil
}

func run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runDir(dir, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func cp(dst, src string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	fi, err := sf.Stat()
	if err != nil {
		return err
	}
	tmpDst := dst + ".tmp"
	df, err := os.Create(tmpDst)
	if err != nil {
		return err
	}
	defer df.Close()
	// Windows doesn't implement Fchmod.
	if runtime.GOOS != "windows" {
		if err := df.Chmod(fi.Mode()); err != nil {
			return err
		}
	}
	_, err = io.Copy(df, sf)
	if err != nil {
		return err
	}
	if err := df.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpDst, dst); err != nil {
		return err
	}
	// Ensure the destination has the same mtime as the source.
	return os.Chtimes(dst, fi.ModTime(), fi.ModTime())
}

func cpDir(dst, src string) error {
	walk := func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, srcPath[len(src):])
		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		return cp(dstPath, srcPath)
	}
	return filepath.Walk(src, walk)
}

func cpAllDir(dst, basePath string, dirs ...string) error {
	for _, dir := range dirs {
		if err := cpDir(filepath.Join(dst, dir), filepath.Join(basePath, dir)); err != nil {
			return err
		}
	}
	return nil
}

func ext() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
