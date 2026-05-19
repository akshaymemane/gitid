package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/gitid/gitid/internal/config"
	"github.com/gitid/gitid/internal/github"
	"github.com/gitid/gitid/internal/ssh"
)

var addCmd = &cobra.Command{
	Use:   "add [profile]",
	Short: "Add a new identity profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return addProfile(cmd, args)
	},
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
