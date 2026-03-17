package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config key constants.
const (
	KeyAccountID           = "cloudflare.account_id"
	KeyZoneID              = "cloudflare.zone_id"
	KeyDomain              = "cloudflare.domain"
	KeyAPITokenEnv         = "cloudflare.api_token_env"
	KeyTeamDomain          = "cloudflare.team_domain"
	KeyCloudflaredBin      = "paths.cloudflared_bin"
	KeyStateDir            = "paths.state_dir"
	KeyDefaultAuth         = "defaults.auth"
	KeyDefaultSessionDur   = "defaults.session_duration"
	KeyDefaultHostTemplate = "defaults.hostname_template"
	KeyDefaultReuseTunnel  = "defaults.reuse_tunnel"
	KeyLogLevel            = "log_level"
)

// Init initializes viper with defaults, config file, and env bindings.
func Init() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	setDefaults(dir)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	viper.SetEnvPrefix("FLARE")
	viper.AutomaticEnv()

	// CLOUDFLARE_API_TOKEN is the standard env var (no prefix).
	_ = viper.BindEnv("cloudflare.api_token", "CLOUDFLARE_API_TOKEN")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // Config not yet created; that's fine.
		}
		return fmt.Errorf("reading config: %w", err)
	}
	return nil
}

func setDefaults(stateDir string) {
	viper.SetDefault(KeyAPITokenEnv, "CLOUDFLARE_API_TOKEN")
	viper.SetDefault(KeyCloudflaredBin, "cloudflared")
	viper.SetDefault(KeyStateDir, stateDir)
	viper.SetDefault(KeyDefaultAuth, "otp")
	viper.SetDefault(KeyDefaultSessionDur, "30m")
	viper.SetDefault(KeyDefaultHostTemplate, "{app}-{id}.{domain}")
	viper.SetDefault(KeyDefaultReuseTunnel, true)
	viper.SetDefault(KeyLogLevel, "info")
}

// Dir returns the config directory path, creating it if needed.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "flare-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	return dir, nil
}

// SessionsDir returns the sessions directory, creating it if needed.
func SessionsDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	sessDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessDir, 0700); err != nil {
		return "", fmt.Errorf("creating sessions directory: %w", err)
	}
	return sessDir, nil
}

// WriteConfig writes the current viper config to the config file.
func WriteConfig() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	viper.SetConfigFile(filepath.Join(dir, "config.yaml"))
	return viper.WriteConfig()
}

// SaveConfig writes viper config, creating the file if it doesn't exist.
func SaveConfig() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "config.yaml")
	viper.SetConfigFile(path)
	if err := viper.WriteConfigAs(path); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// APIToken retrieves the Cloudflare API token from the configured env var.
func APIToken() string {
	// Direct config value takes precedence.
	if token := viper.GetString("cloudflare.api_token"); token != "" {
		return token
	}
	// Fall back to the env var name specified in config.
	envName := viper.GetString(KeyAPITokenEnv)
	if envName == "" {
		envName = "CLOUDFLARE_API_TOKEN"
	}
	return os.Getenv(envName)
}

// Validate checks that required config values are present.
func Validate() error {
	if viper.GetString(KeyAccountID) == "" {
		return fmt.Errorf("cloudflare.account_id is not set (run 'flare init')")
	}
	if viper.GetString(KeyZoneID) == "" {
		return fmt.Errorf("cloudflare.zone_id is not set (run 'flare init')")
	}
	if viper.GetString(KeyDomain) == "" {
		return fmt.Errorf("cloudflare.domain is not set (run 'flare init')")
	}
	if APIToken() == "" {
		return fmt.Errorf("CLOUDFLARE_API_TOKEN environment variable is not set")
	}
	return nil
}
