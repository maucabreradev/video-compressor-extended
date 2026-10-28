package process

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFFmpegProgressParserHandlesChunkBoundaries(t *testing.T) {
	parser := NewFFmpegProgressParser(20 * time.Second)
	if got := parser.Feed([]byte("frame=12\nfps=24.5\nout_time_us=5000000\nspeed=1.5x\nprogress=cont")); len(got) != 0 {
		t.Fatalf("expected no event before progress line is complete, got %#v", got)
	}
	events := parser.Feed([]byte("inue\r\n"))
	if len(events) != 1 {
		t.Fatalf("expected one progress event, got %#v", events)
	}
	got := events[0]
	if got.Frame != 12 || got.FPS != 24.5 || got.OutTime != 5*time.Second || got.Speed != 1.5 || got.Percent != 25 || got.Done {
		t.Fatalf("unexpected parsed progress: %+v", got)
	}
}

func TestFFmpegProgressParserDoesNotRetainUnknownFields(t *testing.T) {
	parser := NewFFmpegProgressParser(time.Minute)
	for index := 0; index < 1000; index++ {
		parser.Feed([]byte(fmt.Sprintf("unknown_%d=value\n", index)))
	}
	if len(parser.fields) != 0 {
		t.Fatalf("parser retained %d unrecognized fields", len(parser.fields))
	}
}

func TestFFmpegProgressParserMarksCompletionAndClampsPercent(t *testing.T) {
	parser := NewFFmpegProgressParser(10 * time.Second)
	events := parser.Feed([]byte("out_time_us=12000000\nprogress=end\n"))
	if len(events) != 1 {
		t.Fatalf("expected one completion event, got %#v", events)
	}
	if !events[0].Done || events[0].Percent != 100 {
		t.Fatalf("expected completed progress clamped to 100%%, got %+v", events[0])
	}
}

func TestFFmpegProgressParserBoundsIncompleteLines(t *testing.T) {
	parser := NewFFmpegProgressParser(time.Minute)
	chunk := strings.Repeat("x", maxProgressLineBytes+10) + "\nprogress=end\n"
	events := parser.Feed([]byte(chunk))
	if len(events) != 1 || !events[0].Done {
		t.Fatalf("expected parser to discard oversized line and retain progress event, got %#v", events)
	}
}
