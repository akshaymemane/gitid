package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	envConfigDir = "GITID_CONFIG_DIR"
	configFile   = "profiles.yaml"
)

type GitProfile struct {
	Username string `json:"username" yaml:"username"`
	Email    string `json:"email" yaml:"email"`
}

type GitHubProfile struct {
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Hostname string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
}

type SSHProfile struct {
	KeyPath   string `json:"key_path,omitempty" yaml:"key_path,omitempty"`
	HostName  string `json:"host_name,omitempty" yaml:"host_name,omitempty"`
	HostAlias string `json:"host_alias,omitempty" yaml:"host_alias,omitempty"`
}

type SigningProfile struct {
	Enabled bool   `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	GPGKey  string `json:"gpg_key,omitempty" yaml:"gpg_key,omitempty"`
}

type AutoSwitchProfile struct {
	Folders []string `json:"folders,omitempty" yaml:"folders,omitempty"`
}

type Profile struct {
	Name       string            `json:"name" yaml:"name"`
	Git        GitProfile        `json:"git" yaml:"git"`
	GitHub     GitHubProfile     `json:"github,omitempty" yaml:"github,omitempty"`
	SSH        SSHProfile        `json:"ssh,omitempty" yaml:"ssh,omitempty"`
	Signing    SigningProfile    `json:"signing,omitempty" yaml:"signing,omitempty"`
	AutoSwitch AutoSwitchProfile `json:"auto_switch,omitempty" yaml:"auto_switch,omitempty"`

	// Legacy fields are kept only to migrate Copilot's first JSON scaffold.
	User     string `json:"user,omitempty" yaml:"-"`
	Email    string `json:"email,omitempty" yaml:"-"`
	SSHKey   string `json:"ssh_key,omitempty" yaml:"-"`
	Provider string `json:"provider,omitempty" yaml:"-"`
}

type Config struct {
	Profiles          []Profile `json:"profiles" yaml:"profiles"`
	Active            string    `json:"active,omitempty" yaml:"active,omitempty"`
	AutoSwitchEnabled bool      `json:"auto_switch_enabled,omitempty" yaml:"auto_switch_enabled,omitempty"`
}

func ConfigDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv(envConfigDir)); override != "" {
		return ExpandPath(override)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gitid"), nil
}

func BackupDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "backups"), nil
}

func LogsDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

func LegacyJSONPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func EnsureConfigDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	for _, path := range []string{dir, filepath.Join(dir, "backups"), filepath.Join(dir, "logs")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(path); err == nil {
		cfg, err := parseYAML(data)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		return cfg, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	legacyPath, err := LegacyJSONPath()
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(legacyPath); err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("read %s: %w", legacyPath, err)
		}
		normalizeConfig(&cfg)
		return &cfg, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return &Config{Profiles: []Profile{}}, nil
}

func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("config cannot be nil")
	}
	normalizeConfig(cfg)
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func FindProfile(cfg *Config, name string) (*Profile, int) {
	if cfg == nil {
		return nil, -1
	}
	for i := range cfg.Profiles {
		if cfg.Profiles[i].Name == name {
			normalizeProfile(&cfg.Profiles[i])
			return &cfg.Profiles[i], i
		}
	}
	return nil, -1
}

func RemoveProfile(cfg *Config, name string) bool {
	_, idx := FindProfile(cfg, name)
	if idx < 0 {
		return false
	}
	cfg.Profiles = append(cfg.Profiles[:idx], cfg.Profiles[idx+1:]...)
	if cfg.Active == name {
		cfg.Active = ""
	}
	return true
}

func ExpandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func NormalizeFolder(path string) (string, error) {
	expanded, err := ExpandPath(path)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	if realPath, err := filepath.EvalSymlinks(abs); err == nil {
		abs = realPath
	}
	return filepath.Clean(abs), nil
}

func DefaultHostAlias(profileName string) string {
	slug := strings.ToLower(strings.TrimSpace(profileName))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-", "/", "-")
	slug = replacer.Replace(slug)
	if slug == "" {
		slug = "default"
	}
	return "github-" + slug
}

func parseYAML(data []byte) (*Config, error) {
	var cfg Config
	if len(strings.TrimSpace(string(data))) == 0 {
		return &Config{Profiles: []Profile{}}, nil
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	normalizeConfig(&cfg)
	return &cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.Profiles == nil {
		cfg.Profiles = []Profile{}
	}
	for i := range cfg.Profiles {
		normalizeProfile(&cfg.Profiles[i])
	}
}

func normalizeProfile(profile *Profile) {
	if profile.Git.Username == "" && profile.User != "" {
		profile.Git.Username = profile.User
	}
	if profile.Git.Email == "" && profile.Email != "" {
		profile.Git.Email = profile.Email
	}
	if profile.SSH.KeyPath == "" && profile.SSHKey != "" {
		profile.SSH.KeyPath = profile.SSHKey
	}
	if profile.GitHub.Hostname == "" {
		profile.GitHub.Hostname = "github.com"
	}
	if profile.SSH.HostName == "" {
		profile.SSH.HostName = profile.GitHub.Hostname
	}
	if profile.SSH.HostAlias == "" && profile.SSH.KeyPath != "" {
		profile.SSH.HostAlias = DefaultHostAlias(profile.Name)
	}
	profile.User = ""
	profile.Email = ""
	profile.SSHKey = ""
	profile.Provider = ""
}
