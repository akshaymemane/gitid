package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/gitid/gitid/internal/backup"
	"github.com/gitid/gitid/internal/config"
	"github.com/gitid/gitid/internal/git"
	"github.com/gitid/gitid/internal/github"
	"github.com/gitid/gitid/internal/ssh"
	"github.com/gitid/gitid/internal/ui"
)

var verbose bool
var plain bool
var out io.Writer = os.Stdout
var errOut io.Writer = os.Stderr
var appUI = ui.New(os.Stdout, ui.Options{})

var rootCmd = &cobra.Command{
	Use:           "gitid",
	Short:         "Fast Git identity switching for developers",
	Long:          "gitid helps developers manage multiple Git identities across profiles, repos, SSH keys, and GitHub CLI accounts.",
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

var addCmd = &cobra.Command{
	Use:   "add [profile]",
	Short: "Add a new identity profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return addProfile(cmd, args)
	},
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage gitid profiles",
}

var profileAddCmd = &cobra.Command{
	Use:   "add [profile]",
	Short: "Add a new identity profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return addProfile(cmd, args)
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

var profileListCmd = &cobra.Command{
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

var profileRemoveCmd = &cobra.Command{
	Use:     "remove [profile]",
	Aliases: []string{"rm"},
	Short:   "Remove a saved identity profile",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeProfile(args[0])
	},
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

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose GitID, Git, SSH, and GitHub CLI configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
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

type switchOptions struct {
	DryRun bool
	Silent bool
}

func addProfile(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	name, _ := cmd.Flags().GetString("name")
	if len(args) > 0 && name == "" {
		name = args[0]
	}
	user, _ := cmd.Flags().GetString("user")
	email, _ := cmd.Flags().GetString("email")
	githubUser, _ := cmd.Flags().GetString("github")
	sshKey, _ := cmd.Flags().GetString("ssh-key")
	host, _ := cmd.Flags().GetString("host")
	hostAlias, _ := cmd.Flags().GetString("host-alias")
	generateSSH, _ := cmd.Flags().GetBool("generate-ssh-key")
	ghAuth, _ := cmd.Flags().GetBool("gh-auth")
	signing, _ := cmd.Flags().GetBool("signing")
	gpgKey, _ := cmd.Flags().GetString("gpg-key")

	var err error
	if name == "" || user == "" || email == "" {
		if isTerminal() && !appUI.Plain() {
			if err := runAddForm(&name, &user, &email, &githubUser, &sshKey, &generateSSH, &ghAuth); err != nil {
				return err
			}
		} else {
			if name, err = promptIfMissing(reader, "Profile Name", name); err != nil {
				return err
			}
			if user, err = promptIfMissing(reader, "Git Username", user); err != nil {
				return err
			}
			if email, err = promptIfMissing(reader, "Git Email", email); err != nil {
				return err
			}
		}
	}
	if githubUser == "" {
		githubUser = name
	}
	if host == "" {
		host = "github.com"
	}
	if generateSSH && sshKey == "" {
		sshKey = "~/.ssh/gitid_" + sanitizeName(name) + "_ed25519"
	}
	if hostAlias == "" && sshKey != "" {
		hostAlias = config.DefaultHostAlias(name)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if existing, _ := config.FindProfile(cfg, name); existing != nil {
		return fmt.Errorf("profile %q already exists", name)
	}
	profile := config.Profile{
		Name: name,
		Git: config.GitProfile{
			Username: user,
			Email:    email,
		},
		GitHub: config.GitHubProfile{
			Username: githubUser,
			Hostname: host,
		},
		SSH: config.SSHProfile{
			KeyPath:   sshKey,
			HostName:  host,
			HostAlias: hostAlias,
		},
		Signing: config.SigningProfile{
			Enabled: signing,
			GPGKey:  gpgKey,
		},
	}
	if err := ssh.EnsureKey(profile, generateSSH); err != nil {
		return err
	}
	cfg.Profiles = append(cfg.Profiles, profile)
	if err := config.SaveConfig(cfg); err != nil {
		return err
	}
	if _, err := ssh.ApplyConfig(cfg, ssh.ApplyOptions{}); err != nil {
		return err
	}
	appUI.Success("Added profile %s", name)
	appUI.KeyValue("Git", fmt.Sprintf("%s <%s>", user, email))
	if githubUser != "" {
		appUI.KeyValue("GitHub", githubUser)
	}
	if sshKey != "" {
		if publicKey, err := ssh.PublicKeyPath(sshKey); err == nil {
			appUI.KeyValue("SSH public key", publicKey)
		}
	}
	if ghAuth {
		if !github.IsAvailable() {
			appUI.Warning("GitHub CLI not installed; skipped gh auth login")
		} else {
			login := exec.Command("gh", "auth", "login", "--hostname", host)
			login.Stdin = os.Stdin
			login.Stdout = os.Stdout
			login.Stderr = os.Stderr
			if err := login.Run(); err != nil {
				return err
			}
		}
	}
	return nil
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

func runAddForm(name, user, email, githubUser, sshKey *string, generateSSH, ghAuth *bool) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Description("A short name like work, personal, or oss.").
				Value(name).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().
				Title("Git author name").
				Description("Used as git user.name.").
				Value(user).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().
				Title("Git email").
				Description("Used as git user.email.").
				Value(email).
				Validate(huh.ValidateNotEmpty()),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("GitHub username").
				Description("Optional. Defaults to the profile name.").
				Value(githubUser),
			huh.NewInput().
				Title("SSH private key path").
				Description("Optional. Example: ~/.ssh/work_ed25519").
				Value(sshKey),
			huh.NewConfirm().
				Title("Generate SSH key if missing?").
				Affirmative("Yes").
				Negative("No").
				Value(generateSSH),
			huh.NewConfirm().
				Title("Run GitHub CLI auth after adding?").
				Affirmative("Yes").
				Negative("No").
				Value(ghAuth),
		),
	).WithTheme(huh.ThemeBase())
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errors.New("profile creation cancelled")
		}
		return err
	}
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

func promptIfMissing(reader *bufio.Reader, label, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value != "" {
		return value, nil
	}
	if !isTerminal() {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	fmt.Printf("%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return line, nil
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
	appUI = ui.New(out, ui.Options{Plain: plain})
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print extra diagnostics")
	rootCmd.PersistentFlags().BoolVar(&plain, "plain", false, "Force plain output without colors or symbols")

	addProfileFlags(addCmd)
	addProfileFlags(profileAddCmd)
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
	rootCmd.AddCommand(profileCmd)

	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileRemoveCmd)
	autoCmd.AddCommand(autoEnableCmd)
	autoCmd.AddCommand(autoDisableCmd)
	autoCmd.AddCommand(autoHookCmd)
	autoCmd.AddCommand(autoApplyCmd)
}

func addProfileFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Profile name")
	cmd.Flags().String("user", "", "Git user.name")
	cmd.Flags().String("email", "", "Git user.email")
	cmd.Flags().String("github", "", "GitHub username")
	cmd.Flags().String("ssh-key", "", "SSH private key path")
	cmd.Flags().String("host", "github.com", "Git provider hostname")
	cmd.Flags().String("host-alias", "", "SSH host alias to manage")
	cmd.Flags().Bool("generate-ssh-key", false, "Generate an ed25519 SSH key when missing")
	cmd.Flags().Bool("gh-auth", false, "Run gh auth login after creating the profile")
	cmd.Flags().Bool("signing", false, "Enable GPG commit signing for this profile")
	cmd.Flags().String("gpg-key", "", "GPG signing key id")
}
