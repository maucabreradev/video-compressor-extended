package process

import (
	"strconv"
	"strings"
	"time"
)

const maxProgressLineBytes = 8 * 1024

// FFmpegProgress is a snapshot emitted by FFmpeg's -progress protocol.
type FFmpegProgress struct {
	Frame     int64
	FPS       float64
	OutTime   time.Duration
	TotalSize int64
	Speed     float64
	Percent   float64
	Done      bool
}

// FFmpegProgressParser incrementally parses progress key/value lines. Its
// pending line buffer is bounded so malformed output cannot grow memory without
// limit.
type FFmpegProgressParser struct {
	duration       time.Duration
	pending        []byte
	discardingLine bool
	fields         map[string]string
}

// NewFFmpegProgressParser creates a parser with an optional media duration for
// percentage calculation.
func NewFFmpegProgressParser(duration time.Duration) *FFmpegProgressParser {
	return &FFmpegProgressParser{
		duration: duration,
		fields:   make(map[string]string),
	}
}

// Feed consumes an arbitrary output chunk and returns each complete progress
// snapshot found in it.
func (parser *FFmpegProgressParser) Feed(chunk []byte) []FFmpegProgress {
	progress := make([]FFmpegProgress, 0)
	for _, char := range chunk {
		if char == '\n' {
			if parser.discardingLine {
				parser.discardingLine = false
				parser.pending = parser.pending[:0]
				continue
			}
			line := strings.TrimSuffix(string(parser.pending), "\r")
			parser.pending = parser.pending[:0]
			if event, ok := parser.consume(line); ok {
				progress = append(progress, event)
			}
			continue
		}
		if parser.discardingLine {
			continue
		}
		if len(parser.pending) >= maxProgressLineBytes {
			parser.pending = parser.pending[:0]
			parser.discardingLine = true
			continue
		}
		parser.pending = append(parser.pending, char)
	}
	return progress
}

func (parser *FFmpegProgressParser) consume(line string) (FFmpegProgress, bool) {
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return FFmpegProgress{}, false
	}
	switch key {
	case "frame", "fps", "out_time_us", "out_time", "total_size", "speed", "progress":
	default:
		return FFmpegProgress{}, false
	}
	parser.fields[key] = value
	if key != "progress" {
		return FFmpegProgress{}, false
	}

	event := FFmpegProgress{
		Frame:     parseInt(parser.fields["frame"]),
		FPS:       parseFloat(parser.fields["fps"]),
		OutTime:   parseOutTime(parser.fields),
		TotalSize: parseInt(parser.fields["total_size"]),
		Speed:     parseFloat(strings.TrimSuffix(parser.fields["speed"], "x")),
		Done:      value == "end",
	}
	if event.Done {
		event.Percent = 100
	} else if parser.duration > 0 {
		event.Percent = float64(event.OutTime) / float64(parser.duration) * 100
		if event.Percent < 0 {
			event.Percent = 0
		}
		if event.Percent > 100 {
			event.Percent = 100
		}
	}
	clear(parser.fields)
	return event, true
}

func parseInt(value string) int64 {
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func parseOutTime(fields map[string]string) time.Duration {
	if microseconds, err := strconv.ParseInt(fields["out_time_us"], 10, 64); err == nil && microseconds >= 0 {
		return time.Duration(microseconds) * time.Microsecond
	}
	return parseTimestamp(fields["out_time"])
}

func parseTimestamp(value string) time.Duration {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0
	}
	hours, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	minutes, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0
	}
	return time.Duration(float64(hours*3600+minutes*60)*float64(time.Second) + seconds*float64(time.Second))
}
