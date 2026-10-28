package cli

import (
	"context"
	"fmt"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/spf13/cobra"
)

// Handler performs one application action using validated configuration.
type Handler func(context.Context, config.Config) error

// Handlers connects CLI commands to application services.
type Handlers struct {
	TUI    Handler
	Run    Handler
	Doctor Handler
}

type cliOptions struct {
	configPath string
	values     config.Config
}

// NewRootCommand creates the VCX command tree. Command behavior is injected so
// the CLI can be tested without launching media tools or terminal interfaces.
func NewRootCommand(version string, handlers Handlers) *cobra.Command {
	options := &cliOptions{values: config.Defaults()}

	root := &cobra.Command{
		Use:           "vcx",
		Short:         "Compress videos and generate subtitles",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.Version = version
	root.SetVersionTemplate("VCX v{{.Version}}\n")

	flags := root.PersistentFlags()
	flags.StringVar(&options.configPath, "config", "", "path to the TOML configuration file")
	flags.StringVar(&options.values.InputRoot, "input", options.values.InputRoot, "input directory or media path")
	flags.StringVar(&options.values.OutputDir, "output-dir", options.values.OutputDir, "directory for processed files")
	flags.IntVar(&options.values.EncodeWorkers, "encode-workers", options.values.EncodeWorkers, "maximum concurrent encode jobs")
	flags.IntVar(&options.values.SubtitleWorkers, "subtitle-workers", options.values.SubtitleWorkers, "maximum concurrent subtitle jobs")
	flags.IntVar(&options.values.WhisperThreads, "whisper-threads", options.values.WhisperThreads, "CPU threads per transcription job")
	flags.StringVar(&options.values.Language, "language", options.values.Language, "subtitle language code")
	flags.StringVar(&options.values.VideoCodec, "video-codec", options.values.VideoCodec, "video codec to use (h264 or hevc)")
	flags.StringVar(&options.values.WhisperModelPath, "whisper-model-path", options.values.WhisperModelPath, "path to the whisper.cpp model file")
	flags.IntVar(&options.values.MaxWidth, "max-width", options.values.MaxWidth, "maximum output width")
	flags.IntVar(&options.values.MaxHeight, "max-height", options.values.MaxHeight, "maximum output height")
	flags.IntVar(&options.values.FrameRate, "frame-rate", options.values.FrameRate, "output frame rate; 0 preserves the source frame rate")

	root.AddCommand(newActionCommand(root, options, "tui", "open the interactive terminal interface", handlers.TUI))
	root.AddCommand(newActionCommand(root, options, "run", "process media without the interactive interface", handlers.Run))
	root.AddCommand(newActionCommand(root, options, "doctor", "check required tools and encoders", handlers.Doctor))
	return root
}

func newActionCommand(
	root *cobra.Command,
	options *cliOptions,
	name string,
	short string,
	handler Handler,
) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := options.load(root)
			if err != nil {
				return err
			}
			if handler == nil {
				return fmt.Errorf("%s command handler is not configured", name)
			}
			return handler(cmd.Context(), cfg)
		},
	}
}

func (options *cliOptions) load(root *cobra.Command) (config.Config, error) {
	cfg, err := config.Parse(options.configPath)
	if err != nil {
		return config.Config{}, err
	}

	flags := root.PersistentFlags()
	if flags.Changed("input") {
		cfg.InputRoot = options.values.InputRoot
	}
	if flags.Changed("output-dir") {
		cfg.OutputDir = options.values.OutputDir
	}
	if flags.Changed("encode-workers") {
		cfg.EncodeWorkers = options.values.EncodeWorkers
	}
	if flags.Changed("subtitle-workers") {
		cfg.SubtitleWorkers = options.values.SubtitleWorkers
	}
	if flags.Changed("whisper-threads") {
		cfg.WhisperThreads = options.values.WhisperThreads
	}
	if flags.Changed("language") {
		cfg.Language = options.values.Language
	}
	if flags.Changed("video-codec") {
		cfg.VideoCodec = options.values.VideoCodec
	}
	if flags.Changed("whisper-model-path") {
		cfg.WhisperModelPath = options.values.WhisperModelPath
	}
	if flags.Changed("max-width") {
		cfg.MaxWidth = options.values.MaxWidth
	}
	if flags.Changed("max-height") {
		cfg.MaxHeight = options.values.MaxHeight
	}
	if flags.Changed("frame-rate") {
		cfg.FrameRate = options.values.FrameRate
	}
	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}
