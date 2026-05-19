package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitid/gitid/internal/config"
	"github.com/gitid/gitid/internal/git"
	"github.com/gitid/gitid/internal/github"
	"github.com/gitid/gitid/internal/ssh"
)

type switchOptions struct {
	DryRun bool
	Silent bool
}

var switchCmd = &cobra.Command{
	Use:     "switch [profile]",
	Aliases: []string{"sw"},
	Short:   "Switch to a saved identity profile",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		silent, _ := cmd.Flags().GetBool("silent")
		return switchProfile(args[0], switchOptions{DryRun: dryRun, Silent: silent})
	},
}

func switchProfile(profileName string, opts switchOptions) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	profile, _ := config.FindProfile(cfg, profileName)
	if profile == nil {
		return fmt.Errorf("profile %q not found", profileName)
	}
	gitResult, err := git.ApplyIdentity(*profile, git.ApplyOptions{DryRun: opts.DryRun})
	if err != nil {
		return err
	}
	sshResult, err := ssh.ApplyConfig(cfg, ssh.ApplyOptions{DryRun: opts.DryRun})
	if err != nil {
		return err
	}
	ghMessage, err := github.Switch(*profile, github.SwitchOptions{DryRun: opts.DryRun})
	if err != nil {
		return err
	}
	if !opts.DryRun {
		cfg.Active = profileName
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}
	}
	if opts.Silent {
		return nil
	}
	if opts.DryRun {
		appUI.Heading("Switch Preview")
		appUI.KeyValue("Profile", profileName)
	} else {
		appUI.Success("Switched to profile %s", profileName)
	}
	appUI.Section("Git")
	appUI.KeyValue("Scope", strings.TrimPrefix(gitResult.Scope, "--"))
	appUI.KeyValue("Target", gitResult.Target)
	for _, change := range gitResult.Changes {
		appUI.Bullet(change)
	}
	if len(sshResult.Changes) > 0 {
		appUI.Section("SSH")
		for _, change := range sshResult.Changes {
			appUI.Bullet(change)
		}
	}
	if ghMessage != "" {
		appUI.Section("GitHub")
		if strings.Contains(strings.ToLower(ghMessage), "skipped") {
			appUI.Warning("%s", ghMessage)
		} else {
			appUI.Success("%s", ghMessage)
		}
	}
	if verbose {
		appUI.Section("Backups")
		for _, path := range append(gitResult.Backups, sshResult.Backups...) {
			appUI.KeyValue("Backup", path)
		}
	}
	return nil
}
