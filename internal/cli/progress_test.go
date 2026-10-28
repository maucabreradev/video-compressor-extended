package cli

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

func TestBatchOutputReporterFormatsAndRateLimitsEvents(t *testing.T) {
	var output bytes.Buffer
	reporter := NewBatchOutputReporter(&output)
	reporter.now = func() time.Time { return time.Unix(100, 0) }
	source := filepath.Join("media folder", "clip.mkv")
	destination := filepath.Join("output", "clip.mp4")

	if err := reporter.Report(app.RunEvent{Kind: app.RunStarted, Source: source}); err != nil {
		t.Fatal(err)
	}
	for frame := int64(1); frame <= 50; frame++ {
		if err := reporter.Report(app.RunEvent{
			Kind: app.RunProgress, Source: source, DurationKnown: true,
			Progress: process.FFmpegProgress{
				Frame: frame, OutTime: time.Duration(frame) * time.Second, Speed: 1.25,
				Percent: float64(frame),
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := reporter.Report(app.RunEvent{
		Kind: app.RunProgress, Source: source, DurationKnown: true,
		Progress: process.FFmpegProgress{
			Frame: 50, OutTime: 50 * time.Second, Speed: 1.25, Percent: 100, Done: true,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Report(app.RunEvent{
		Kind: app.RunCompleted, Source: source, Destination: destination,
	}); err != nil {
		t.Fatal(err)
	}

	got := output.String()
	want := "Compressing clip.mkv\n" +
		"Progress clip.mkv: frame=1 time=1s speed=1.25x 1.0%\n" +
		"Progress clip.mkv: frame=50 time=50s speed=1.25x 100.0%\n" +
		"Completed clip.mkv -> " + destination + "\n"
	if got != want {
		t.Fatalf("unexpected batch rendering:\n got: %q\nwant: %q", got, want)
	}
}

func TestBatchOutputReporterOmitsUnknownPercentageButShowsCompletion(t *testing.T) {
	var output bytes.Buffer
	reporter := NewBatchOutputReporter(&output)
	reporter.now = func() time.Time { return time.Unix(100, 0) }
	if err := reporter.Report(app.RunEvent{
		Kind: app.RunProgress, Source: "clip.mkv",
		Progress: process.FFmpegProgress{Frame: 4, OutTime: 2 * time.Second, Speed: 1.5},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Report(app.RunEvent{
		Kind: app.RunProgress, Source: "clip.mkv",
		Progress: process.FFmpegProgress{Frame: 5, OutTime: 3 * time.Second, Speed: 1.5, Percent: 100, Done: true},
	}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"frame=4 time=2s speed=1.50x\n", "frame=5 time=3s speed=1.50x 100.0%\n"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("batch output missing %q: %q", want, output.String())
		}
	}
}

func TestBatchOutputReporterRendersStageQueueAndFailureStates(t *testing.T) {
	var output bytes.Buffer
	reporter := NewBatchOutputReporter(&output)
	for _, event := range []app.RunEvent{
		{Kind: app.RunStageStateChanged, Source: "clip.mkv", Stage: scheduler.Compression, State: scheduler.Queued},
		{Kind: app.RunStageStateChanged, Source: "clip.mkv", Stage: scheduler.Transcription, State: scheduler.Running},
		{Kind: app.RunStageStateChanged, Source: "clip.mkv", Stage: scheduler.Compression, State: scheduler.Failed, Err: fmt.Errorf("encoder unavailable")},
	} {
		if err := reporter.Report(event); err != nil {
			t.Fatalf("render stage event: %v", err)
		}
	}
	want := "Queued compression: clip.mkv\n" +
		"Transcribing clip.mkv\n" +
		"Failed compression for clip.mkv: encoder unavailable\n"
	if got := output.String(); got != want {
		t.Fatalf("unexpected stage output:\n got: %q\nwant: %q", got, want)
	}
}

func TestBatchOutputReporterReturnsWriterErrors(t *testing.T) {
	reporter := NewBatchOutputReporter(errorWriter{})
	if err := reporter.Report(app.RunEvent{Kind: app.RunStarted, Source: "clip.mkv"}); err == nil ||
		!strings.Contains(err.Error(), "write") {
		t.Fatalf("expected batch output write error, got %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("terminal unavailable") }
