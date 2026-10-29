//go:build !windows

package subtitle

func prepareWhisperInvocation(executable, model string) (string, string, string, func(), error) {
	return executable, model, "", func() {}, nil
}
