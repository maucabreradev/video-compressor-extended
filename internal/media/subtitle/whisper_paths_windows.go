//go:build windows

package subtitle

import (
	"fmt"

	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

func prepareWhisperInvocation(executable, model string) (string, string, string, func(), error) {
	shortExecutable, err := process.WindowsShortPath(executable)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("create Windows short path for whisper.cpp executable: %w", err)
	}
	shortModel, err := process.WindowsShortPath(model)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("create Windows short path for Whisper model: %w", err)
	}
	return shortExecutable, shortModel, "", func() {}, nil
}
