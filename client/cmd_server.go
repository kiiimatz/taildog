package main

import (
	"github.com/kiiimatz/taildog/relay/server"
	"github.com/spf13/cobra"
)

func cmdServer() *cobra.Command {
	cfg := server.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the taildog relay server",
		Long: `Start the taildog relay server.

The server listens for client daemon connections (control port) and serves
the REST API and web dashboard (api port) over TLS.

On first run you will be prompted to create an admin account.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Run(cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "directory for DB, certs and secrets")
	cmd.Flags().IntVar(&cfg.ControlPort, "control-port", cfg.ControlPort, "TLS port for client connections")
	cmd.Flags().IntVar(&cfg.APIPort, "api-port", cfg.APIPort, "TLS port for the REST API and dashboard")
	cmd.Flags().StringVar(&cfg.APIHost, "api-host", cfg.APIHost, "listen address for the API")

	return cmd
}
