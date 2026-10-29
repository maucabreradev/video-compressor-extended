//go:build !windows

package process

func normalizeExecutablePath(path string) (string, error) {
	return path, nil
}
