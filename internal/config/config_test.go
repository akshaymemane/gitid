package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadConfigUsesYAML(t *testing.T) {
	t.Setenv(envConfigDir, t.TempDir())

	cfg := &Config{
		Profiles: []Profile{{
			Name: "work",
			Git: GitProfile{
				Username: "Work User",
				Email:    "work@example.com",
			},
			GitHub: GitHubProfile{Username: "workhub"},
			SSH:    SSHProfile{KeyPath: "~/.ssh/work_ed25519"},
		}},
		Active:            "work",
		AutoSwitchEnabled: true,
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}
	if filepath.Base(path) != "profiles.yaml" {
		t.Fatalf("ConfigPath() = %q, want profiles.yaml", path)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	profile, _ := FindProfile(loaded, "work")
	if profile == nil {
		t.Fatal("FindProfile(work) = nil")
	}
	if profile.Git.Email != "work@example.com" {
		t.Fatalf("profile email = %q", profile.Git.Email)
	}
	if profile.GitHub.Hostname != "github.com" {
		t.Fatalf("default GitHub hostname = %q", profile.GitHub.Hostname)
	}
}

func TestLoadLegacyJSONConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envConfigDir, dir)

	legacy := `{"profiles":[{"name":"personal","user":"Personal User","email":"me@example.com","ssh_key":"~/.ssh/me"}],"active":"personal"}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	profile, _ := FindProfile(cfg, "personal")
	if profile == nil {
		t.Fatal("FindProfile(personal) = nil")
	}
	if profile.Git.Username != "Personal User" || profile.SSH.KeyPath != "~/.ssh/me" {
		t.Fatalf("legacy migration failed: %+v", profile)
	}
}
