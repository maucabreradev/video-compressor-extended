package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/app"
)

const batchProgressInterval = time.Second

// BatchOutputReporter renders application run events for non-interactive use.
type BatchOutputReporter struct {
	writer      io.Writer
	lastWritten time.Time
	now         func() time.Time
}

// NewBatchOutputReporter creates a rate-limited renderer for batch lifecycle
// and progress events. A nil writer discards the rendered output.
func NewBatchOutputReporter(writer io.Writer) *BatchOutputReporter {
	if writer == nil {
		writer = io.Discard
	}
	return &BatchOutputReporter{writer: writer, now: time.Now}
}

// Report writes one user-facing batch status line. Progress updates are limited
// to one per second, while the final event is always rendered.
func (reporter *BatchOutputReporter) Report(event app.RunEvent) error {
	file := filepath.Base(event.Source)
	switch event.Kind {
	case app.RunStarted:
		reporter.lastWritten = time.Time{}
		_, err := fmt.Fprintf(reporter.writer, "Compressing %s\n", file)
		return wrapBatchWriteError(err)
	case app.RunProgress:
		now := reporter.now()
		if !event.Progress.Done && !reporter.lastWritten.IsZero() && now.Sub(reporter.lastWritten) < batchProgressInterval {
			return nil
		}
		line := fmt.Sprintf("Progress %s: frame=%d time=%s speed=%.2fx", file, event.Progress.Frame, event.Progress.OutTime, event.Progress.Speed)
		if event.DurationKnown || event.Progress.Done {
			line += fmt.Sprintf(" %.1f%%", event.Progress.Percent)
		}
		if _, err := fmt.Fprintln(reporter.writer, line); err != nil {
			return fmt.Errorf("write batch progress: %w", err)
		}
		reporter.lastWritten = now
		return nil
	case app.RunCompleted:
		_, err := fmt.Fprintf(reporter.writer, "Completed %s -> %s\n", file, event.Destination)
		return wrapBatchWriteError(err)
	default:
		return fmt.Errorf("unsupported batch event kind %q", event.Kind)
	}
}

func wrapBatchWriteError(err error) error {
	if err != nil {
		return fmt.Errorf("write batch status: %w", err)
	}
	return nil
}
