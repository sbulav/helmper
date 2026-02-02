package bootstrap

import (
	"os"

	"log/slog"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// ProvideLogger sets up the slog configuration
func ProvideLogger(v *viper.Viper) *slog.Logger {
	slogHandlerOpts := &slog.HandlerOptions{}

	// Check environment variable first, then viper config for verbose flag
	if os.Getenv("HELMPER_LOG_LEVEL") == "DEBUG" || v.GetBool("verbose") {
		slogHandlerOpts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, slogHandlerOpts))

	// Set this logger as the default
	slog.SetDefault(logger)

	// Example log entries
	slog.Info("Application started")
	slog.Debug("Debugging application")

	return logger
}

var LoggerModule = fx.Provide(ProvideLogger)
