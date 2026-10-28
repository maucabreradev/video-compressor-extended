package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maucabreradev/video-compressor-extended/internal/media/pathutil"
)

var supportedExtensions = map[string]struct{}{
	".avi":  {},
	".flv":  {},
	".m4v":  {},
	".mkv":  {},
	".mov":  {},
	".mp4":  {},
	".ts":   {},
	".webm": {},
	".wmv":  {},
}

// Discover returns supported regular files from input files or directories.
// Directory traversal is recursive, deterministic, and excludes the output tree.
func Discover(inputs []string, outputRoot string) ([]string, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("at least one input path is required")
	}
	if strings.TrimSpace(outputRoot) == "" {
		return nil, fmt.Errorf("output root must not be empty")
	}

	outputAbs, err := pathutil.Resolve(outputRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve output root %q: %w", outputRoot, err)
	}

	found := make(map[string]struct{})
	for _, input := range inputs {
		inputAbs, err := pathutil.Resolve(input)
		if err != nil {
			return nil, fmt.Errorf("resolve input %q: %w", input, err)
		}
		info, err := os.Stat(inputAbs)
		if err != nil {
			return nil, fmt.Errorf("inspect input %q: %w", inputAbs, err)
		}

		if info.IsDir() {
			err = filepath.WalkDir(inputAbs, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return fmt.Errorf("walk %q: %w", path, walkErr)
				}
				pathAbs, err := pathutil.Resolve(path)
				if err != nil {
					return fmt.Errorf("resolve discovered path %q: %w", path, err)
				}
				if entry.IsDir() && isWithin(outputAbs, pathAbs) {
					return filepath.SkipDir
				}
				if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
					return nil
				}
				fileInfo, err := entry.Info()
				if err != nil {
					return fmt.Errorf("inspect discovered file %q: %w", pathAbs, err)
				}
				if fileInfo.Mode().IsRegular() && isSupported(pathAbs) && !isWithin(outputAbs, pathAbs) {
					found[pathAbs] = struct{}{}
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("discover inputs under %q: %w", inputAbs, err)
			}
			continue
		}

		if info.Mode().IsRegular() && isSupported(inputAbs) && !isWithin(outputAbs, inputAbs) {
			found[inputAbs] = struct{}{}
		}
	}

	result := make([]string, 0, len(found))
	for path := range found {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func isSupported(path string) bool {
	_, ok := supportedExtensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	return relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}
