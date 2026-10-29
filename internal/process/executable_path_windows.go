//go:build windows

package process

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
)

func normalizeExecutablePath(path string) (string, error) {
	return WindowsShortPath(path)
}

// WindowsShortPath returns an ASCII short-name alias when the path contains
// non-ASCII characters. Some pinned third-party Windows CRT builds crash when
// their executable path is passed through an ANSI argv[0].
func WindowsShortPath(path string) (string, error) {
	if !containsNonASCIIPathCharacter(path) {
		return path, nil
	}
	longPath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", fmt.Errorf("encode path as UTF-16: %w", err)
	}
	buffer := make([]uint16, 32768)
	length, err := windows.GetShortPathName(longPath, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", fmt.Errorf("GetShortPathNameW: %w", err)
	}
	if length == 0 || int(length) >= len(buffer) {
		return "", fmt.Errorf("GetShortPathNameW returned invalid path length %d", length)
	}
	shortPath := windows.UTF16ToString(buffer[:length])
	if strings.TrimSpace(shortPath) == "" || containsNonASCIIPathCharacter(shortPath) {
		return "", fmt.Errorf("Windows did not provide an ASCII short-name alias; use a filesystem with 8.3 short-name support")
	}
	return shortPath, nil
}

func containsNonASCIIPathCharacter(path string) bool {
	for _, character := range path {
		if character > 0x7f {
			return true
		}
	}
	return false
}
