package bootstrap

import (
	"log"
	"os"

	"log/slog"

	"github.com/ChristofferNissen/helmper/pkg/util/deduplog"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// LoggerResult is returned by ProvideLogger to provide both logger and dedup writer
type LoggerResult struct {
	fx.Out
	Logger      *slog.Logger
	DedupWriter *deduplog.DeduplicatingWriter `optional:"true"`
}

// ProvideLogger sets up the slog configuration and Helm SDK log deduplication
func ProvideLogger(v *viper.Viper) LoggerResult {
	slogHandlerOpts := &slog.HandlerOptions{}

	// Check environment variable first, then viper config for verbose flag
	if os.Getenv("HELMPER_LOG_LEVEL") == "DEBUG" || v.GetBool("verbose") {
		slogHandlerOpts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, slogHandlerOpts))

	// Set this logger as the default
	slog.SetDefault(logger)

	result := LoggerResult{
		Logger: logger,
	}

	// Set up deduplicating writer for Helm SDK logs
	// The Helm SDK uses Go's standard log package, which we redirect here
	dedupWriter, err := deduplog.NewDeduplicatingWriter(os.Stdout, `skipping loading invalid entry`)
	if err != nil {
		slog.Warn("failed to create deduplicating log writer, using standard output",
			slog.String("error", err.Error()))
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(dedupWriter)
		log.SetFlags(0) // Remove timestamp/file flags since Helm SDK adds its own
		result.DedupWriter = dedupWriter
	}

	// Example log entries
	slog.Info("Application started")
	slog.Debug("Debugging application")

	return result
}

var LoggerModule = fx.Provide(ProvideLogger)
