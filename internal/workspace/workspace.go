// Package workspace creates short, per-user workspaces for external media tools.
package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf16"
)

const (
	defaultMaxPathUTF16Units = 240
	workspacePrefix          = "w-"
	workspaceIDBytes         = 6
	workspaceCreateAttempts  = 16
)

var artifactNames = map[string]struct{}{
	"audio.wav":      {},
	"video.mp4":      {},
	"transcript":     {},
	"transcript.srt": {},
}

// Manager creates opaque workspaces under caller-approved per-user roots.
// Roots are considered in order, with roots on outputDirectory's volume tried
// first. If Roots is empty, the per-user cache directory is used.
type Manager struct {
	// Roots are caller-approved, per-user roots. The default root is the per-user
	// cache directory, which provides the platform's inherited user ACLs.
	Roots             []string
	MaxPathUTF16Units int
}

// Workspace identifies one directory owned by VCX. Remove deletes only this
// directory; it never removes the workspace root, source media, or final output.
type Workspace struct {
	path  string
	root  string
	limit int
}

// Create makes a unique workspace and verifies that its planned artifact paths
// fit the configured tool path budget and that the directory is writable.
func (manager Manager) Create(outputDirectory string) (*Workspace, error) {
	if strings.TrimSpace(outputDirectory) == "" {
		return nil, fmt.Errorf("output directory is required to select a workspace volume")
	}

	outputPath, err := filepath.Abs(outputDirectory)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory %q: %w", outputDirectory, err)
	}

	limit := manager.MaxPathUTF16Units
	if limit <= 0 {
		limit = defaultMaxPathUTF16Units
	}

	roots, err := manager.roots()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(roots, func(i, j int) bool {
		return sameVolume(roots[i], outputPath) && !sameVolume(roots[j], outputPath)
	})

	var rootErrors []error
	for _, candidate := range roots {
		workspace, err := createInRoot(candidate, limit, randomWorkspaceID)
		if err == nil {
			return workspace, nil
		}
		rootErrors = append(rootErrors, fmt.Errorf("workspace root %q: %w", candidate, err))
	}
	return nil, fmt.Errorf("no suitable short writable workspace is available: %w", errors.Join(rootErrors...))
}

func (manager Manager) roots() ([]string, error) {
	roots := manager.Roots
	if len(roots) == 0 {
		cache, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("find per-user cache directory: %w", err)
		}
		roots = []string{filepath.Join(cache, "vcx", "work")}
	}

	absRoots := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		absolute, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace root %q: %w", root, err)
		}
		absolute = filepath.Clean(absolute)
		key := absolute
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		absRoots = append(absRoots, absolute)
	}
	if len(absRoots) == 0 {
		return nil, fmt.Errorf("at least one per-user workspace root is required")
	}
	return absRoots, nil
}

func createInRoot(root string, limit int, nextID func() (string, error)) (*Workspace, error) {
	if !fitsWorkspacePath(root, limit) {
		return nil, fmt.Errorf("root leaves insufficient path budget for media-tool artifacts")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create per-user workspace root: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect per-user workspace root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("per-user workspace root is not a regular directory")
	}

	for attempt := 0; attempt < workspaceCreateAttempts; attempt++ {
		id, err := nextID()
		if err != nil {
			return nil, fmt.Errorf("generate workspace ID: %w", err)
		}
		if !validWorkspaceID(id) {
			return nil, fmt.Errorf("generated invalid workspace ID")
		}
		path := filepath.Join(root, workspacePrefix+id)
		if !fitsArtifactPath(path, "transcript.srt", limit) {
			return nil, fmt.Errorf("workspace path leaves insufficient path budget for artifacts")
		}
		if err := os.Mkdir(path, 0o700); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return nil, fmt.Errorf("create workspace directory: %w", err)
		}

		workspace := &Workspace{path: path, root: root, limit: limit}
		if err := workspace.probeWritable(); err != nil {
			if removeErr := workspace.Remove(); removeErr != nil {
				err = errors.Join(err, fmt.Errorf("remove unusable workspace: %w", removeErr))
			}
			return nil, err
		}
		return workspace, nil
	}
	return nil, fmt.Errorf("could not reserve a unique workspace ID after %d attempts", workspaceCreateAttempts)
}

func randomWorkspaceID() (string, error) {
	var data [workspaceIDBytes]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

func validWorkspaceID(id string) bool {
	if len(id) != workspaceIDBytes*2 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func sameVolume(first, second string) bool {
	firstVolume := filepath.VolumeName(first)
	secondVolume := filepath.VolumeName(second)
	if firstVolume == "" && secondVolume == "" {
		return runtime.GOOS != "windows"
	}
	return strings.EqualFold(firstVolume, secondVolume)
}

func fitsWorkspacePath(root string, limit int) bool {
	path := filepath.Join(root, workspacePrefix+strings.Repeat("0", workspaceIDBytes*2))
	return fitsArtifactPath(path, "transcript.srt", limit)
}

func fitsArtifactPath(workspacePath, artifactName string, limit int) bool {
	return utf16PathUnits(filepath.Join(workspacePath, artifactName)) <= limit
}

func utf16PathUnits(path string) int {
	return len(utf16.Encode([]rune(path)))
}

// Path returns the absolute workspace directory.
func (workspace *Workspace) Path() string {
	if workspace == nil {
		return ""
	}
	return workspace.path
}

// ArtifactPath returns a path for one of VCX's fixed short artifact names.
// Arbitrary names are rejected so callers cannot accidentally include source
// filenames or escape the workspace directory.
func (workspace *Workspace) ArtifactPath(name string) (string, error) {
	if workspace == nil || workspace.path == "" {
		return "", fmt.Errorf("workspace is required")
	}
	if _, ok := artifactNames[name]; !ok {
		return "", fmt.Errorf("unsupported workspace artifact name %q", name)
	}
	path := filepath.Join(workspace.path, name)
	if !fitsArtifactPath(workspace.path, name, workspace.limit) {
		return "", fmt.Errorf("workspace artifact path exceeds the configured path budget")
	}
	return path, nil
}

// Remove removes the workspace directory if it still exists. It refuses to
// remove a path that no longer matches the manager's direct-child workspace
// layout.
func (workspace *Workspace) Remove() error {
	if workspace == nil || workspace.path == "" || workspace.root == "" {
		return nil
	}
	base := filepath.Base(workspace.path)
	if filepath.Dir(workspace.path) != workspace.root || !strings.HasPrefix(base, workspacePrefix) ||
		!validWorkspaceID(strings.TrimPrefix(base, workspacePrefix)) {
		return fmt.Errorf("refuse to remove path outside the owned workspace layout")
	}
	info, err := os.Lstat(workspace.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect workspace before cleanup: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse to remove workspace that is not a regular directory")
	}
	if err := os.RemoveAll(workspace.path); err != nil {
		return fmt.Errorf("remove workspace: %w", err)
	}
	return nil
}

func (workspace *Workspace) probeWritable() error {
	name := filepath.Join(workspace.path, ".p")
	probe, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("verify workspace is writable: %w", err)
	}
	if err := probe.Close(); err != nil {
		if removeErr := os.Remove(name); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("remove workspace write probe: %w", removeErr))
		}
		return fmt.Errorf("close workspace write probe: %w", err)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove workspace write probe: %w", err)
	}
	return nil
}
