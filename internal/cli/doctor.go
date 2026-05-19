package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitid/gitid/internal/config"
	"github.com/gitid/gitid/internal/git"
	"github.com/gitid/gitid/internal/github"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose GitID, Git, SSH, and GitHub CLI configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	appUI.Heading("GitID Doctor")
	appUI.Section("Core")
	ok("Config readable", mustConfigPath())
	if git.IsAvailable() {
		ok("Git installed", "")
	} else {
		fail("Git installed", "git executable not found in PATH")
	}
	name, email, err := git.CurrentIdentity()
	if err != nil {
		warn("Git identity", err.Error())
	} else {
		ok("Git identity", fmt.Sprintf("%s <%s>", name, email))
	}
	appUI.Section("Profile")
	if cfg.Active == "" {
		warn("Active profile", "none")
	} else if profile, _ := config.FindProfile(cfg, cfg.Active); profile == nil {
		fail("Active profile", fmt.Sprintf("%q is missing", cfg.Active))
	} else {
		ok("Active profile", cfg.Active)
		if profile.SSH.KeyPath != "" {
			keyPath, _ := config.ExpandPath(profile.SSH.KeyPath)
			if _, err := os.Stat(keyPath); err == nil {
				ok("SSH key", keyPath)
			} else {
				fail("SSH key", fmt.Sprintf("%s not found", keyPath))
			}
		}
		if _, hasAgent := os.LookupEnv("SSH_AUTH_SOCK"); hasAgent {
			ok("SSH agent", "SSH_AUTH_SOCK is set")
		} else {
			warn("SSH agent", "SSH_AUTH_SOCK is not set")
		}
		appUI.Section("GitHub")
		if status, err := github.Status(profile.GitHub.Hostname); err != nil {
			warn("GitHub auth", firstLine(status, err.Error()))
		} else {
			ok("GitHub auth", firstLine(status, "authenticated"))
		}
	}
	appUI.Section("Repository")
	if git.IsInsideGitRepo() {
		if remote, err := git.RemoteURL(); err == nil {
			if strings.HasPrefix(remote, "https://") {
				warn("Remote origin", "uses HTTPS; SSH identity switching may not affect pushes")
			} else {
				ok("Remote origin", remote)
			}
		} else {
			warn("Remote origin", "origin remote not configured")
		}
	} else {
		warn("Git repository", "not inside a Git worktree")
	}
	appUI.Section("Automation")
	if !cfg.AutoSwitchEnabled {
		warn("Auto switching", "disabled")
	} else {
		ok("Auto switching", "enabled")
	}
	printDoctorHints(cfg)
	return nil
}
