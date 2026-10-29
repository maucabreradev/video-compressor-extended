package ffmpeg

import (
	"fmt"
	"strings"
)

// Backend identifies the platform-specific AMD encoder API.
type Backend string

const (
	BackendAMF   Backend = "amf"
	BackendVAAPI Backend = "vaapi"
)

// Options contains the codec, resize, and backend settings shared by encoding
// and short hardware capability probes.
type Options struct {
	Backend     Backend
	Codec       string
	VAAPIDevice string
	MaxWidth    int
	MaxHeight   int
	FrameRate   int
}

// EncoderName maps a requested codec and backend to its FFmpeg encoder name.
func EncoderName(backend Backend, codec string) (string, error) {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec != "h264" && codec != "hevc" {
		return "", fmt.Errorf("unsupported video codec %q; choose h264 or hevc", codec)
	}
	var suffix string
	switch backend {
	case BackendAMF:
		suffix = "_amf"
	case BackendVAAPI:
		suffix = "_vaapi"
	default:
		return "", fmt.Errorf("unsupported AMD encoder backend %q", backend)
	}
	return codec + suffix, nil
}

// BuildEncodeArgs creates FFmpeg arguments for an AMD-backed MP4 encode. The
// arguments are intended for exec.Command-style invocation, not shell parsing.
func BuildEncodeArgs(input, output string, options Options) ([]string, error) {
	encoder, err := validateOptions(input, output, options)
	if err != nil {
		return nil, err
	}
	args := []string{"-hide_banner", "-nostdin", "-y"}
	if options.Backend == BackendVAAPI {
		args = append(args, "-vaapi_device", options.VAAPIDevice)
	}
	args = append(args,
		"-i", input,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-vf", videoFilter(options),
		"-c:v", encoder,
	)
	args = append(args, qualityArgs(options.Backend)...)
	args = append(args,
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-progress", "pipe:1",
		"-nostats",
		"-f", "mp4",
		output,
	)
	return args, nil
}

// BuildProbeArgs creates FFmpeg arguments that encode one synthetic frame into
// the provided temporary output path using the selected AMD backend.
func BuildProbeArgs(output string, options Options) ([]string, error) {
	encoder, err := validateOptions("synthetic:color", output, options)
	if err != nil {
		return nil, err
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y"}
	if options.Backend == BackendVAAPI {
		args = append(args, "-vaapi_device", options.VAAPIDevice)
	}
	args = append(args,
		"-f", "lavfi",
		"-i", "color=c=black:s=128x128:r=30:d=0.1",
		"-frames:v", "1",
		"-an",
		"-vf", probeFilter(options),
		"-c:v", encoder,
	)
	args = append(args, qualityArgs(options.Backend)...)
	args = append(args, "-f", "mp4", output)
	return args, nil
}

func validateOptions(input, output string, options Options) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("input path must not be empty")
	}
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("output path must not be empty")
	}
	if options.MaxWidth < 2 || options.MaxHeight < 2 {
		return "", fmt.Errorf("maximum width and height must both be at least 2")
	}
	if options.FrameRate < 0 || options.FrameRate > 240 {
		return "", fmt.Errorf("frame rate must be 0 or between 1 and 240, got %d", options.FrameRate)
	}
	if options.Backend == BackendVAAPI && strings.TrimSpace(options.VAAPIDevice) == "" {
		return "", fmt.Errorf("VAAPI requires a render device")
	}
	return EncoderName(options.Backend, options.Codec)
}

func videoFilter(options Options) string {
	filters := make([]string, 0, 3)
	if options.FrameRate > 0 {
		filters = append(filters, fmt.Sprintf("fps=%d", options.FrameRate))
	}
	if options.Backend == BackendVAAPI {
		filters = append(filters, "format=nv12", "hwupload", fmt.Sprintf(
			"scale_vaapi=w='min(%d,iw)':h='min(%d,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2",
			options.MaxWidth,
			options.MaxHeight,
		))
	} else {
		filters = append(filters, fmt.Sprintf(
			"scale=w='min(%d,iw)':h='min(%d,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2",
			options.MaxWidth,
			options.MaxHeight,
		), "format=nv12")
	}
	return strings.Join(filters, ",")
}

func probeFilter(options Options) string {
	if options.Backend == BackendVAAPI {
		return "format=nv12,hwupload"
	}
	return "format=nv12"
}

func qualityArgs(backend Backend) []string {
	switch backend {
	case BackendAMF:
		return []string{"-quality", "balanced", "-rc", "cqp", "-qp_i", "28", "-qp_p", "28"}
	case BackendVAAPI:
		return []string{"-qp", "24"}
	default:
		return nil
	}
}
