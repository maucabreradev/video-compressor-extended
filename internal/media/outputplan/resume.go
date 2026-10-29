package outputplan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/maucabreradev/video-compressor-extended/internal/media/pathutil"
)

// ResolveArtifactPath returns the canonical path when it is missing or already
// contains a valid artifact. If it exists but fails validation, a numbered
// retry path is selected without moving or replacing the existing file. Paths
// reserved for other planned outputs are never selected as retries.
func ResolveArtifactPath(destination string, reserved []string, validate func(string) error) (path string, skip bool, err error) {
	if strings.TrimSpace(destination) == "" {
		return "", false, fmt.Errorf("artifact destination must not be empty")
	}
	if validate == nil {
		return "", false, fmt.Errorf("artifact validator must not be nil")
	}
	reservedPaths := make(map[string]struct{}, len(reserved))
	for _, candidate := range reserved {
		key, err := resolvedPathKey(candidate)
		if err != nil {
			return "", false, fmt.Errorf("resolve reserved artifact path %q: %w", candidate, err)
		}
		reservedPaths[key] = struct{}{}
	}

	info, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return destination, false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("inspect artifact %q: %w", destination, err)
	}
	if info.Mode().IsRegular() && validate(destination) == nil {
		return destination, true, nil
	}

	extension := filepath.Ext(destination)
	base := strings.TrimSuffix(destination, extension)
	for attempt := 1; ; attempt++ {
		candidate := fmt.Sprintf("%s.vcx-retry-%d%s", base, attempt, extension)
		key, err := resolvedPathKey(candidate)
		if err != nil {
			return "", false, fmt.Errorf("resolve retry artifact path %q: %w", candidate, err)
		}
		if _, occupied := reservedPaths[key]; occupied {
			continue
		}
		info, err := os.Lstat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			return candidate, false, nil
		}
		if err != nil {
			return "", false, fmt.Errorf("inspect retry artifact %q: %w", candidate, err)
		}
		if info.Mode().IsRegular() && validate(candidate) == nil {
			return candidate, true, nil
		}
	}
}

func resolvedPathKey(path string) (string, error) {
	resolved, err := pathutil.Resolve(path)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		resolved = strings.ToLower(resolved)
	}
	return filepath.Clean(resolved), nil
}
