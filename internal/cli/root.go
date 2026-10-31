package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/spf13/cobra"
)

// Handler performs one application action using validated configuration.
type Handler func(context.Context, config.Config) error

// Handlers connects CLI commands to application services. Nil TUI, Run, or
// Doctor handlers leave those actions unavailable; nil ListJobs or ResumeJob
// disables the corresponding saved-job operation.
type Handlers struct {
	TUI       Handler
	Run       Handler
	Doctor    Handler
	ListJobs  func(context.Context) ([]RecoveryJob, error)
	ResumeJob func(context.Context, string) error
}

// RecoveryJob is the script-readable summary emitted by `vcx jobs list`.
type RecoveryJob struct {
	ID               string    `json:"id"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
	SourceSummary    string    `json:"source_summary,omitempty"`
	OutputSummary    string    `json:"output_summary,omitempty"`
	CompressionState string    `json:"compression_state,omitempty"`
	SubtitleState    string    `json:"subtitle_state,omitempty"`
	Status           string    `json:"status"`
	Detail           string    `json:"detail,omitempty"`
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
	flags.StringVar(&options.values.ResourceProfile, "resource-profile", options.values.ResourceProfile, "resource profile (conservative or custom)")
	flags.IntVar(&options.values.EncodeWorkers, "encode-workers", options.values.EncodeWorkers, "maximum concurrent encode jobs")
	flags.IntVar(&options.values.SubtitleWorkers, "subtitle-workers", options.values.SubtitleWorkers, "maximum concurrent subtitle jobs")
	flags.IntVar(&options.values.WhisperThreads, "whisper-threads", options.values.WhisperThreads, "CPU threads per transcription job")
	flags.StringVar(&options.values.ProcessPriority, "process-priority", options.values.ProcessPriority, "best-effort process priority (normal or below_normal); not a resource quota")
	flags.BoolVar(&options.values.WhisperGPU, "whisper-gpu", options.values.WhisperGPU, "allow GPU acceleration in whisper.cpp when its build supports it")
	flags.StringVar(&options.values.Language, "language", options.values.Language, "subtitle language code")
	flags.StringVar(&options.values.VideoCodec, "video-codec", options.values.VideoCodec, "video codec to use (h264 or hevc)")
	flags.StringVar(&options.values.WhisperModelPath, "whisper-model-path", options.values.WhisperModelPath, "path to the whisper.cpp model file")
	flags.BoolVar(&options.values.CompressionEnabled, "compression-enabled", options.values.CompressionEnabled, "enable video compression")
	flags.BoolVar(&options.values.SubtitlesEnabled, "subtitles-enabled", options.values.SubtitlesEnabled, "enable subtitle generation")
	flags.BoolVar(&options.values.AllowSoftwareEncoding, "allow-software-encoding", options.values.AllowSoftwareEncoding, "confirm software H.264 fallback when AMD AMF is unavailable")
	flags.IntVar(&options.values.MaxWidth, "max-width", options.values.MaxWidth, "maximum output width")
	flags.IntVar(&options.values.MaxHeight, "max-height", options.values.MaxHeight, "maximum output height")
	flags.IntVar(&options.values.FrameRate, "frame-rate", options.values.FrameRate, "output frame rate; 0 preserves the source frame rate")

	root.AddCommand(newActionCommand(root, options, "tui", "open the interactive terminal interface", handlers.TUI))
	root.AddCommand(newActionCommand(root, options, "run", "process media without the interactive interface", handlers.Run))
	root.AddCommand(newActionCommand(root, options, "doctor", "check required tools and encoders", handlers.Doctor))
	root.AddCommand(newJobsCommand(root, options, handlers))
	return root
}

func newJobsCommand(root *cobra.Command, options *cliOptions, handlers Handlers) *cobra.Command {
	jobs := &cobra.Command{
		Use:   "jobs",
		Short: "list and resume saved batch jobs",
		Args:  cobra.NoArgs,
	}
	jobs.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "list saved jobs as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if handlers.ListJobs == nil {
				return fmt.Errorf("jobs list handler is not configured")
			}
			jobs, err := handlers.ListJobs(cmd.Context())
			if err != nil {
				return err
			}
			if jobs == nil {
				jobs = []RecoveryJob{}
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(jobs)
		},
	})
	jobs.AddCommand(&cobra.Command{
		Use:   "resume <job-id>",
		Short: "resume a saved job by its complete job ID",
		Long:  "Resume exactly the saved job ID shown by `vcx jobs list`. Common config and CLI options are validated, but the saved manifest remains authoritative for the job's inputs, outputs, and processing identity.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := options.load(root); err != nil {
				return err
			}
			if handlers.ResumeJob == nil {
				return fmt.Errorf("jobs resume handler is not configured")
			}
			return handlers.ResumeJob(cmd.Context(), args[0])
		},
	})
	return jobs
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
		cfg.InputPaths = nil
	}
	if flags.Changed("output-dir") {
		cfg.OutputDir = options.values.OutputDir
	}
	if flags.Changed("resource-profile") {
		cfg.ResourceProfile = options.values.ResourceProfile
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
	if flags.Changed("process-priority") {
		cfg.ProcessPriority = options.values.ProcessPriority
	}
	if flags.Changed("whisper-gpu") {
		cfg.WhisperGPU = options.values.WhisperGPU
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
	if flags.Changed("compression-enabled") {
		cfg.CompressionEnabled = options.values.CompressionEnabled
	}
	if flags.Changed("subtitles-enabled") {
		cfg.SubtitlesEnabled = options.values.SubtitlesEnabled
	}
	if flags.Changed("allow-software-encoding") {
		cfg.AllowSoftwareEncoding = options.values.AllowSoftwareEncoding
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
	if !flags.Changed("resource-profile") && (flags.Changed("encode-workers") ||
		flags.Changed("subtitle-workers") || flags.Changed("whisper-threads") ||
		flags.Changed("process-priority") || flags.Changed("whisper-gpu")) {
		cfg.ResourceProfile = config.ResourceProfileCustom
	}
	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}
