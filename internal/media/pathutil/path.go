// Package pathutil provides filesystem-aware path operations shared by media
// discovery and output planning.
package pathutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Resolve returns an absolute, cleaned path with all existing path components
// resolved through symbolic links. Missing trailing components are preserved.
func Resolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	probe := filepath.Clean(abs)
	var missing []string
	for {
		_, err := os.Lstat(probe)
		if err == nil {
			probe, err = filepath.EvalSymlinks(probe)
			if err != nil {
				return "", err
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", err
		}
		missing = append(missing, filepath.Base(probe))
		probe = parent
	}
	for index := len(missing) - 1; index >= 0; index-- {
		probe = filepath.Join(probe, missing[index])
	}
	return filepath.Clean(probe), nil
}
