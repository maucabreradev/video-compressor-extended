// Package publisher contains safe output-publication primitives.
package publisher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PublishNoReplace atomically creates destination as a hard link to temporary.
// It never replaces an existing destination. The caller owns cleanup of the
// temporary path; placing both paths in the destination directory guarantees
// that they are on the same filesystem.
func PublishNoReplace(temporary, destination string) error {
	if strings.TrimSpace(temporary) == "" || strings.TrimSpace(destination) == "" {
		return fmt.Errorf("temporary and destination paths are required")
	}
	if err := os.Link(temporary, destination); err != nil {
		return fmt.Errorf("create output %q without replacement: %w", destination, err)
	}
	return nil
}

// PublishNoReplaceWithCopy publishes a validated artifact without replacing an
// existing destination. It first attempts the same-volume hard-link fast path.
// If that fails, it copies the artifact to a unique file beside destination,
// validates that copy, and publishes it with a no-replace hard link. The
// caller retains ownership of temporary in all cases.
func PublishNoReplaceWithCopy(ctx context.Context, temporary, destination string, validate func(string) error) error {
	if ctx == nil {
		return fmt.Errorf("publication context is required")
	}
	if strings.TrimSpace(temporary) == "" || strings.TrimSpace(destination) == "" {
		return fmt.Errorf("temporary and destination paths are required")
	}
	return publishNoReplaceWithLink(ctx, temporary, destination, validate, os.Link)
}

func publishNoReplaceWithLink(ctx context.Context, temporary, destination string, validate func(string) error, link func(string, string) error) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := link(temporary, destination); err == nil {
		return nil
	}

	staging, err := os.CreateTemp(filepath.Dir(destination), ".vcx-publish-*")
	if err != nil {
		return fmt.Errorf("create destination-side staging file for %q: %w", destination, err)
	}
	stagingPath := staging.Name()
	defer func() {
		if err := os.Remove(stagingPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove destination-side staging file %q: %w", stagingPath, err))
		}
	}()

	source, err := os.Open(temporary)
	if err != nil {
		_ = staging.Close()
		return fmt.Errorf("open staged artifact %q for copying: %w", temporary, err)
	}
	_, copyErr := io.Copy(staging, contextReader{ctx: ctx, reader: source})
	sourceCloseErr := source.Close()
	if copyErr != nil {
		_ = staging.Close()
		return fmt.Errorf("copy artifact to destination-side staging file: %w", copyErr)
	}
	if sourceCloseErr != nil {
		_ = staging.Close()
		return fmt.Errorf("close staged artifact after copying: %w", sourceCloseErr)
	}
	if err := ctx.Err(); err != nil {
		_ = staging.Close()
		return err
	}
	if err := staging.Sync(); err != nil {
		_ = staging.Close()
		return fmt.Errorf("flush destination-side staging file: %w", err)
	}
	if err := staging.Close(); err != nil {
		return fmt.Errorf("close destination-side staging file: %w", err)
	}
	if validate != nil {
		if err := validate(stagingPath); err != nil {
			return fmt.Errorf("validate copied artifact before publication: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := link(stagingPath, destination); err != nil {
		return fmt.Errorf("publish copied output %q without replacement: %w", destination, err)
	}
	return nil
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
