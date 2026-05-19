package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gitid/gitid/internal/backup"
	"github.com/gitid/gitid/internal/config"
)

type ApplyOptions struct {
	DryRun bool
}

type ApplyResult struct {
	Scope   string
	Target  string
	Changes []string
	Backups []string
}

func IsAvailable() bool {
	return exec.Command("git", "--version").Run() == nil
}

func RequireGit() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git executable not found in PATH")
	}
	return nil
}

func IsInsideGitRepo() bool {
	return exec.Command("git", "rev-parse", "--is-inside-work-tree").Run() == nil
}

func ApplyIdentity(profile config.Profile, opts ApplyOptions) (*ApplyResult, error) {
	if err := RequireGit(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(profile.Git.Username) == "" || strings.TrimSpace(profile.Git.Email) == "" {
		return nil, fmt.Errorf("profile %q must include git username and email", profile.Name)
	}

	scope, target, err := configScope()
	if err != nil {
		return nil, err
	}
	result := &ApplyResult{
		Scope:  scope,
		Target: target,
		Changes: []string{
			fmt.Sprintf("set user.name=%q", profile.Git.Username),
			fmt.Sprintf("set user.email=%q", profile.Git.Email),
		},
	}

	if profile.SSH.KeyPath != "" {
		keyPath, err := config.ExpandPath(profile.SSH.KeyPath)
		if err != nil {
			return nil, err
		}
		result.Changes = append(result.Changes, fmt.Sprintf("set core.sshCommand=%q", sshCommand(keyPath)))
	} else {
		result.Changes = append(result.Changes, "unset core.sshCommand if present")
	}

	if profile.Signing.Enabled {
		result.Changes = append(result.Changes, "enable commit signing")
		if profile.Signing.GPGKey != "" {
			result.Changes = append(result.Changes, fmt.Sprintf("set user.signingkey=%q", profile.Signing.GPGKey))
		}
	}

	if opts.DryRun {
		return result, nil
	}

	backupLabel := "gitconfig"
	if scope == "--local" {
		backupLabel = "repo_gitconfig"
	}
	if backupPath, err := backup.Create(target, backupLabel); err != nil {
		return nil, err
	} else if backupPath != "" {
		result.Backups = append(result.Backups, backupPath)
	}

	applied := []string{}
	rollback := func(cause error) error {
		if len(result.Backups) == 0 {
			return cause
		}
		if err := restoreFile(result.Backups[len(result.Backups)-1], target); err != nil {
			return fmt.Errorf("%w; rollback failed: %v", cause, err)
		}
		return cause
	}

	if err := runGitConfig(scope, "user.name", profile.Git.Username); err != nil {
		return nil, rollback(err)
	}
	applied = append(applied, "user.name")
	if err := runGitConfig(scope, "user.email", profile.Git.Email); err != nil {
		return nil, rollback(err)
	}
	applied = append(applied, "user.email")

	if profile.SSH.KeyPath != "" {
		keyPath, err := config.ExpandPath(profile.SSH.KeyPath)
		if err != nil {
			return nil, rollback(err)
		}
		if err := runGitConfig(scope, "core.sshCommand", sshCommand(keyPath)); err != nil {
			return nil, rollback(err)
		}
		applied = append(applied, "core.sshCommand")
	} else if value, _ := ConfigValue("core.sshCommand"); strings.TrimSpace(value) != "" {
		if err := unsetGitConfig(scope, "core.sshCommand"); err != nil {
			return nil, rollback(err)
		}
		applied = append(applied, "core.sshCommand")
	}

	if profile.Signing.Enabled {
		if err := runGitConfig(scope, "commit.gpgsign", "true"); err != nil {
			return nil, rollback(err)
		}
		applied = append(applied, "commit.gpgsign")
		if profile.Signing.GPGKey != "" {
			if err := runGitConfig(scope, "user.signingkey", profile.Signing.GPGKey); err != nil {
				return nil, rollback(err)
			}
			applied = append(applied, "user.signingkey")
		}
	}
	_ = applied
	return result, nil
}

func CurrentIdentity() (string, string, error) {
	if err := RequireGit(); err != nil {
		return "", "", err
	}
	name, err := ConfigValue("user.name")
	if err != nil {
		name = ""
	}
	email, err := ConfigValue("user.email")
	if err != nil {
		email = ""
	}
	if name == "" && email == "" {
		return "", "", errors.New("git user.name and user.email are not configured")
	}
	return name, email, nil
}

func ConfigValue(key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

func RemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

func configScope() (scope, target string, err error) {
	if IsInsideGitRepo() {
		out, err := exec.Command("git", "rev-parse", "--git-path", "config").Output()
		if err != nil {
			return "", "", err
		}
		target = strings.TrimSpace(string(out))
		if !filepath.IsAbs(target) {
			wd, err := os.Getwd()
			if err != nil {
				return "", "", err
			}
			target = filepath.Join(wd, target)
		}
		return "--local", target, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	return "--global", filepath.Join(home, ".gitconfig"), nil
}

func runGitConfig(scope, key, value string) error {
	cmd := exec.Command("git", "config", scope, key, value)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config %s failed: %s", key, string(bytes.TrimSpace(output)))
	}
	return nil
}

func unsetGitConfig(scope, key string) error {
	cmd := exec.Command("git", "config", scope, "--unset", key)
	if output, err := cmd.CombinedOutput(); err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			trimmed = err.Error()
		}
		return fmt.Errorf("git config unset %s failed: %s", key, trimmed)
	}
	return nil
}

func sshCommand(keyPath string) string {
	return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", shellQuote(keyPath))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func restoreFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o600)
}
