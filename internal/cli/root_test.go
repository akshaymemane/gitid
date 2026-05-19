package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gitid/gitid/internal/config"
)

func TestMatchAttachedProfileChoosesLongestFolder(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	project := filepath.Join(work, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(project); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg := &config.Config{Profiles: []config.Profile{
		{Name: "base", AutoSwitch: config.AutoSwitchProfile{Folders: []string{root}}},
		{Name: "work", AutoSwitch: config.AutoSwitchProfile{Folders: []string{work}}},
	}}

	profile := matchAttachedProfile(cfg)
	if profile == nil || profile.Name != "work" {
		t.Fatalf("matchAttachedProfile() = %+v, want work", profile)
	}
}
