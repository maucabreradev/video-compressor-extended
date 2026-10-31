package jobmanifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	manifestSuffix   = ".json"
	maxManifestBytes = 32 << 20
)

// Store persists job manifests in a per-user directory. Manifests do not expire
// automatically; Remove deletes only one manifest after the caller coordinates
// job and workspace cleanup. Manifest operations are serialized within a Store;
// TryLock coordinates job execution across processes.
type Store struct {
	root string
	mu   sync.Mutex
}

// NewStore opens a manifest store. An empty root selects the local per-user
// cache directory so job metadata does not roam with a Windows profile. An
// explicit root must be a dedicated directory with per-user access controls.
func NewStore(root string) (*Store, error) {
	if root == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("find per-user job storage: %w", err)
		}
		if strings.TrimSpace(cache) == "" {
			return nil, fmt.Errorf("per-user cache directory is empty")
		}
		root = filepath.Join(cache, "vcx", "jobs")
	}
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("job storage path must not be empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve job storage path %q: %w", root, err)
	}
	root = filepath.Clean(absolute)
	if filepath.Dir(root) == root {
		return nil, fmt.Errorf("job storage path must be a dedicated subdirectory, not a volume root")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create per-user job storage: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect per-user job storage: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("per-user job storage is not a regular directory")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("per-user job storage permissions must not allow group or other access")
	}
	return &Store{root: root}, nil
}

func (store *Store) ready() error {
	if store == nil || store.root == "" {
		return fmt.Errorf("job manifest store is not initialized")
	}
	return nil
}

// Create atomically writes a new manifest without replacing an existing job.
func (store *Store) Create(manifest Manifest) error {
	if err := store.ready(); err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("validate job manifest before creation: %w", err)
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	target := store.manifestPath(manifest.JobID)
	temporary, err := writeTemporaryManifest(store.root, manifest)
	if err != nil {
		return err
	}
	if err := os.Link(temporary, target); err != nil {
		cleanupErr := removeTemporaryFile(temporary)
		return fmt.Errorf("publish new job manifest without replacement: %w", errors.Join(err, cleanupErr))
	}
	if err := os.Remove(temporary); err != nil {
		return fmt.Errorf("job manifest was created but its staging link could not be removed: %w", err)
	}
	return nil
}

// Load reads and validates a persisted manifest.
func (store *Store) Load(jobID string) (Manifest, error) {
	if err := store.ready(); err != nil {
		return Manifest{}, err
	}
	if !validJobID(jobID) {
		return Manifest{}, fmt.Errorf("invalid job ID")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.loadUnlocked(jobID)
}

// Update applies one in-process serialized mutation and atomically replaces the
// manifest. The job ID, schema version, and creation time cannot be changed.
func (store *Store) Update(jobID string, update func(*Manifest) error) error {
	if err := store.ready(); err != nil {
		return err
	}
	if !validJobID(jobID) {
		return fmt.Errorf("invalid job ID")
	}
	if update == nil {
		return fmt.Errorf("manifest update function is required")
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	current, err := store.loadUnlocked(jobID)
	if err != nil {
		return err
	}
	updated := current
	updated.Items = append([]Item(nil), current.Items...)
	if err := update(&updated); err != nil {
		return fmt.Errorf("update job manifest: %w", err)
	}
	if updated.JobID != current.JobID || updated.SchemaVersion != current.SchemaVersion ||
		!updated.CreatedAt.Equal(current.CreatedAt) || !sameJobDefinition(current, updated) {
		return fmt.Errorf("job ID, schema version, creation time, inputs, destinations, and processing identity are immutable")
	}
	updated.UpdatedAt = time.Now().UTC()
	if !updated.UpdatedAt.After(current.UpdatedAt) {
		updated.UpdatedAt = current.UpdatedAt.Add(time.Nanosecond)
	}
	if err := updated.Validate(); err != nil {
		return fmt.Errorf("validate updated job manifest: %w", err)
	}

	temporary, err := writeTemporaryManifest(store.root, updated)
	if err != nil {
		return err
	}
	target := store.manifestPath(jobID)
	if err := os.Rename(temporary, target); err != nil {
		cleanupErr := removeTemporaryFile(temporary)
		return fmt.Errorf("atomically replace job manifest: %w", errors.Join(err, cleanupErr))
	}
	return nil
}

func sameJobDefinition(first, second Manifest) bool {
	if first.Identity != second.Identity || len(first.Items) != len(second.Items) {
		return false
	}
	for index, item := range first.Items {
		other := second.Items[index]
		if item.SourcePath != other.SourcePath || item.SourceFingerprint != other.SourceFingerprint ||
			item.VideoDestination != other.VideoDestination || item.SubtitleDestination != other.SubtitleDestination {
			return false
		}
	}
	return true
}

// Remove deletes only the manifest file. It never removes workspace artifacts,
// source media, or final outputs; callers must coordinate job lifecycle cleanup.
func (store *Store) Remove(jobID string) error {
	if err := store.ready(); err != nil {
		return err
	}
	if !validJobID(jobID) {
		return fmt.Errorf("invalid job ID")
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	path := store.manifestPath(jobID)
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect job manifest before removal: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse to remove a job manifest that is not a regular file")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove job manifest: %w", err)
	}
	return nil
}

func (store *Store) loadUnlocked(jobID string) (Manifest, error) {
	path := store.manifestPath(jobID)
	info, err := os.Lstat(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("inspect job manifest %q: %w", jobID, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Manifest{}, fmt.Errorf("job manifest %q is not a regular file", jobID)
	}
	if info.Size() > maxManifestBytes {
		return Manifest{}, fmt.Errorf("job manifest %q exceeds the %d-byte limit", jobID, maxManifestBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open job manifest %q: %w", jobID, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return Manifest{}, fmt.Errorf("read job manifest %q: %w", jobID, readErr)
	}
	if closeErr != nil {
		return Manifest{}, fmt.Errorf("close job manifest %q: %w", jobID, closeErr)
	}
	if len(data) > maxManifestBytes {
		return Manifest{}, fmt.Errorf("job manifest %q exceeds the %d-byte limit", jobID, maxManifestBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse job manifest %q: %w", jobID, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return Manifest{}, fmt.Errorf("parse trailing data in job manifest %q: %w", jobID, err)
	}
	if manifest.JobID != jobID {
		return Manifest{}, fmt.Errorf("job manifest filename and job ID do not match")
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("validate loaded job manifest %q: %w", jobID, err)
	}
	return manifest, nil
}

func (store *Store) manifestPath(jobID string) string {
	return filepath.Join(store.root, jobID+manifestSuffix)
}

func writeTemporaryManifest(root string, manifest Manifest) (string, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode job manifest: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxManifestBytes {
		return "", fmt.Errorf("job manifest exceeds the %d-byte limit", maxManifestBytes)
	}
	file, err := os.CreateTemp(root, ".manifest-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create staged job manifest: %w", err)
	}
	path := file.Name()
	if _, err := file.Write(data); err != nil {
		closeErr := file.Close()
		cleanupErr := removeTemporaryFile(path)
		return "", fmt.Errorf("write staged job manifest: %w", errors.Join(err, closeErr, cleanupErr))
	}
	if err := file.Sync(); err != nil {
		closeErr := file.Close()
		cleanupErr := removeTemporaryFile(path)
		return "", fmt.Errorf("sync staged job manifest: %w", errors.Join(err, closeErr, cleanupErr))
	}
	if err := file.Close(); err != nil {
		cleanupErr := removeTemporaryFile(path)
		return "", fmt.Errorf("close staged job manifest: %w", errors.Join(err, cleanupErr))
	}
	return path, nil
}

func removeTemporaryFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove staged manifest file: %w", err)
	}
	return nil
}
