package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gitid/gitid/internal/config"
	"github.com/gitid/gitid/internal/ui"
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

func TestListProfilesPlainOutput(t *testing.T) {
	t.Setenv("GITID_CONFIG_DIR", t.TempDir())
	cfg := &config.Config{
		Active: "work",
		Profiles: []config.Profile{{
			Name:   "work",
			Git:    config.GitProfile{Username: "Work User", Email: "work@example.com"},
			GitHub: config.GitHubProfile{Username: "workhub"},
			SSH:    config.SSHProfile{KeyPath: "~/.ssh/work_ed25519"},
		}},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	var buf bytes.Buffer
	withTestUI(&buf, true, func() {
		if err := listProfiles(); err != nil {
			t.Fatalf("listProfiles() error = %v", err)
		}
	})

	got := buf.String()
	for _, want := range []string{"Profiles", "Active", "work", "Work User <work@example.com>", "workhub", "yes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("listProfiles() output missing %q:\n%s", want, got)
		}
	}
}

func TestDoctorPlainOutput(t *testing.T) {
	t.Setenv("GITID_CONFIG_DIR", t.TempDir())
	if err := config.SaveConfig(&config.Config{}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	var buf bytes.Buffer
	withTestUI(&buf, true, func() {
		if err := runDoctor(); err != nil {
			t.Fatalf("runDoctor() error = %v", err)
		}
	})

	got := buf.String()
	for _, want := range []string{"GitID Doctor", "[OK] Config readable", "Suggested Fixes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("runDoctor() output missing %q:\n%s", want, got)
		}
	}
}

func TestSwitchDryRunPlainOutput(t *testing.T) {
	t.Setenv("GITID_CONFIG_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cfg := &config.Config{
		Profiles: []config.Profile{{
			Name: "work",
			Git:  config.GitProfile{Username: "Work User", Email: "work@example.com"},
		}},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	var buf bytes.Buffer
	withTestUI(&buf, true, func() {
		if err := switchProfile("work", switchOptions{DryRun: true}); err != nil {
			t.Fatalf("switchProfile() error = %v", err)
		}
	})

	got := buf.String()
	for _, want := range []string{"Switch Preview", "Profile: work", "set user.name=\"Work User\"", "set user.email=\"work@example.com\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("switchProfile() output missing %q:\n%s", want, got)
		}
	}
}

func TestAddProfileWithFlagsNonInteractive(t *testing.T) {
	t.Setenv("GITID_CONFIG_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cmd := &cobra.Command{}
	addProfileFlags(cmd)
	if err := cmd.Flags().Set("user", "Work User"); err != nil {
		t.Fatalf("set user flag: %v", err)
	}
	if err := cmd.Flags().Set("email", "work@example.com"); err != nil {
		t.Fatalf("set email flag: %v", err)
	}
	if err := cmd.Flags().Set("github", "workhub"); err != nil {
		t.Fatalf("set github flag: %v", err)
	}

	var buf bytes.Buffer
	withTestUI(&buf, true, func() {
		if err := addProfile(cmd, []string{"work"}); err != nil {
			t.Fatalf("addProfile() error = %v", err)
		}
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	profile, _ := config.FindProfile(cfg, "work")
	if profile == nil {
		t.Fatal("FindProfile(work) = nil")
	}
	if profile.Git.Email != "work@example.com" || profile.GitHub.Username != "workhub" {
		t.Fatalf("profile mismatch: %+v", profile)
	}
	if !strings.Contains(buf.String(), "[OK] Added profile work") {
		t.Fatalf("addProfile() output mismatch:\n%s", buf.String())
	}
}

func withTestUI(buf *bytes.Buffer, usePlain bool, fn func()) {
	oldUI, oldOut, oldPlain := appUI, out, plain
	defer func() {
		appUI, out, plain = oldUI, oldOut, oldPlain
	}()
	out = buf
	plain = usePlain
	appUI = ui.New(buf, ui.Options{Plain: usePlain})
	fn()
}
