package subtitle

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxSRTLineBytes = 1024 * 1024

// ValidateSRT verifies that path contains at least one sequential, timed SRT
// cue with non-empty subtitle text.
func ValidateSRT(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open SRT %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxSRTLineBytes)
	lineNumber := 0
	cueNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" {
			continue
		}

		index, err := strconv.Atoi(line)
		if err != nil || index != cueNumber+1 {
			return fmt.Errorf("invalid SRT cue index at line %d", lineNumber)
		}
		cueNumber++

		if !scanner.Scan() {
			return fmt.Errorf("missing timestamp for SRT cue %d", cueNumber)
		}
		lineNumber++
		start, end, err := parseSRTTimeRange(strings.TrimSpace(scanner.Text()))
		if err != nil {
			return fmt.Errorf("invalid timestamp for SRT cue %d at line %d: %w", cueNumber, lineNumber, err)
		}
		if end <= start {
			return fmt.Errorf("SRT cue %d ends before or at its start time", cueNumber)
		}

		textLines := 0
		for scanner.Scan() {
			lineNumber++
			if strings.TrimSpace(scanner.Text()) == "" {
				break
			}
			textLines++
		}
		if textLines == 0 {
			return fmt.Errorf("SRT cue %d has no subtitle text", cueNumber)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SRT %q: %w", path, err)
	}
	if cueNumber == 0 {
		return fmt.Errorf("SRT %q contains no subtitle cues", path)
	}
	return nil
}

func parseSRTTimeRange(value string) (time.Duration, time.Duration, error) {
	startText, endText, ok := strings.Cut(value, " --> ")
	if !ok {
		return 0, 0, fmt.Errorf("expected start and end separated by %q", " --> ")
	}
	start, err := parseSRTTimestamp(startText)
	if err != nil {
		return 0, 0, fmt.Errorf("parse start time: %w", err)
	}
	end, err := parseSRTTimestamp(endText)
	if err != nil {
		return 0, 0, fmt.Errorf("parse end time: %w", err)
	}
	return start, end, nil
}

func parseSRTTimestamp(value string) (time.Duration, error) {
	clock, millisecondsText, ok := strings.Cut(value, ",")
	if !ok {
		return 0, fmt.Errorf("expected comma before milliseconds")
	}
	parts := strings.Split(clock, ":")
	if len(parts) != 3 || len(millisecondsText) != 3 {
		return 0, fmt.Errorf("expected HH:MM:SS,mmm, got %q", value)
	}
	hours, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || hours < 0 {
		return 0, fmt.Errorf("invalid hour value %q", parts[0])
	}
	minutes, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, fmt.Errorf("invalid minute value %q", parts[1])
	}
	seconds, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || seconds < 0 || seconds > 59 {
		return 0, fmt.Errorf("invalid second value %q", parts[2])
	}
	milliseconds, err := strconv.ParseInt(millisecondsText, 10, 64)
	if err != nil || milliseconds < 0 || milliseconds > 999 {
		return 0, fmt.Errorf("invalid millisecond value %q", millisecondsText)
	}
	maxMilliseconds := int64(1<<63-1) / int64(time.Millisecond)
	if hours > maxMilliseconds/3_600_000 {
		return 0, fmt.Errorf("timestamp exceeds supported duration")
	}
	totalMilliseconds := hours*3_600_000 + minutes*60_000 + seconds*1000 + milliseconds
	if totalMilliseconds > maxMilliseconds {
		return 0, fmt.Errorf("timestamp exceeds supported duration")
	}
	return time.Duration(totalMilliseconds) * time.Millisecond, nil
}
