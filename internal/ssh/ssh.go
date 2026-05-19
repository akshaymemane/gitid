package ssh

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

const (
	managedStart = "# >>> gitid managed identities"
	managedEnd   = "# <<< gitid managed identities"
)

type ApplyOptions struct {
	DryRun bool
}

type ApplyResult struct {
	Backups []string
	Changes []string
}

func EnsureKey(profile config.Profile, generate bool) error {
	if profile.SSH.KeyPath == "" || !generate {
		return nil
	}
	keyPath, err := config.ExpandPath(profile.SSH.KeyPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(keyPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return errors.New("ssh-keygen executable not found in PATH")
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return err
	}
	comment := profile.Git.Email
	if comment == "" {
		comment = "gitid-" + profile.Name
	}
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", comment, "-f", keyPath, "-N", "")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh-keygen failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func ApplyConfig(cfg *config.Config, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}
	entries := []string{}
	for _, profile := range cfg.Profiles {
		if profile.SSH.KeyPath == "" {
			continue
		}
		keyPath, err := config.ExpandPath(profile.SSH.KeyPath)
		if err != nil {
			return nil, err
		}
		hostName := profile.SSH.HostName
		if hostName == "" {
			hostName = profile.GitHub.Hostname
		}
		if hostName == "" {
			hostName = "github.com"
		}
		hostAlias := profile.SSH.HostAlias
		if hostAlias == "" {
			hostAlias = config.DefaultHostAlias(profile.Name)
		}
		entries = append(entries, fmt.Sprintf("Host %s\n  HostName %s\n  User git\n  IdentityFile %s\n  IdentitiesOnly yes", hostAlias, hostName, keyPath))
		result.Changes = append(result.Changes, fmt.Sprintf("ensure SSH host %s uses %s", hostAlias, keyPath))
	}
	if len(entries) == 0 {
		return result, nil
	}
	if opts.DryRun {
		return result, nil
	}

	sshConfig, err := sshConfigPath()
	if err != nil {
		return nil, err
	}
	if backupPath, err := backup.Create(sshConfig, "ssh_config"); err != nil {
		return nil, err
	} else if backupPath != "" {
		result.Backups = append(result.Backups, backupPath)
	}
	if err := os.MkdirAll(filepath.Dir(sshConfig), 0o700); err != nil {
		return nil, err
	}

	existing, err := os.ReadFile(sshConfig)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	updated := replaceManagedBlock(string(existing), strings.Join(entries, "\n\n"))
	if err := os.WriteFile(sshConfig, []byte(updated), 0o600); err != nil {
		return nil, err
	}
	return result, nil
}

func PublicKeyPath(privateKeyPath string) (string, error) {
	expanded, err := config.ExpandPath(privateKeyPath)
	if err != nil {
		return "", err
	}
	return expanded + ".pub", nil
}

func AgentHasKey(privateKeyPath string) (bool, error) {
	if _, err := exec.LookPath("ssh-add"); err != nil {
		return false, errors.New("ssh-add executable not found in PATH")
	}
	keyPath, err := config.ExpandPath(privateKeyPath)
	if err != nil {
		return false, err
	}
	out, err := exec.Command("ssh-add", "-l").Output()
	if err != nil {
		return false, err
	}
	return bytes.Contains(out, []byte(keyPath)) || bytes.Contains(out, []byte(filepath.Base(keyPath))), nil
}

func sshConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

func replaceManagedBlock(existing, body string) string {
	block := managedStart + "\n" + body + "\n" + managedEnd
	start := strings.Index(existing, managedStart)
	end := strings.Index(existing, managedEnd)
	if start >= 0 && end >= start {
		end += len(managedEnd)
		updated := strings.TrimRight(existing[:start], "\n") + "\n\n" + block + "\n" + strings.TrimLeft(existing[end:], "\n")
		return strings.TrimLeft(updated, "\n")
	}
	if strings.TrimSpace(existing) == "" {
		return block + "\n"
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block + "\n"
}
