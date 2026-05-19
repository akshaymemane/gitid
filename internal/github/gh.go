package github

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gitid/gitid/internal/config"
)

type SwitchOptions struct {
	DryRun bool
}

func IsAvailable() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func Switch(profile config.Profile, opts SwitchOptions) (string, error) {
	if profile.GitHub.Username == "" {
		return "no GitHub username configured", nil
	}
	hostname := profile.GitHub.Hostname
	if hostname == "" {
		hostname = "github.com"
	}
	if opts.DryRun {
		return fmt.Sprintf("would run gh auth switch --hostname %s --user %s", hostname, profile.GitHub.Username), nil
	}
	if !IsAvailable() {
		return "gh CLI not installed; skipped GitHub auth switch", nil
	}
	cmd := exec.Command("gh", "auth", "switch", "--hostname", hostname, "--user", profile.GitHub.Username)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Sprintf("gh auth switch skipped: %s", strings.TrimSpace(string(output))), nil
	}
	return fmt.Sprintf("GitHub auth switched to %s", profile.GitHub.Username), nil
}

func Status(hostname string) (string, error) {
	if !IsAvailable() {
		return "", errors.New("gh CLI not installed")
	}
	if hostname == "" {
		hostname = "github.com"
	}
	cmd := exec.Command("gh", "auth", "status", "--hostname", hostname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(bytes.TrimSpace(output)), err
	}
	return string(bytes.TrimSpace(output)), nil
}
