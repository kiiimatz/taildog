package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config holds the full client configuration.
type Config struct {
	RelayHost  string   `yaml:"relay_host"`
	RelayPort  int      `yaml:"relay_port"`
	AuthToken  string   `yaml:"auth_token"`
	CACertPath string   `yaml:"ca_cert_path"`
	Tunnels    []Tunnel `yaml:"tunnels"`
}

// Tunnel describes a single tunnel entry.
type Tunnel struct {
	ID         string `yaml:"id"`
	Protocol   string `yaml:"protocol"`
	LocalPort  int    `yaml:"local_port"`
	RemotePort int    `yaml:"remote_port"`
}

// ConfigPath returns the platform-appropriate path to the config file.
func ConfigPath() string {
	var dir string
	if runtime.GOOS == "windows" {
		dir = filepath.Join(os.Getenv("APPDATA"), "taildog")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".taildog")
	}
	return filepath.Join(dir, "config.yaml")
}

// ConfigDir returns the directory that contains the config file.
func ConfigDir() string {
	return filepath.Dir(ConfigPath())
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		RelayHost: "",
		RelayPort: 7700,
		Tunnels:   []Tunnel{},
	}
}

// Load reads the config from the platform-appropriate path.
// If the file does not exist it returns a default Config without error.
func Load() (*Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config back to the platform-appropriate path,
// creating the directory if it does not exist.
func Save(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
