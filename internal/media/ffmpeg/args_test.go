package ffmpeg

import (
	"strings"
	"testing"
)

func TestBuildAMFEncodeArgsUsesExplicitQualityAndPreservesSourceFrameRate(t *testing.T) {
	args, err := BuildEncodeArgs("input with spaces.mkv", "output.mp4", Options{
		Backend:   BackendAMF,
		Codec:     "h264",
		MaxWidth:  1280,
		MaxHeight: 720,
	})
	if err != nil {
		t.Fatalf("build AMF args: %v", err)
	}
	for _, pair := range [][2]string{
		{"-i", "input with spaces.mkv"},
		{"-c:v", "h264_amf"},
		{"-quality", "balanced"},
		{"-rc", "cqp"},
		{"-qp_i", "28"},
		{"-qp_p", "28"},
		{"-c:a", "aac"},
		{"-movflags", "+faststart"},
		{"-progress", "pipe:1"},
		{"-f", "mp4"},
	} {
		if !containsPair(args, pair[0], pair[1]) {
			t.Errorf("AMF args do not contain %q %q: %#v", pair[0], pair[1], args)
		}
	}
	filter := argumentAfter(args, "-vf")
	if !strings.Contains(filter, "force_original_aspect_ratio=decrease") ||
		!strings.Contains(filter, "min(1280,iw)") ||
		!strings.Contains(filter, "min(720,ih)") ||
		!strings.Contains(filter, "format=nv12") {
		t.Fatalf("AMF filter must preserve aspect ratio, avoid upscaling, and use NV12: %q", filter)
	}
	if strings.Contains(filter, "fps=") || contains(args, "-r") {
		t.Fatalf("default options must preserve source frame rate: %#v", args)
	}
}

func TestBuildVAAPIEncodeArgsUsesDeviceAndExplicitFrameRate(t *testing.T) {
	args, err := BuildEncodeArgs("input.mkv", "output.mp4", Options{
		Backend:     BackendVAAPI,
		Codec:       "hevc",
		VAAPIDevice: "/dev/dri/renderD128",
		MaxWidth:    1280,
		MaxHeight:   720,
		FrameRate:   30,
	})
	if err != nil {
		t.Fatalf("build VAAPI args: %v", err)
	}
	for _, pair := range [][2]string{
		{"-vaapi_device", "/dev/dri/renderD128"},
		{"-c:v", "hevc_vaapi"},
		{"-qp", "24"},
		{"-progress", "pipe:1"},
	} {
		if !containsPair(args, pair[0], pair[1]) {
			t.Errorf("VAAPI args do not contain %q %q: %#v", pair[0], pair[1], args)
		}
	}
	filter := argumentAfter(args, "-vf")
	if !strings.Contains(filter, "fps=30") || !strings.Contains(filter, "hwupload") ||
		!strings.Contains(filter, "scale_vaapi") ||
		!strings.Contains(filter, "min(1280,iw)") ||
		!strings.Contains(filter, "min(720,ih)") {
		t.Fatalf("VAAPI filter does not preserve frame rate, aspect ratio, and resolution limits: %q", filter)
	}
}

func TestBuildProbeArgsUsesValidAMFInitializationParameters(t *testing.T) {
	args, err := BuildProbeArgs("probe.mp4", Options{
		Backend:   BackendAMF,
		Codec:     "h264",
		MaxWidth:  128,
		MaxHeight: 128,
	})
	if err != nil {
		t.Fatalf("build AMF probe args: %v", err)
	}
	for _, pair := range [][2]string{{"-vf", "format=nv12"}, {"-quality", "balanced"}, {"-rc", "cqp"}, {"-qp_i", "28"}, {"-qp_p", "28"}} {
		if !containsPair(args, pair[0], pair[1]) {
			t.Errorf("AMF probe args do not contain %q %q: %#v", pair[0], pair[1], args)
		}
	}
}

func TestBuildArgsRejectsUnsupportedBackendAndDimensions(t *testing.T) {
	tests := []Options{
		{Backend: Backend("software"), Codec: "h264", MaxWidth: 1280, MaxHeight: 720},
		{Backend: BackendAMF, Codec: "h264", MaxWidth: 0, MaxHeight: 720},
		{Backend: BackendVAAPI, Codec: "h264", MaxWidth: 1280, MaxHeight: 720},
	}
	for index, options := range tests {
		if _, err := BuildEncodeArgs("input.mp4", "output.mp4", options); err == nil {
			t.Errorf("case %d: expected invalid backend options to fail", index)
		}
	}
}

func contains(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted {
			return true
		}
	}
	return false
}

func containsPair(args []string, key, value string) bool {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key && args[index+1] == value {
			return true
		}
	}
	return false
}

func argumentAfter(args []string, key string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key {
			return args[index+1]
		}
	}
	return ""
}
