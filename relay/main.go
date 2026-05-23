// taildog relay daemon — entry point.
package main

import (
	"flag"
	"log"

	"github.com/kiiimatz/taildog/relay/server"
)

var version = "dev"

func main() {
	cfg := server.DefaultConfig()

	flag.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "directory for DB, certs and secrets")
	flag.IntVar(&cfg.ControlPort, "control-port", cfg.ControlPort, "TLS port for client daemon connections")
	flag.IntVar(&cfg.APIPort, "api-port", cfg.APIPort, "TLS port for the REST API / dashboard")
	flag.StringVar(&cfg.APIHost, "api-host", cfg.APIHost, "listen address for the REST API")
	flag.StringVar(&cfg.DashboardOrigin, "dashboard-origin", cfg.DashboardOrigin, "allowed CORS origin (dev only)")
	flag.Parse()

	cfg.Version = version

	if err := server.Run(cfg); err != nil {
		log.Fatalf("relay: %v", err)
	}
}
