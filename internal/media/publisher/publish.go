// Package publisher contains safe output-publication primitives.
package publisher

import (
	"fmt"
	"os"
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
