package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitid/gitid/internal/backup"
	"github.com/gitid/gitid/internal/config"
	"github.com/gitid/gitid/internal/git"
	"github.com/gitid/gitid/internal/github"
	"github.com/gitid/gitid/internal/ssh"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:           "gitid",
	Short:         "Fast Git identity switching for developers",
	Long:          "gitid helps developers manage multiple Git identities across profiles, repos, SSH keys, and GitHub CLI accounts.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "gitid:", err)
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
		fmt.Printf("Initialized gitid at %s\n", dir)
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
		fmt.Printf("Config: %s\n", mustConfigPath())
		if cfg.Active == "" {
			fmt.Println("Current Profile: none")
		} else {
			fmt.Printf("Current Profile: %s\n", cfg.Active)
			if profile, _ := config.FindProfile(cfg, cfg.Active); profile != nil {
				printProfile(*profile)
			}
		}
		name, email, err := git.CurrentIdentity()
		if err != nil {
			fmt.Printf("\nGit Config:\n  %s\n", err)
			return nil
		}
		fmt.Printf("\nGit Config:\n  %s\n  %s\n", name, email)
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
				fmt.Printf("%s is already attached to %s\n", folder, profile.Name)
				return nil
			}
		}
		profile.AutoSwitch.Folders = append(profile.AutoSwitch.Folders, folder)
		sort.Strings(profile.AutoSwitch.Folders)
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}
		fmt.Printf("Attached %s to profile %s\n", folder, profile.Name)
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
			fmt.Println("Auto switching enabled in gitid config.")
			fmt.Println("Add this to your shell profile:")
			fmt.Println()
			fmt.Println(hookScript())
			return nil
		}
		path, err := shellProfilePath()
		if err != nil {
			return err
		}
		if backupPath, err := backup.Create(path, "shell_profile"); err != nil {
			return err
		} else if backupPath != "" && verbose {
			fmt.Printf("Backup: %s\n", backupPath)
		}
		existing, _ := os.ReadFile(path)
		if strings.Contains(string(existing), "gitid auto apply") {
			fmt.Printf("Shell hook already installed in %s\n", path)
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
		fmt.Printf("Installed auto-switch hook in %s\n", path)
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
		fmt.Println(hookScript())
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
				fmt.Printf("Skipped missing %s\n", item.path)
			} else {
				fmt.Printf("Backed up %s -> %s\n", item.path, backupPath)
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
				fmt.Printf("Skipped %s: %s\n", item.path, err)
				continue
			}
			fmt.Printf("Restored %s from %s\n", item.path, src)
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
	if name, err = promptIfMissing(reader, "Profile Name", name); err != nil {
		return err
	}
	if user, err = promptIfMissing(reader, "Git Username", user); err != nil {
		return err
	}
	if email, err = promptIfMissing(reader, "Git Email", email); err != nil {
		return err
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
	fmt.Printf("Added profile: %s\n", name)
	if sshKey != "" {
		if publicKey, err := ssh.PublicKeyPath(sshKey); err == nil {
			fmt.Printf("SSH key: %s\n", publicKey)
		}
	}
	if ghAuth {
		if !github.IsAvailable() {
			fmt.Println("GitHub CLI not installed; skipped gh auth login")
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
		fmt.Printf("Dry run for profile: %s\n", profileName)
	} else {
		fmt.Printf("Switched to profile: %s\n", profileName)
	}
	fmt.Printf("Git identity updated (%s: %s)\n", strings.TrimPrefix(gitResult.Scope, "--"), gitResult.Target)
	for _, change := range gitResult.Changes {
		fmt.Printf("  - %s\n", change)
	}
	if len(sshResult.Changes) > 0 {
		fmt.Println("SSH config updated")
		for _, change := range sshResult.Changes {
			fmt.Printf("  - %s\n", change)
		}
	}
	if ghMessage != "" {
		fmt.Println(ghMessage)
	}
	if verbose {
		for _, path := range append(gitResult.Backups, sshResult.Backups...) {
			fmt.Printf("Backup: %s\n", path)
		}
	}
	return nil
}

func runDoctor() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
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
		if status, err := github.Status(profile.GitHub.Hostname); err != nil {
			warn("GitHub auth", firstLine(status, err.Error()))
		} else {
			ok("GitHub auth", firstLine(status, "authenticated"))
		}
	}
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
	}
	if !cfg.AutoSwitchEnabled {
		warn("Auto switching", "disabled")
	} else {
		ok("Auto switching", "enabled")
	}
	return nil
}

func printProfile(profile config.Profile) {
	fmt.Println()
	fmt.Println("Git:")
	fmt.Printf("  %s\n", profile.Git.Username)
	fmt.Printf("  %s\n", profile.Git.Email)
	if profile.GitHub.Username != "" {
		fmt.Println("GitHub:")
		fmt.Printf("  %s\n", profile.GitHub.Username)
	}
	if profile.SSH.KeyPath != "" {
		fmt.Println("SSH:")
		fmt.Printf("  %s\n", profile.SSH.KeyPath)
	}
}

func listProfiles() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles found. Use 'gitid add work' to create one.")
		return nil
	}
	fmt.Println("Profiles:")
	for _, p := range cfg.Profiles {
		mark := " "
		if cfg.Active == p.Name {
			mark = "*"
		}
		githubUser := p.GitHub.Username
		if githubUser == "" {
			githubUser = "-"
		}
		fmt.Printf("%s %-16s %s <%s> github:%s\n", mark, p.Name, p.Git.Username, p.Git.Email, githubUser)
	}
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
	fmt.Printf("Removed profile: %s\n", name)
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
	printCheck("OK", label, detail)
}

func warn(label, detail string) {
	printCheck("WARN", label, detail)
}

func fail(label, detail string) {
	printCheck("FAIL", label, detail)
}

func printCheck(state, label, detail string) {
	if detail == "" {
		fmt.Printf("[%s] %s\n", state, label)
		return
	}
	fmt.Printf("[%s] %s: %s\n", state, label, detail)
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

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print extra diagnostics")

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
