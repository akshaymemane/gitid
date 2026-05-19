# `gitid add`

Create a new identity profile.

```bash
gitid add work
```

When required values are missing and the terminal is interactive, GitID opens a guided form.

## Flag-Based Usage

```bash
gitid add work \
  --user "Work Name" \
  --email work@example.com \
  --github work-github \
  --ssh-key ~/.ssh/work_ed25519
```

## Options

| Option | Description |
| --- | --- |
| `--name` | Profile name. Usually supplied as the positional argument. |
| `--user` | Git `user.name` for commits. |
| `--email` | Git `user.email` for commits. |
| `--github` | GitHub username for `gh auth switch`. |
| `--ssh-key` | SSH private key path for this profile. |
| `--host` | Git provider hostname. Defaults to `github.com`. |
| `--host-alias` | SSH host alias managed by GitID. |
| `--generate-ssh-key` | Generate an ed25519 key if the configured key is missing. |
| `--gh-auth` | Run `gh auth login` after creating the profile. |
| `--signing` | Enable GPG commit signing for this profile. |
| `--gpg-key` | GPG signing key id. |

## What Gets Stored

Profiles are stored locally in `~/.config/gitid/profiles.yaml`.

GitID stores metadata only. It does not store GitHub tokens, passwords, or private key contents.

