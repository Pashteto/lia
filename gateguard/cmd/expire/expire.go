package expire

import (
	"github.com/misnaged/scriptorium/logger"
	"github.com/spf13/cobra"

	"gateguard/internal"
)

// Cmd returns the "serve" command of the application.
// This command is responsible for initializing and
func Cmd(app *internal.App) *cobra.Command {
	return &cobra.Command{
		Use:   "expire",
		Short: "Expire Invitations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Expire()
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			logger.Log().Info(app.Version())
		},
	}
}
