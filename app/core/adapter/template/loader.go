package template

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type OSFileSystemLoader struct {
	dirs []string
}

// NewOSFileSystemLoader returns an initialized OSFileSystemLoader.
func NewOSFileSystemLoader(paths ...string) *OSFileSystemLoader {
	return &OSFileSystemLoader{dirs: paths}
}

func (l *OSFileSystemLoader) getCustomTemplate(name string) []string {
	var customName = strings.Split(name, "::")

	// return strings.Replace(customName[1], "{name}","-asdasd-", -1)
	return customName
}

// Open opens a file from OS file system.
func (l *OSFileSystemLoader) Open(name string) (io.ReadCloser, error) {
	// name = l.getCustomTemplate(name)

	return os.Open(name)
}

// Exists checks if the template name exists by walking the list of template paths
// returns string with the full path of the template and bool true if the template file was found
func (l *OSFileSystemLoader) Exists(name string) (string, bool) {
	for i := 0; i < len(l.dirs); i++ {
		var fileName string

		if !strings.Contains(name, "::") {
			fileName = path.Join(l.dirs[i], name)
		} else {
			if strings.Contains(l.dirs[i], "{name}") {
				var customName = l.getCustomTemplate(name)
				fileName = path.Join(l.dirs[i], customName[1])
				fileName = strings.Replace(fileName, "{name}", customName[0], -1)
			}
		}

		if _, err := os.Stat(fileName); err == nil {
			return fileName, true
		}
	}
	return "", false
}

// AddPath adds the path to the internal list of paths searched when loading templates.
func (l *OSFileSystemLoader) AddPath(path string) {
	l.dirs = append(l.dirs, path)
}

// AddGopathPath adds a path located in the GOPATH.
// Example: l.AddGopathPath("github.com/CloudyKit/jet/example/views")
func (l *OSFileSystemLoader) AddGopathPath(path string) {
	paths := filepath.SplitList(os.Getenv("GOPATH"))
	for i := 0; i < len(paths); i++ {
		var err error
		path, err = filepath.Abs(filepath.Join(paths[i], "src", path))
		if err != nil {
			panic(errors.New("Can't add this path err: " + err.Error()))
		}

		if fstats, err := os.Stat(path); os.IsNotExist(err) == false && fstats.IsDir() {
			l.AddPath(path)
			return
		}
	}

	if fstats, err := os.Stat(path); os.IsNotExist(err) == false && fstats.IsDir() {
		l.AddPath(path)
	}
}
