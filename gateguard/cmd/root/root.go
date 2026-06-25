package root

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"gateguard/config"
	"gateguard/internal"
)

// Cmd returns the root command for the application
func Cmd(app *internal.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "microservice",
		Short:            "Service Template",
		TraverseChildren: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd, app.Config())
		},
	}

	cmd.SetVersionTemplate(app.Version())

	return cmd
}

// initializeConfig reads in config file and sets configuration
// via environment variables
func initializeConfig(cmd *cobra.Command, cfg *config.Scheme) error {
	// set config via env vars
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)
	viper.SetConfigType("env")
	viper.SetConfigName(".env")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("read config file: %w", err)
		}
	}

	bindFlags(cmd)

	return viper.Unmarshal(cfg)
}

// bindFlags binds flags to the command
func bindFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
