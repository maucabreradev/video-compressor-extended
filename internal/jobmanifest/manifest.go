// Package jobmanifest stores durable, validated metadata for resumable media jobs.
package jobmanifest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CurrentSchemaVersion is the only manifest schema this build can read and write.
const CurrentSchemaVersion = 1

// StageState is the persisted lifecycle state of one processing stage.
type StageState string

// StageState values persist whether work is pending, active, safely staged,
// published, failed, cancelled, or skipped because a validated output was reused.
const (
	StagePending   StageState = "pending"
	StageRunning   StageState = "running"
	StageStaged    StageState = "staged"
	StagePublished StageState = "published"
	StageFailed    StageState = "failed"
	StageCancelled StageState = "cancelled"
	StageSkipped   StageState = "skipped"
)

// ProcessingIdentity contains output-affecting options and the exact runtime
// assets needed to decide whether a prior result is reusable.
type ProcessingIdentity struct {
	CompressionEnabled bool   `json:"compression_enabled"`
	SubtitlesEnabled   bool   `json:"subtitles_enabled"`
	VideoBackend       string `json:"video_backend,omitempty"`
	VideoCodec         string `json:"video_codec,omitempty"`
	MaxWidth           int    `json:"max_width,omitempty"`
	MaxHeight          int    `json:"max_height,omitempty"`
	FrameRate          int    `json:"frame_rate,omitempty"`
	Language           string `json:"language,omitempty"`
	WhisperThreads     int    `json:"whisper_threads,omitempty"`
	WhisperGPU         bool   `json:"whisper_gpu,omitempty"`
	FFmpegRevision     string `json:"ffmpeg_revision,omitempty"`
	WhisperRevision    string `json:"whisper_revision,omitempty"`
	ModelSHA256        string `json:"model_sha256,omitempty"`
}

// SourceFingerprint avoids hashing large media files. Together with the
// normalized source path, it checks byte length and modification time. Content
// changes that preserve both metadata values are outside this fingerprint's
// guarantees.
type SourceFingerprint struct {
	SizeBytes  int64     `json:"size_bytes"`
	ModifiedAt time.Time `json:"modified_at"`
}

// StageRecord maps one stage to its state and optional workspace artifact.
type StageRecord struct {
	State        StageState `json:"state"`
	WorkspaceID  string     `json:"workspace_id,omitempty"`
	ArtifactName string     `json:"artifact_name,omitempty"`
}

// Item records the source, destinations, fingerprint, and independent stage
// state for one input file. Full source paths, rather than basenames, distinguish
// files with identical names in different directories.
type Item struct {
	SourcePath          string            `json:"source_path"`
	SourceFingerprint   SourceFingerprint `json:"source_fingerprint"`
	VideoDestination    string            `json:"video_destination"`
	SubtitleDestination string            `json:"subtitle_destination"`
	Compression         StageRecord       `json:"compression"`
	Subtitles           StageRecord       `json:"subtitles"`
}

// Manifest is a versioned snapshot of a job suitable for recovery by another
// VCX process running as the same user.
type Manifest struct {
	SchemaVersion int                `json:"schema_version"`
	JobID         string             `json:"job_id"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Identity      ProcessingIdentity `json:"processing_identity"`
	Items         []Item             `json:"items"`
}

// NewManifest creates a manifest with a random job identifier and pending stage
// states. It normalizes all paths but does not hash the source media contents.
func NewManifest(identity ProcessingIdentity, items []Item) (Manifest, error) {
	jobID, err := newJobID()
	if err != nil {
		return Manifest{}, fmt.Errorf("generate job ID: %w", err)
	}
	now := time.Now().UTC()
	manifest := Manifest{
		SchemaVersion: CurrentSchemaVersion,
		JobID:         jobID,
		CreatedAt:     now,
		UpdatedAt:     now,
		Identity:      identity,
		Items:         append([]Item(nil), items...),
	}
	for index := range manifest.Items {
		item := &manifest.Items[index]
		for _, field := range []struct {
			label string
			value *string
		}{
			{label: "source", value: &item.SourcePath},
			{label: "video", value: &item.VideoDestination},
			{label: "subtitle", value: &item.SubtitleDestination},
		} {
			*field.value, err = normalizePath(*field.value)
			if err != nil {
				return Manifest{}, fmt.Errorf("normalize %s path for item %d: %w", field.label, index, err)
			}
		}
		if item.Compression.State == "" {
			item.Compression.State = StagePending
		}
		if item.Subtitles.State == "" {
			item.Subtitles.State = StagePending
		}
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("validate new job manifest: %w", err)
	}
	return manifest, nil
}

// NewItem normalizes the paths and captures source metadata for one manifest
// entry. It does not read or hash the media contents.
func NewItem(sourcePath, videoDestination, subtitleDestination string) (Item, error) {
	sourcePath, err := normalizePath(sourcePath)
	if err != nil {
		return Item{}, fmt.Errorf("normalize source path: %w", err)
	}
	videoDestination, err = normalizePath(videoDestination)
	if err != nil {
		return Item{}, fmt.Errorf("normalize video destination: %w", err)
	}
	subtitleDestination, err = normalizePath(subtitleDestination)
	if err != nil {
		return Item{}, fmt.Errorf("normalize subtitle destination: %w", err)
	}
	fingerprint, err := CaptureSourceFingerprint(sourcePath)
	if err != nil {
		return Item{}, err
	}
	return Item{
		SourcePath:          sourcePath,
		SourceFingerprint:   fingerprint,
		VideoDestination:    videoDestination,
		SubtitleDestination: subtitleDestination,
		Compression:         StageRecord{State: StagePending},
		Subtitles:           StageRecord{State: StagePending},
	}, nil
}

// CaptureSourceFingerprint records file size and modification time, without
// reading the entire media file.
func CaptureSourceFingerprint(path string) (SourceFingerprint, error) {
	info, err := os.Stat(path)
	if err != nil {
		return SourceFingerprint{}, fmt.Errorf("inspect source for job fingerprint %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return SourceFingerprint{}, fmt.Errorf("source for job fingerprint %q is not a regular file", path)
	}
	return SourceFingerprint{SizeBytes: info.Size(), ModifiedAt: info.ModTime().UTC()}, nil
}

// Matches reports whether the size and modification time still match a source
// file. It intentionally does not compute a content digest.
func (fingerprint SourceFingerprint) Matches(path string) (bool, error) {
	current, err := CaptureSourceFingerprint(path)
	if err != nil {
		return false, err
	}
	return fingerprint.SizeBytes == current.SizeBytes && fingerprint.ModifiedAt.Equal(current.ModifiedAt), nil
}

// MatchesSource checks both the normalized path and file metadata before a
// staged artifact is considered reusable.
func (item Item) MatchesSource(path string) (bool, error) {
	normalized, err := normalizePath(path)
	if err != nil {
		return false, err
	}
	if pathKey(normalized) != pathKey(item.SourcePath) {
		return false, nil
	}
	return item.SourceFingerprint.Matches(normalized)
}

// Matches reports whether all output-affecting options and runtime identities
// are identical.
func (identity ProcessingIdentity) Matches(other ProcessingIdentity) bool {
	return identity == other
}

// Validate checks the schema and every path and stage reference before a
// manifest is persisted or trusted after loading.
func (manifest Manifest) Validate() error {
	if manifest.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported job manifest schema version %d", manifest.SchemaVersion)
	}
	if !validJobID(manifest.JobID) {
		return fmt.Errorf("job manifest has an invalid job ID")
	}
	if manifest.CreatedAt.IsZero() || manifest.UpdatedAt.IsZero() || manifest.UpdatedAt.Before(manifest.CreatedAt) {
		return fmt.Errorf("job manifest timestamps are invalid")
	}
	if err := manifest.Identity.validate(); err != nil {
		return fmt.Errorf("validate processing identity: %w", err)
	}
	if len(manifest.Items) == 0 {
		return fmt.Errorf("job manifest must contain at least one input")
	}

	sources := make(map[string]int, len(manifest.Items))
	destinations := make(map[string]int, len(manifest.Items)*2)
	workspaceOwners := make(map[string]struct{})
	for index, item := range manifest.Items {
		for _, field := range []struct {
			name string
			path string
		}{
			{name: "source", path: item.SourcePath},
			{name: "video destination", path: item.VideoDestination},
			{name: "subtitle destination", path: item.SubtitleDestination},
		} {
			if err := validateNormalizedAbsolutePath(field.path); err != nil {
				return fmt.Errorf("item %d %s path: %w", index, field.name, err)
			}
		}
		if item.SourceFingerprint.SizeBytes < 0 || item.SourceFingerprint.ModifiedAt.IsZero() {
			return fmt.Errorf("item %d source fingerprint is invalid", index)
		}

		sourceKey := pathKey(item.SourcePath)
		if previous, exists := sources[sourceKey]; exists {
			return fmt.Errorf("items %d and %d refer to the same source path", previous, index)
		}
		sources[sourceKey] = index

		for _, destination := range []string{item.VideoDestination, item.SubtitleDestination} {
			key := pathKey(destination)
			if previous, exists := destinations[key]; exists {
				return fmt.Errorf("items %d and %d use the same output destination", previous, index)
			}
			if sourceIndex, exists := sources[key]; exists {
				return fmt.Errorf("item %d output destination would replace source from item %d", index, sourceIndex)
			}
			destinations[key] = index
		}
		if destination, exists := destinations[sourceKey]; exists {
			return fmt.Errorf("item %d source path collides with output from item %d", index, destination)
		}

		if err := item.Compression.validate("video.mp4"); err != nil {
			return fmt.Errorf("item %d compression stage: %w", index, err)
		}
		if err := item.Subtitles.validate("transcript.srt"); err != nil {
			return fmt.Errorf("item %d subtitle stage: %w", index, err)
		}
		for _, stage := range []StageRecord{item.Compression, item.Subtitles} {
			if stage.WorkspaceID == "" {
				continue
			}
			if _, exists := workspaceOwners[stage.WorkspaceID]; exists {
				return fmt.Errorf("workspace %q is assigned to multiple stage artifacts", stage.WorkspaceID)
			}
			workspaceOwners[stage.WorkspaceID] = struct{}{}
		}
	}
	return nil
}

func (identity ProcessingIdentity) validate() error {
	if !identity.CompressionEnabled && !identity.SubtitlesEnabled {
		return fmt.Errorf("at least one processing stage must be enabled")
	}
	if identity.CompressionEnabled {
		codec := strings.ToLower(strings.TrimSpace(identity.VideoCodec))
		if (codec != "h264" && codec != "hevc") || identity.MaxWidth < 2 || identity.MaxWidth > 16384 ||
			identity.MaxHeight < 2 || identity.MaxHeight > 16384 || identity.FrameRate < 0 || identity.FrameRate > 240 {
			return fmt.Errorf("compression settings are incomplete")
		}
		if identity.VideoBackend != "amf" && identity.VideoBackend != "software" {
			return fmt.Errorf("video backend must be amf or software")
		}
		if strings.TrimSpace(identity.FFmpegRevision) == "" {
			return fmt.Errorf("FFmpeg runtime identity is required for compression")
		}
	}
	if identity.SubtitlesEnabled {
		if strings.TrimSpace(identity.Language) == "" || identity.WhisperThreads < 1 {
			return fmt.Errorf("subtitle settings are incomplete")
		}
		if strings.TrimSpace(identity.WhisperRevision) == "" || !validSHA256(identity.ModelSHA256) {
			return fmt.Errorf("Whisper runtime revision and model SHA-256 are required for subtitles")
		}
	}
	return nil
}

func (stage StageRecord) validate(expectedArtifact string) error {
	switch stage.State {
	case StagePending, StageRunning, StageStaged, StagePublished, StageFailed, StageCancelled, StageSkipped:
	default:
		return fmt.Errorf("unknown state %q", stage.State)
	}
	if (stage.WorkspaceID == "") != (stage.ArtifactName == "") {
		return fmt.Errorf("workspace ID and artifact name must be set together")
	}
	if stage.WorkspaceID != "" && !validWorkspaceID(stage.WorkspaceID) {
		return fmt.Errorf("workspace ID is invalid")
	}
	if stage.ArtifactName != "" && stage.ArtifactName != expectedArtifact {
		return fmt.Errorf("artifact name %q is invalid for this stage", stage.ArtifactName)
	}
	if stage.State == StageStaged && stage.WorkspaceID == "" {
		return fmt.Errorf("staged output requires a workspace ID and artifact name")
	}
	return nil
}

func normalizePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
		return "", fmt.Errorf("path must be non-empty and contain no NUL character")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func validateNormalizedAbsolutePath(path string) error {
	if path == "" || strings.ContainsRune(path, '\x00') || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("path must be normalized, absolute, and contain no NUL character")
	}
	return nil
}

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func validJobID(id string) bool {
	if len(id) != 32 || strings.ToLower(id) != id {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validWorkspaceID(id string) bool {
	if len(id) != 14 || !strings.HasPrefix(id, "w-") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(id, "w-"))
	return err == nil
}

func validSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func newJobID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}
