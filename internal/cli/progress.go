package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
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
	case app.RunStageStateChanged:
		stage := string(event.Stage)
		switch event.State {
		case scheduler.Queued:
			_, err := fmt.Fprintf(reporter.writer, "Queued %s: %s\n", stage, file)
			return wrapBatchWriteError(err)
		case scheduler.Running:
			if event.Stage == scheduler.Transcription {
				_, err := fmt.Fprintf(reporter.writer, "Transcribing %s\n", file)
				return wrapBatchWriteError(err)
			}
			return nil // RunStarted renders compression as "Compressing".
		case scheduler.Completed:
			if event.Stage == scheduler.Transcription {
				_, err := fmt.Fprintf(reporter.writer, "Transcription completed: %s\n", file)
				return wrapBatchWriteError(err)
			}
			return nil // RunCompleted reports the completed file.
		case scheduler.Skipped:
			output := event.Destination
			if output == "" {
				output = file
			}
			_, err := fmt.Fprintf(reporter.writer, "Skipped %s (validated output): %s\n", stage, output)
			return wrapBatchWriteError(err)
		case scheduler.Cancelled:
			_, err := fmt.Fprintf(reporter.writer, "Cancelled %s: %s\n", stage, file)
			return wrapBatchWriteError(err)
		case scheduler.Failed:
			message := fmt.Sprintf("Failed %s for %s", stage, file)
			if event.Err != nil {
				message += ": " + event.Err.Error()
			}
			_, err := fmt.Fprintln(reporter.writer, message)
			return wrapBatchWriteError(err)
		default:
			return fmt.Errorf("unsupported batch stage state %q", event.State)
		}
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
