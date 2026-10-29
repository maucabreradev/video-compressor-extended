package runtimeassets

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestManagerUsesVerifiedBundledModelWithoutConsent(t *testing.T) {
	payload := []byte("model bytes")
	manifest, asset := managerManifest("whisper-model", "model", "ggml-base.bin", payload)
	temp := t.TempDir()
	bundleRoot := filepath.Join(temp, "VCX Bundled Assets ü")
	cacheRoot := filepath.Join(temp, "User Cache With Spaces")
	if err := os.MkdirAll(bundleRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleRoot, asset.Filename), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	manager := Manager{Manifest: manifest, Root: cacheRoot, BundleRoot: bundleRoot}

	prepared, err := manager.Prepare(context.Background(), asset.ID, nil, nil)
	if err != nil {
		t.Fatalf("prepare bundled model without consent: %v", err)
	}
	got, err := os.ReadFile(prepared.Files["model"])
	if err != nil {
		t.Fatalf("read prepared model: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("prepared model = %q, want %q", got, payload)
	}
	if !strings.Contains(prepared.Files["model"], "Cache With Spaces") {
		t.Fatalf("prepared model path does not use the configured cache: %q", prepared.Files["model"])
	}
}

func TestManagerDownloadsOnlyAfterConsentAndThenUsesCache(t *testing.T) {
	payload := []byte("model bytes from official source")
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		_, _ = writer.Write(payload)
	}))
	defer server.Close()
	manifest, asset := managerManifest("whisper-model", "model", "ggml-base.bin", payload)
	asset.DownloadURL = server.URL + "/model"
	manifest.Assets[0] = asset
	manager := Manager{Manifest: manifest, Root: filepath.Join(t.TempDir(), "VCX cache ü"), Client: server.Client()}
	var consentCalls int
	var progressCalls int
	prepared, err := manager.Prepare(context.Background(), asset.ID, func(request Asset) bool {
		consentCalls++
		if request.ID != asset.ID || request.SizeBytes != int64(len(payload)) {
			t.Errorf("consent request metadata = %#v", request)
		}
		return true
	}, func(progress DownloadProgress) {
		progressCalls++
		if progress.TotalBytes != int64(len(payload)) {
			t.Errorf("progress total = %d, want %d", progress.TotalBytes, len(payload))
		}
	})
	if err != nil {
		t.Fatalf("download managed model: %v", err)
	}
	if consentCalls != 1 || requests.Load() != 1 || progressCalls == 0 {
		t.Fatalf("consent calls=%d requests=%d progress callbacks=%d", consentCalls, requests.Load(), progressCalls)
	}
	cached, err := manager.Prepare(context.Background(), asset.ID, func(Asset) bool {
		t.Fatal("cached assets must not prompt for consent")
		return false
	}, nil)
	if err != nil {
		t.Fatalf("prepare cached model: %v", err)
	}
	if cached.Files["model"] != prepared.Files["model"] || requests.Load() != 1 {
		t.Fatalf("cache result changed: first=%q second=%q requests=%d", prepared.Files["model"], cached.Files["model"], requests.Load())
	}
	if err := os.WriteFile(cached.Files["model"], bytes.Repeat([]byte("x"), len(payload)), 0o600); err != nil {
		t.Fatalf("corrupt cached model for integrity check: %v", err)
	}
	if _, err := manager.Prepare(context.Background(), asset.ID, nil, nil); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("Prepare() error = %v, want corrupted-cache SHA-256 mismatch", err)
	}
}

func TestManagerRejectsDeclinedDownloadBeforeRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()
	manifest, asset := managerManifest("whisper-model", "model", "ggml-base.bin", []byte("model"))
	asset.DownloadURL = server.URL
	manifest.Assets[0] = asset
	manager := Manager{Manifest: manifest, Root: t.TempDir(), Client: server.Client()}

	_, err := manager.Prepare(context.Background(), asset.ID, func(Asset) bool { return false }, nil)
	if err == nil || !strings.Contains(err.Error(), "consent") {
		t.Fatalf("Prepare() error = %v, want consent error", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("download requests = %d, want 0", requests.Load())
	}
}

func TestManagerRejectsDigestMismatchWithoutPublishing(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "tampered payload")
	}))
	defer server.Close()
	manifest, asset := managerManifest("whisper-model", "model", "ggml-base.bin", []byte("expected payload"))
	asset.DownloadURL = server.URL
	manifest.Assets[0] = asset
	root := filepath.Join(t.TempDir(), "VCX cache")
	manager := Manager{Manifest: manifest, Root: root, Client: server.Client()}

	_, err := manager.Prepare(context.Background(), asset.ID, func(Asset) bool { return true }, nil)
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("Prepare() error = %v, want size or digest mismatch", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, asset.ID, asset.Revision)); !os.IsNotExist(statErr) {
		t.Fatalf("failed download left a published cache entry: stat error = %v", statErr)
	}
}

func TestManagerExtractsArchiveAndRejectsTraversal(t *testing.T) {
	executableBytes := []byte("executable bytes")
	archive := zipBytes(t, map[string]string{
		"ffmpeg-9/bin/ffmpeg.exe": string(executableBytes),
		"ffmpeg-9/bin/unused.exe": "not installed",
	})
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(archive)
	}))
	defer server.Close()
	manifest, asset := managerManifest("ffmpeg", "archive", "ffmpeg.zip", archive)
	asset.DownloadURL = server.URL
	asset.Files = map[string]AssetFile{"ffmpeg": assetFile("ffmpeg-9/bin/ffmpeg.exe", executableBytes)}
	manifest.Assets[0] = asset
	manager := Manager{Manifest: manifest, Root: t.TempDir(), Client: server.Client()}
	prepared, err := manager.Prepare(context.Background(), asset.ID, func(Asset) bool { return true }, nil)
	if err != nil {
		t.Fatalf("prepare FFmpeg archive: %v", err)
	}
	got, err := os.ReadFile(prepared.Files["ffmpeg"])
	if err != nil || string(got) != "executable bytes" {
		t.Fatalf("extracted tool bytes = %q, error = %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(prepared.Root, "ffmpeg-9", "bin", "unused.exe")); !os.IsNotExist(err) {
		t.Fatalf("unmanifested archive executable should not be extracted: %v", err)
	}
	if err := os.WriteFile(prepared.Files["ffmpeg"], bytes.Repeat([]byte("x"), len(got)), 0o600); err != nil {
		t.Fatalf("corrupt extracted executable for cache verification: %v", err)
	}
	if _, err := manager.Prepare(context.Background(), asset.ID, nil, nil); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("Prepare() error = %v, want extracted-file SHA-256 mismatch", err)
	}

	traversal := zipBytes(t, map[string]string{"../outside.exe": "must not escape"})
	badServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(traversal)
	}))
	defer badServer.Close()
	badManifest, badAsset := managerManifest("unsafe-archive", "archive", "unsafe.zip", traversal)
	badAsset.DownloadURL = badServer.URL
	badAsset.Files = map[string]AssetFile{"tool": assetFile("outside.exe", []byte("must not escape"))}
	badManifest.Assets[0] = badAsset
	badRoot := t.TempDir()
	badManager := Manager{Manifest: badManifest, Root: badRoot, Client: badServer.Client()}
	_, err = badManager.Prepare(context.Background(), badAsset.ID, func(Asset) bool { return true }, nil)
	if err == nil || !strings.Contains(err.Error(), "unsafe archive path") {
		t.Fatalf("Prepare() error = %v, want unsafe archive path", err)
	}
	if _, statErr := os.Stat(filepath.Join(badRoot, "unsafe-archive", badAsset.Revision, "outside.exe")); !os.IsNotExist(statErr) {
		t.Fatalf("archive traversal escaped the extraction root: %v", statErr)
	}
}

func TestManagerCancellationAndOfflineFailureDoNotPublish(t *testing.T) {
	payload := []byte("model bytes")
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if requests.Add(1) == 1 {
			writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(payload[:2])
			writer.(http.Flusher).Flush()
			<-request.Context().Done()
			return
		}
		_, _ = writer.Write(payload)
	}))
	defer server.Close()
	manifest, asset := managerManifest("whisper-model", "model", "ggml-base.bin", payload)
	root := t.TempDir()
	asset.DownloadURL = server.URL
	manifest.Assets[0] = asset
	manager := Manager{Manifest: manifest, Root: root, Client: server.Client()}
	cancelled, cancel := context.WithCancel(context.Background())
	progressed := make(chan struct{}, 1)
	result := make(chan error, 1)
	go func() {
		_, err := manager.Prepare(cancelled, asset.ID, func(Asset) bool { return true }, func(DownloadProgress) {
			select {
			case progressed <- struct{}{}:
			default:
			}
		})
		result <- err
	}()
	select {
	case <-progressed:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("download did not report progress before cancellation")
	}
	cancel()
	err := <-result
	if err == nil {
		t.Fatal("Prepare() succeeded after download cancellation")
	}
	assetRoot := filepath.Join(root, asset.ID, asset.Revision)
	if _, statErr := os.Stat(assetRoot); !os.IsNotExist(statErr) {
		t.Fatalf("cancelled download left a cache entry: %v", statErr)
	}
	prepared, err := manager.Prepare(context.Background(), asset.ID, func(Asset) bool { return true }, nil)
	if err != nil {
		t.Fatalf("retry after cancelled download: %v", err)
	}
	if requests.Load() != 2 {
		t.Fatalf("request count after retry = %d, want 2", requests.Load())
	}
	if _, err := os.Stat(prepared.Files["model"]); err != nil {
		t.Fatalf("retried model is missing: %v", err)
	}
}

func TestManagerOfflineSourceDoesNotPublish(t *testing.T) {
	payload := []byte("model bytes")
	manifest, asset := managerManifest("whisper-model", "model", "ggml-base.bin", payload)
	server := httptest.NewTLSServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	asset.DownloadURL = url
	manifest.Assets[0] = asset
	root := t.TempDir()
	manager := Manager{Manifest: manifest, Root: root, Client: server.Client()}
	_, err := manager.Prepare(context.Background(), asset.ID, func(Asset) bool { return true }, nil)
	if err == nil {
		t.Fatal("Prepare() succeeded while the source server was offline")
	}
	if _, statErr := os.Stat(filepath.Join(root, asset.ID, asset.Revision)); !os.IsNotExist(statErr) {
		t.Fatalf("offline download left a cache entry: %v", statErr)
	}
}

func managerManifest(id, kind, filename string, payload []byte) (Manifest, Asset) {
	digest := sha256.Sum256(payload)
	asset := Asset{
		ID:          id,
		Version:     "test-1.0.0",
		Revision:    strings.Repeat("a", 40),
		Target:      "windows-amd64",
		Kind:        kind,
		Filename:    filename,
		DownloadURL: "https://downloads.example/" + filename,
		SourceURL:   "https://github.com/example/project",
		SHA256:      hex.EncodeToString(digest[:]),
		SizeBytes:   int64(len(payload)),
		License:     "MIT",
		Files:       map[string]AssetFile{"model": assetFile(filename, payload)},
	}
	return Manifest{SchemaVersion: 1, Assets: []Asset{asset}}, asset
}

func assetFile(path string, contents []byte) AssetFile {
	digest := sha256.Sum256(contents)
	return AssetFile{Path: path, SHA256: hex.EncodeToString(digest[:]), SizeBytes: int64(len(contents))}
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, contents := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create ZIP entry %q: %v", name, err)
		}
		if _, err := io.WriteString(file, contents); err != nil {
			t.Fatalf("write ZIP entry %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close ZIP archive: %v", err)
	}
	return buffer.Bytes()
}
