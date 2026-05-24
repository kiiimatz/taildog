package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/kiiimatz/taildog/client/internal/config"
	"github.com/kiiimatz/taildog/client/internal/daemon"
	"github.com/spf13/cobra"
)

const clientVersion = "0.1.0"

// validProtocols is the set of protocols accepted by the add command.
var validProtocols = map[string]struct{}{
	"tcp": {}, "udp": {}, "http": {}, "https": {},
	"socks5": {}, "quic": {}, "ws": {}, "scp": {}, "smtp": {},
}

func main() {
	root := buildRoot()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "taildog",
		Short: "taildog – secure tunnel client",
		Long:  "taildog manages encrypted tunnels to a remote relay server.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return requireAdmin()
		},
	}

	root.AddCommand(
		cmdUp(),
		cmdDown(),
		cmdStatus(),
		cmdAdd(),
		cmdRemove(),
		cmdLogs(),
		cmdVersion(),
		cmdServer(),
	)
	return root
}

// ---------------------------------------------------------------------------
// taildog up
// ---------------------------------------------------------------------------

func cmdUp() *cobra.Command {
	var foreground bool
	cmd := &cobra.Command{
		Use:   "up <relay-host>",
		Short: "Start the taildog daemon",
		Long: `Start the taildog daemon and connect to the relay server.

The relay server's IP or hostname is required.
This overwrites relay_host in the config file.

  taildog up 203.0.113.10
  taildog up relay.example.com`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if foreground {
				// Internal re-invocation by daemon.Start() — no relay-host arg needed.
				return daemon.RunForeground()
			}
			if len(args) == 0 {
				return fmt.Errorf("relay-host is required\n\nUsage: taildog up <relay-host>")
			}
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			cfg.RelayHost = args[0]
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Printf("relay host → %s\n", args[0])
			if err := daemon.Start(); err != nil {
				return err
			}
			installAutostart()
			return nil
		},
	}
	cmd.Flags().BoolVar(&foreground, "foreground", false,
		"Run daemon in the foreground (used by service managers)")
	return cmd
}

// ---------------------------------------------------------------------------
// taildog down
// ---------------------------------------------------------------------------

func cmdDown() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the taildog daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return daemon.Stop()
		},
	}
}

// ---------------------------------------------------------------------------
// taildog status
// ---------------------------------------------------------------------------

func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon and tunnel status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			alive, pid, err := daemon.IsRunning()
			if err != nil {
				return err
			}

			// Daemon line
			if alive {
				fmt.Printf("Daemon:  running (PID: %d)\n", pid)
			} else {
				fmt.Println("Daemon:  stopped")
			}

			// Relay line
			relay := cfg.RelayHost
			if relay == "" {
				relay = "(not configured)"
			}
			fmt.Printf("Relay:   %s:%d\n", relay, cfg.RelayPort)
			fmt.Printf("Tunnels: %d\n", len(cfg.Tunnels))

			if len(cfg.Tunnels) == 0 {
				return nil
			}

			// Tunnel table
			fmt.Println()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tProtocol\tLocalPort\tRemotePort\tStatus")
			fmt.Fprintln(w, "──────────────────────────────────────────────────────────────")
			for _, t := range cfg.Tunnels {
				status := "inactive"
				if alive {
					status = "active"
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
					t.ID, t.Protocol, t.LocalPort, t.RemotePort, status)
			}
			w.Flush()
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// taildog add <proto> <localPort> [remotePort]
// ---------------------------------------------------------------------------

func cmdAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <protocol> <localPort> [remotePort]",
		Short: "Add a tunnel to the configuration",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			proto := strings.ToLower(args[0])
			if _, ok := validProtocols[proto]; !ok {
				sorted := sortedProtocols()
				return fmt.Errorf("unknown protocol %q — valid: %s", proto, strings.Join(sorted, ", "))
			}

			localPort, err := strconv.Atoi(args[1])
			if err != nil || localPort < 1 || localPort > 65535 {
				return fmt.Errorf("invalid local port: %s", args[1])
			}

			remotePort := 0
			if len(args) == 3 {
				remotePort, err = strconv.Atoi(args[2])
				if err != nil || remotePort < 0 || remotePort > 65535 {
					return fmt.Errorf("invalid remote port: %s", args[2])
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			id := uuid.NewString()
			tunnel := config.Tunnel{
				ID:         id,
				Protocol:   proto,
				LocalPort:  localPort,
				RemotePort: remotePort,
			}
			cfg.Tunnels = append(cfg.Tunnels, tunnel)

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			remoteStr := strconv.Itoa(remotePort)
			if remotePort == 0 {
				remoteStr = "auto"
			}
			fmt.Printf("Tunnel added:\n")
			fmt.Printf("  ID:         %s\n", id)
			fmt.Printf("  Protocol:   %s\n", proto)
			fmt.Printf("  LocalPort:  %d\n", localPort)
			fmt.Printf("  RemotePort: %s\n", remoteStr)
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// taildog remove <id>
// ---------------------------------------------------------------------------

func cmdRemove() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a tunnel from the configuration (prefix match)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			var kept []config.Tunnel
			var removed []config.Tunnel
			for _, t := range cfg.Tunnels {
				if strings.HasPrefix(t.ID, prefix) {
					removed = append(removed, t)
				} else {
					kept = append(kept, t)
				}
			}

			if len(removed) == 0 {
				return fmt.Errorf("no tunnel found with ID prefix %q", prefix)
			}
			if len(removed) > 1 {
				return fmt.Errorf("prefix %q is ambiguous — matches %d tunnels; use a longer prefix", prefix, len(removed))
			}

			cfg.Tunnels = kept
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Tunnel removed: %s (%s local:%d)\n",
				removed[0].ID, removed[0].Protocol, removed[0].LocalPort)
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// taildog logs
// ---------------------------------------------------------------------------

func cmdLogs() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Tail the daemon log (last 50 lines, then follow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return daemon.TailLog(os.Stdout, 50)
		},
	}
}

// ---------------------------------------------------------------------------
// taildog version
// ---------------------------------------------------------------------------

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("taildog client v%s\n", clientVersion)
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sortedProtocols() []string {
	// Return protocols in a deterministic, human-friendly order.
	order := []string{"tcp", "udp", "http", "https", "socks5", "quic", "ws", "scp", "smtp"}
	var out []string
	for _, p := range order {
		if _, ok := validProtocols[p]; ok {
			out = append(out, p)
		}
	}
	return out
}
