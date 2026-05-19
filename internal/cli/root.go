package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitid/gitid/internal/backup"
	"github.com/gitid/gitid/internal/config"
	"github.com/gitid/gitid/internal/git"
	"github.com/gitid/gitid/internal/ssh"
	"github.com/gitid/gitid/internal/ui"
)

var version = "0.3.0"

var verbose bool
var plain bool
var colorMode string
var out io.Writer = os.Stdout
var errOut io.Writer = os.Stderr
var appUI = ui.New(os.Stdout, ui.Options{})

var rootCmd = &cobra.Command{
	Use:           "gitid",
	Short:         "Fast Git identity switching for developers",
	Long:          "gitid helps developers manage multiple Git identities across profiles, repos, SSH keys, and GitHub CLI accounts.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configureUI()
	},
}

func Execute() {
	configureUI()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(errOut, "gitid:", err)
		os.Exit(1)
	}
}

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"init"},
	Short:   "Initialize gitid configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.EnsureConfigDir()
		if err != nil {
			return err
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}
		appUI.Success("Initialized gitid")
		appUI.KeyValue("Config", appUI.Path(dir))
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List saved identity profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listProfiles()
	},
}

var removeCmd = &cobra.Command{
	Use:     "remove [profile]",
	Aliases: []string{"rm"},
	Short:   "Remove a saved identity profile",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeProfile(args[0])
	},
}

var currentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"status"},
	Short:   "Show the active gitid profile and Git identity",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		appUI.Heading("Current Identity")
		appUI.KeyValue("Config", mustConfigPath())
		if cfg.Active == "" {
			appUI.Warning("Current profile is not set")
		} else {
			appUI.KeyValue("Current Profile", cfg.Active)
			if profile, _ := config.FindProfile(cfg, cfg.Active); profile != nil {
				printProfile(*profile)
			}
		}
		name, email, err := git.CurrentIdentity()
		if err != nil {
			appUI.Section("Git Config")
			appUI.Warning("%s", err)
			return nil
		}
		appUI.Section("Git Config")
		appUI.KeyValue("Name", name)
		appUI.KeyValue("Email", email)
		return nil
	},
}

var attachCmd = &cobra.Command{
	Use:   "attach [profile] [folder]",
	Short: "Attach a folder to a profile for automatic switching",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		profile, _ := config.FindProfile(cfg, args[0])
		if profile == nil {
			return fmt.Errorf("profile %q not found", args[0])
		}
		folder, err := config.NormalizeFolder(args[1])
		if err != nil {
			return err
		}
		for _, existing := range profile.AutoSwitch.Folders {
			if existing == folder {
				appUI.Warning("%s is already attached to %s", appUI.Path(folder), appUI.ProfileName(profile.Name))
				return nil
			}
		}
		profile.AutoSwitch.Folders = append(profile.AutoSwitch.Folders, folder)
		sort.Strings(profile.AutoSwitch.Folders)
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}
		appUI.Success("Attached folder")
		appUI.KeyValue("Profile", profile.Name)
		appUI.KeyValue("Folder", folder)
		return nil
	},
}

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Manage folder-based automatic switching",
}

var autoEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable automatic switching in your current shell",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		cfg.AutoSwitchEnabled = true
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}
		install, _ := cmd.Flags().GetBool("install")
		if !install {
			appUI.Success("Auto switching enabled")
			appUI.Hint("Add the shell hook below to your shell profile")
			appUI.Println()
			appUI.Println(hookScript())
			return nil
		}
		path, err := shellProfilePath()
		if err != nil {
			return err
		}
		if backupPath, err := backup.Create(path, "shell_profile"); err != nil {
			return err
		} else if backupPath != "" && verbose {
			appUI.KeyValue("Backup", backupPath)
		}
		existing, _ := os.ReadFile(path)
		if strings.Contains(string(existing), "gitid auto apply") {
			appUI.Warning("Shell hook already installed")
			appUI.KeyValue("File", path)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := fmt.Fprintf(f, "\n# gitid automatic switching\n%s\n", hookScript()); err != nil {
			return err
		}
		appUI.Success("Installed auto-switch hook")
		appUI.KeyValue("File", path)
		return nil
	},
}

var autoDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable automatic switching",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		cfg.AutoSwitchEnabled = false
		return config.SaveConfig(cfg)
	},
}

var autoHookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Print the shell hook for automatic switching",
	Run: func(cmd *cobra.Command, args []string) {
		appUI.Println(hookScript())
	},
}

var autoApplyCmd = &cobra.Command{
	Use:    "apply",
	Short:  "Apply the profile attached to the current directory",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		silent, _ := cmd.Flags().GetBool("silent")
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		if !cfg.AutoSwitchEnabled {
			return nil
		}
		profile := matchAttachedProfile(cfg)
		if profile == nil {
			return nil
		}
		return switchProfile(profile.Name, switchOptions{Silent: silent})
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Back up ~/.gitconfig and ~/.ssh/config",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		for _, item := range []struct{ path, label string }{
			{filepath.Join(home, ".gitconfig"), "gitconfig"},
			{filepath.Join(home, ".ssh", "config"), "ssh_config"},
		} {
			backupPath, err := backup.Create(item.path, item.label)
			if err != nil {
				return err
			}
			if backupPath == "" {
				appUI.Warning("Skipped missing %s", item.path)
			} else {
				appUI.Success("Backed up %s", item.label)
				appUI.KeyValue("Source", item.path)
				appUI.KeyValue("Backup", backupPath)
			}
		}
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore the latest ~/.gitconfig and ~/.ssh/config backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		for _, item := range []struct{ path, label string }{
			{filepath.Join(home, ".gitconfig"), "gitconfig"},
			{filepath.Join(home, ".ssh", "config"), "ssh_config"},
		} {
			src, err := backup.RestoreLatest(item.label, item.path)
			if err != nil {
				appUI.Warning("Skipped %s: %s", item.path, err)
				continue
			}
			appUI.Success("Restored %s", item.label)
			appUI.KeyValue("Target", item.path)
			appUI.KeyValue("Backup", src)
		}
		return nil
	},
}

func printProfile(profile config.Profile) {
	appUI.Section("Profile Details")
	appUI.KeyValue("Git name", profile.Git.Username)
	appUI.KeyValue("Git email", profile.Git.Email)
	if profile.GitHub.Username != "" {
		appUI.KeyValue("GitHub", profile.GitHub.Username)
	}
	if profile.SSH.KeyPath != "" {
		appUI.KeyValue("SSH key", profile.SSH.KeyPath)
	}
}

func listProfiles() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		appUI.Warning("No profiles found")
		appUI.Hint("gitid add work")
		return nil
	}
	appUI.Heading("Profiles")
	rows := make([][]string, 0, len(cfg.Profiles))
	for _, p := range cfg.Profiles {
		mark := " "
		if cfg.Active == p.Name {
			mark = "*"
		}
		githubUser := p.GitHub.Username
		if githubUser == "" {
			githubUser = "-"
		}
		sshKey := "no"
		if p.SSH.KeyPath != "" {
			sshKey = "yes"
		}
		rows = append(rows, []string{mark, p.Name, fmt.Sprintf("%s <%s>", p.Git.Username, p.Git.Email), githubUser, sshKey})
	}
	appUI.Println(appUI.ProfilesTable(rows))
	return nil
}

func removeProfile(name string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if !config.RemoveProfile(cfg, name) {
		return fmt.Errorf("profile %q not found", name)
	}
	if err := config.SaveConfig(cfg); err != nil {
		return err
	}
	if _, err := ssh.ApplyConfig(cfg, ssh.ApplyOptions{}); err != nil {
		return err
	}
	appUI.Success("Removed profile %s", name)
	return nil
}

func matchAttachedProfile(cfg *config.Config) *config.Profile {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	cwd, err = config.NormalizeFolder(cwd)
	if err != nil {
		return nil
	}
	var best *config.Profile
	bestLen := -1
	for i := range cfg.Profiles {
		for _, folder := range cfg.Profiles[i].AutoSwitch.Folders {
			normalized, err := config.NormalizeFolder(folder)
			if err != nil {
				continue
			}
			if cwd == normalized || strings.HasPrefix(cwd, normalized+string(os.PathSeparator)) {
				if len(normalized) > bestLen {
					best = &cfg.Profiles[i]
					bestLen = len(normalized)
				}
			}
		}
	}
	return best
}

func hookScript() string {
	return strings.TrimSpace(`
_gitid_auto_switch() {
  command gitid auto apply --silent >/dev/null 2>&1 || true
}
case "$SHELL" in
  *zsh*)
    autoload -U add-zsh-hook 2>/dev/null || true
    add-zsh-hook chpwd _gitid_auto_switch 2>/dev/null || true
    ;;
  *)
    _gitid_cd() { builtin cd "$@" && _gitid_auto_switch; }
    alias cd=_gitid_cd
    ;;
esac
_gitid_auto_switch
`)
}

func shellProfilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return filepath.Join(home, ".zshrc"), nil
	}
	if strings.Contains(shell, "bash") {
		return filepath.Join(home, ".bashrc"), nil
	}
	return "", errors.New("unsupported shell for --install; use 'gitid auto hook' and add it manually")
}

func mustConfigPath() string {
	path, err := config.ConfigPath()
	if err != nil {
		return "unknown"
	}
	return path
}

func isTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(" ", "_", "-", "_", ".", "_", "/", "_")
	name = replacer.Replace(name)
	if name == "" {
		return "profile"
	}
	return name
}

func ok(label, detail string) {
	appUI.Check(ui.CheckOK, label, detail)
}

func warn(label, detail string) {
	appUI.Check(ui.CheckWarn, label, detail)
}

func fail(label, detail string) {
	appUI.Check(ui.CheckFail, label, detail)
}

func printDoctorHints(cfg *config.Config) {
	hints := []string{}
	if cfg.Active == "" {
		hints = append(hints, "Set an active profile with gitid switch <profile>.")
	}
	if !cfg.AutoSwitchEnabled {
		hints = append(hints, "Enable folder switching with gitid auto enable.")
	}
	if len(hints) == 0 {
		return
	}
	appUI.Section("Suggested Fixes")
	for _, hint := range hints {
		appUI.Bullet(hint)
	}
}

func firstLine(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return value[:idx]
	}
	return value
}

func configureUI() {
	appUI = ui.New(out, ui.Options{Plain: plain, Color: colorMode})
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print extra diagnostics")
	rootCmd.PersistentFlags().BoolVar(&plain, "plain", false, "Force plain output without colors or symbols")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "Color output: auto, always, or never")

	addProfileFlags(addCmd)
	switchCmd.Flags().Bool("dry-run", false, "Show changes without applying them")
	switchCmd.Flags().Bool("silent", false, "Suppress normal output")
	switchCmd.Flags().MarkHidden("silent")
	autoEnableCmd.Flags().Bool("install", false, "Append the shell hook to your shell profile")
	autoApplyCmd.Flags().Bool("silent", false, "Suppress normal output")

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(autoCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)

	autoCmd.AddCommand(autoEnableCmd)
	autoCmd.AddCommand(autoDisableCmd)
	autoCmd.AddCommand(autoHookCmd)
	autoCmd.AddCommand(autoApplyCmd)
}
