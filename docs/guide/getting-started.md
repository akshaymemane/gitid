# Getting Started

Install GitID from npm:

```bash
npm install -g @akshaymemane/git-id
```

The npm package installs the `gitid` command.

## Initialize GitID

```bash
gitid setup
```

This creates GitID's local config directory at `~/.config/gitid`.

## Add Your First Profile

Use the guided form:

```bash
gitid add work
```

Or pass everything as flags:

```bash
gitid add work \
  --user "Work Name" \
  --email work@example.com \
  --github work-github \
  --ssh-key ~/.ssh/work_ed25519
```

## Preview A Switch

```bash
gitid switch work --dry-run
```

Dry-run mode shows what GitID would change without applying it.

## Switch Profiles

```bash
gitid switch work
```

GitID applies the Git identity, updates SSH integration, and attempts a GitHub CLI auth switch when a GitHub username is configured.

## Check Your Setup

```bash
gitid doctor
```

Use `doctor` whenever Git, SSH, or GitHub auth does not feel right.

