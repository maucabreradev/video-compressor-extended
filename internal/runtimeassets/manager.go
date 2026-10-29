package runtimeassets

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const defaultMaxExpandedBytes int64 = 4 << 30

// Manager resolves verified bundled or cached runtime assets and downloads
// missing assets only after the caller obtains user consent.
type Manager struct {
	Manifest         Manifest
	Root             string
	BundleRoot       string
	Client           *http.Client
	MaxExpandedBytes int64
}

// DownloadProgress reports bytes received for an asset acquisition.
type DownloadProgress struct {
	AssetID       string
	ReceivedBytes int64
	TotalBytes    int64
}

// PreparedAsset contains the verified root and absolute paths for the
// manifest's logical files.
type PreparedAsset struct {
	Asset Asset
	Root  string
	Files map[string]string
}

// Prepare resolves an installed asset or acquires and prepares it after the
// consent callback approves the full source, version, and size metadata. The
// caller is responsible for presenting those details; a nil or false callback
// declines the download.
func (manager Manager) Prepare(
	ctx context.Context,
	id string,
	consent func(Asset) bool,
	onProgress func(DownloadProgress),
) (PreparedAsset, error) {
	if ctx == nil {
		return PreparedAsset{}, fmt.Errorf("asset setup context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return PreparedAsset{}, err
	}
	manifest := manager.Manifest
	if len(manifest.Assets) == 0 {
		var err error
		manifest, err = Default()
		if err != nil {
			return PreparedAsset{}, err
		}
	}
	if err := manifest.Validate(); err != nil {
		return PreparedAsset{}, fmt.Errorf("validate runtime manifest: %w", err)
	}
	asset, ok := manifestAsset(manifest, id)
	if !ok {
		return PreparedAsset{}, fmt.Errorf("runtime asset %q is not in the manifest", id)
	}
	root, err := manager.cacheRoot()
	if err != nil {
		return PreparedAsset{}, err
	}
	assetRoot := filepath.Join(root, asset.ID, asset.Revision)
	if _, err := os.Lstat(assetRoot); err == nil {
		prepared, err := loadPrepared(asset, assetRoot)
		if err != nil {
			return PreparedAsset{}, fmt.Errorf("validate cached asset %q: %w", id, err)
		}
		return prepared, nil
	} else if !os.IsNotExist(err) {
		return PreparedAsset{}, fmt.Errorf("inspect cached asset %q: %w", id, err)
	}

	var bundledPath string
	if manager.BundleRoot != "" {
		candidate := filepath.Join(manager.BundleRoot, asset.Filename)
		if info, err := os.Lstat(candidate); err == nil {
			if !info.Mode().IsRegular() {
				return PreparedAsset{}, fmt.Errorf("bundled asset %q is not a regular file", id)
			}
			bundledPath = candidate
		} else if !os.IsNotExist(err) {
			return PreparedAsset{}, fmt.Errorf("inspect bundled asset %q: %w", id, err)
		}
	}

	if bundledPath == "" && (consent == nil || !consent(asset)) {
		return PreparedAsset{}, fmt.Errorf(
			"runtime asset %q version %s is not cached; download from %s (%d bytes) requires user consent",
			asset.ID, asset.Version, asset.DownloadURL, asset.SizeBytes,
		)
	}
	if err := ctx.Err(); err != nil {
		return PreparedAsset{}, err
	}

	parent := filepath.Dir(assetRoot)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return PreparedAsset{}, fmt.Errorf("create runtime asset cache: %w", err)
	}
	stage, err := os.MkdirTemp(parent, ".vcx-install-")
	if err != nil {
		return PreparedAsset{}, fmt.Errorf("create runtime asset staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	artifactPath := filepath.Join(stage, asset.Filename)
	if bundledPath != "" {
		if err := copyVerifiedBundle(ctx, bundledPath, artifactPath, asset); err != nil {
			return PreparedAsset{}, fmt.Errorf("prepare bundled asset %q: %w", id, err)
		}
	} else if err := manager.download(ctx, artifactPath, asset, onProgress); err != nil {
		return PreparedAsset{}, fmt.Errorf("download runtime asset %q: %w", id, err)
	}

	contentRoot := stage
	if asset.Kind == "archive" {
		contentRoot = filepath.Join(stage, "content")
		if err := extractArchive(ctx, artifactPath, contentRoot, manager.maxExpandedBytes(), asset.Files); err != nil {
			return PreparedAsset{}, fmt.Errorf("extract runtime asset %q: %w", id, err)
		}
	}
	_, err = resolvePrepared(asset, contentRoot)
	if err != nil {
		return PreparedAsset{}, fmt.Errorf("validate prepared asset %q: %w", id, err)
	}
	if err := ctx.Err(); err != nil {
		return PreparedAsset{}, err
	}
	if err := os.Rename(stage, assetRoot); err != nil {
		if _, statErr := os.Lstat(assetRoot); statErr == nil {
			cached, cacheErr := loadPrepared(asset, assetRoot)
			if cacheErr == nil {
				return cached, nil
			}
		}
		return PreparedAsset{}, fmt.Errorf("publish runtime asset %q: %w", id, err)
	}
	return loadPrepared(asset, assetRoot)
}

func (manager Manager) cacheRoot() (string, error) {
	root := manager.Root
	if root == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("find per-user cache directory: %w", err)
		}
		root = filepath.Join(cache, "vcx", "runtime")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve runtime asset cache path: %w", err)
	}
	return absolute, nil
}

func (manager Manager) maxExpandedBytes() int64 {
	if manager.MaxExpandedBytes > 0 {
		return manager.MaxExpandedBytes
	}
	return defaultMaxExpandedBytes
}

func (manager Manager) download(
	ctx context.Context,
	destination string,
	asset Asset,
	onProgress func(DownloadProgress),
) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	client := manager.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request asset source: %w", err)
	}
	defer response.Body.Close()
	if response.Request == nil || response.Request.URL == nil || response.Request.URL.Scheme != "https" {
		return fmt.Errorf("asset source redirected away from HTTPS")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("asset source returned HTTP %s", response.Status)
	}
	if response.ContentLength >= 0 && response.ContentLength != asset.SizeBytes {
		return fmt.Errorf("asset source size mismatch: server reports %d bytes, manifest expects %d", response.ContentLength, asset.SizeBytes)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create staged download: %w", err)
	}
	reader := &progressReader{
		reader: io.LimitReader(response.Body, asset.SizeBytes+1),
		ctx:    ctx,
		asset:  asset,
		update: onProgress,
	}
	_, copyErr := io.Copy(file, reader)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("write staged download: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close staged download: %w", closeErr)
	}
	return verifyFile(asset, destination)
}

type progressReader struct {
	reader   io.Reader
	ctx      context.Context
	asset    Asset
	update   func(DownloadProgress)
	received int64
}

func (reader *progressReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.reader.Read(buffer)
	if count > 0 {
		reader.received += int64(count)
		if reader.update != nil {
			reader.update(DownloadProgress{
				AssetID: reader.asset.ID, ReceivedBytes: reader.received,
				TotalBytes: reader.asset.SizeBytes,
			})
		}
	}
	return count, err
}

func copyVerifiedBundle(ctx context.Context, sourcePath, destination string, asset Asset) error {
	if err := verifyFile(asset, sourcePath); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open bundled file: %w", err)
	}
	defer source.Close()
	destinationFile, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create staged bundle copy: %w", err)
	}
	_, copyErr := io.Copy(destinationFile, contextReader{ctx: ctx, reader: source})
	closeErr := destinationFile.Close()
	if copyErr != nil {
		return fmt.Errorf("copy bundled file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close staged bundle copy: %w", closeErr)
	}
	return verifyFile(asset, destination)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

func extractArchive(
	ctx context.Context,
	archivePath string,
	destination string,
	maxExpandedBytes int64,
	files map[string]AssetFile,
) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open ZIP archive: %w", err)
	}
	defer archive.Close()
	var expandedBytes uint64
	for _, entry := range archive.File {
		if _, err := safeArchivePath(entry.Name); err != nil {
			return err
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive contains a symbolic link %q", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if entry.UncompressedSize64 > uint64(maxExpandedBytes) ||
			expandedBytes > uint64(maxExpandedBytes)-entry.UncompressedSize64 {
			return fmt.Errorf("archive expands beyond the %d-byte safety limit", maxExpandedBytes)
		}
		expandedBytes += entry.UncompressedSize64
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	root, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve extraction directory: %w", err)
	}
	wantedFiles := make(map[string]struct{}, len(files))
	for _, file := range files {
		wantedFiles[file.Path] = struct{}{}
	}
	for _, entry := range archive.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name, err := safeArchivePath(entry.Name)
		if err != nil {
			return err
		}
		target, err := containedPath(root, name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if _, wanted := wantedFiles[name]; !wanted {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("create parent for archive file %q: %w", entry.Name, err)
		}
		input, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open archive file %q: %w", entry.Name, err)
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			input.Close()
			return fmt.Errorf("create archive file %q: %w", entry.Name, err)
		}
		written, copyErr := io.CopyN(output, contextReader{ctx: ctx, reader: input}, int64(entry.UncompressedSize64))
		inputCloseErr := input.Close()
		outputCloseErr := output.Close()
		if copyErr != nil {
			return fmt.Errorf("extract archive file %q after %d bytes: %w", entry.Name, written, copyErr)
		}
		if inputCloseErr != nil {
			return fmt.Errorf("close archive file %q: %w", entry.Name, inputCloseErr)
		}
		if outputCloseErr != nil {
			return fmt.Errorf("close extracted file %q: %w", entry.Name, outputCloseErr)
		}
	}
	return nil
}

func safeArchivePath(name string) (string, error) {
	if name == "" || strings.Contains(name, `\`) || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	trimmed := strings.TrimSuffix(name, "/")
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(trimmed)))
	if cleaned != trimmed || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return cleaned, nil
}

func containedPath(root, relative string) (string, error) {
	target := filepath.Join(root, filepath.FromSlash(relative))
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("resolve archive path %q: %w", relative, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("unsafe archive path %q", relative)
	}
	return target, nil
}

func manifestAsset(manifest Manifest, id string) (Asset, bool) {
	for _, asset := range manifest.Assets {
		if asset.ID == id {
			return asset, true
		}
	}
	return Asset{}, false
}

func verifyFile(asset Asset, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open asset file: %w", err)
	}
	defer file.Close()
	return asset.Verify(file)
}

func loadPrepared(asset Asset, assetRoot string) (PreparedAsset, error) {
	artifactPath := filepath.Join(assetRoot, asset.Filename)
	if err := verifyFile(asset, artifactPath); err != nil {
		return PreparedAsset{}, err
	}
	contentRoot := assetRoot
	if asset.Kind == "archive" {
		contentRoot = filepath.Join(assetRoot, "content")
	}
	return resolvePrepared(asset, contentRoot)
}

func resolvePrepared(asset Asset, contentRoot string) (PreparedAsset, error) {
	files := make(map[string]string, len(asset.Files))
	for name, file := range asset.Files {
		target, err := containedPath(contentRoot, file.Path)
		if err != nil {
			return PreparedAsset{}, err
		}
		info, err := os.Lstat(target)
		if err != nil {
			return PreparedAsset{}, fmt.Errorf("inspect prepared file %q: %w", name, err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return PreparedAsset{}, fmt.Errorf("prepared file %q is not a non-empty regular file", name)
		}
		if err := verifyFile(Asset{SHA256: file.SHA256, SizeBytes: file.SizeBytes}, target); err != nil {
			return PreparedAsset{}, fmt.Errorf("verify prepared file %q: %w", name, err)
		}
		files[name] = target
	}
	return PreparedAsset{Asset: asset, Root: contentRoot, Files: files}, nil
}
