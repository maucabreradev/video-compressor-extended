package subtitle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSRTAcceptsSequentialTimedCues(t *testing.T) {
	path := writeSRT(t, "1\r\n00:00:00,000 --> 00:00:01,250\r\nHello, world.\r\n\r\n2\r\n00:00:01,500 --> 00:00:02,000\r\nSecond line\r\ncontinued\r\n")

	if err := ValidateSRT(path); err != nil {
		t.Fatalf("validate valid SRT: %v", err)
	}
}

func TestValidateSRTAcceptsWhisperCPPCLIOutput(t *testing.T) {
	contents := "1\n00:00:00,000 --> 00:00:07,940\n And so my fellow Americans ask not what your country can do for you\n\n" +
		"2\n00:00:07,940 --> 00:00:10,760\n ask what you can do for your country.\n"
	if err := ValidateSRT(writeSRT(t, contents)); err != nil {
		t.Fatalf("validate SRT emitted by whisper.cpp: %v", err)
	}
}

func TestValidateSRTRejectsEmptyOutput(t *testing.T) {
	path := writeSRT(t, "\n  \n")

	if err := ValidateSRT(path); err == nil {
		t.Fatal("expected empty subtitle output to be rejected")
	}
}

func TestValidateSRTRejectsMalformedCueAndTimestamp(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "missing timestamp", text: "1\njust text\n"},
		{name: "invalid timestamp", text: "1\n00:00:61,000 --> 00:00:62,000\ntext\n"},
		{name: "empty cue text", text: "1\n00:00:00,000 --> 00:00:01,000\n\n"},
		{name: "non-sequential cue index", text: "2\n00:00:00,000 --> 00:00:01,000\ntext\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateSRT(writeSRT(t, test.text)); err == nil {
				t.Fatal("expected malformed SRT to be rejected")
			}
		})
	}
}

func TestParseSRTTimestampRejectsDurationOverflow(t *testing.T) {
	if _, err := parseSRTTimestamp("3000000000000:00:00,000"); err == nil {
		t.Fatal("expected a timestamp outside time.Duration range to be rejected")
	}
}

func writeSRT(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "captions.srt")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
