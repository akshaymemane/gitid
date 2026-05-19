# GitID PRD

### Fast Git Identity Switching for Developers

---

## 1. Overview

### Product Name

`gitid`

### Tagline

> Fast Git identity switching for developers using multiple accounts.

### Problem Statement

Developers frequently work with:

* work GitHub accounts
* personal GitHub accounts
* open-source identities
* multiple SSH keys
* multiple Git providers

Managing these identities today is fragmented and error-prone.

Common issues:

* committing with wrong email
* wrong SSH key used during push
* GitHub CLI auth conflicts
* manual git config switching
* directory-based hacks using `includeIf`
* shared laptops causing credential conflicts

Git technically supports multiple identities, but the developer experience is poor and highly manual.

---

# 2. Vision

Create the easiest and safest way to:

* create Git identities
* switch identities
* auto-switch identities
* manage SSH + GitHub auth
* prevent accidental commits with wrong credentials

The tool should feel:

* lightweight
* fast
* terminal-native
* beginner friendly
* trustworthy

---

# 3. Goals

## Primary Goals

* One-command setup
* Fast profile switching
* Auto-switching by project/folder/repo
* SSH + GitHub CLI orchestration
* Cross-platform support
* Zero manual SSH config editing

---

# 4. Non Goals (V1)

Not intended to become:

* enterprise IAM platform
* secrets manager
* cloud credential orchestrator
* full Git GUI

No AI features.
No SaaS dependency.
No account sync.

---

# 5. Target Users

## Primary

Developers with:

* work + personal GitHub accounts
* multiple Git providers
* shared machines
* consulting/freelancing workflows

## Secondary

* OSS contributors
* DevOps engineers
* students using college + personal accounts

---

# 6. Existing Solutions

## Git Native

* `git config`
* `includeIf`
* SSH config
* local repo config

Powerful but difficult and manual.

---

## GitHub CLI

Supports multi-account switching:

```bash
gh auth switch
```

But does not orchestrate:

* SSH keys
* repo mappings
* git user/email
* auto switching

---

## Existing Tools

* `gguser`
* `git-identity`
* `git-toggler`
* `git-account-switcher`

Most lack:

* polished UX
* automatic switching
* full credential orchestration
* safety tooling

---

# 7. Core Product Philosophy

The product is NOT the switching logic.

The product is:

* confidence
* simplicity
* predictability
* low friction

---

# 8. Tech Stack

## CLI Runtime

Go

Reason:

* static binaries
* cross-platform
* fast startup
* easy shell integration
* SSH/process handling

---

## Distribution

### Preferred Installation

```bash
npm install -g gitid
```

npm acts as:

* installation layer
* update mechanism
* discoverability

Internally installs platform-specific Go binaries.

---

# 9. Supported Platforms

## V1

* macOS
* Linux

## V2

* Windows

---

# 10. High-Level Architecture

```text
CLI Layer
   ↓
Profile Manager
   ↓
Git Config Manager
SSH Manager
GitHub CLI Manager
Shell Hook Engine
   ↓
Filesystem + Native Git Tools
```

---

# 11. Identity Profile Model

Each profile contains:

```yaml
name: work

git:
  username: Akshay Memane
  email: akshay@company.com

github:
  username: akshay-work

ssh:
  key_path: ~/.ssh/work_ed25519

signing:
  enabled: true
  gpg_key: ABC123

auto_switch:
  folders:
    - ~/office
    - ~/company
```

---

# 12. Core Features

## 12.1 Add Profile

### Command

```bash
gitid add work
```

### Interactive Flow

```text
Profile Name: work
Git Username: Akshay Memane
Git Email: akshay@company.com
Auth Type: SSH
Generate SSH Key? Yes
Authenticate GitHub CLI? Yes
```

### Actions

* create profile
* optionally generate SSH key
* update SSH config safely
* authenticate GitHub CLI
* store metadata

---

## 12.2 Switch Profile

### Command

```bash
gitid switch work
```

### Actions

* update active Git config
* activate SSH identity
* switch GitHub CLI auth
* update shell environment if needed

### Output

```text
✔ Switched to profile: work
✔ Git identity updated
✔ SSH identity activated
✔ GitHub auth switched
```

---

## 12.3 Current Profile

### Command

```bash
gitid current
```

### Output

```text
Current Profile: work

Git:
  Akshay Memane
  akshay@company.com

GitHub:
  akshay-work

SSH:
  ~/.ssh/work_ed25519
```

---

## 12.4 List Profiles

### Command

```bash
gitid list
```

---

## 12.5 Remove Profile

### Command

```bash
gitid remove work
```

---

## 12.6 Attach Folder

### Command

```bash
gitid attach work ~/office
```

Maps directories to identities.

---

## 12.7 Automatic Switching

### Command

```bash
gitid auto enable
```

### Behavior

When user enters:

```bash
cd ~/office/project-a
```

GitID automatically:

* switches profile
* activates SSH identity
* updates Git identity

without manual intervention.

---

## 12.8 Help System

### Command

```bash
gitid help
```

Must provide:

* concise documentation
* examples
* subcommand help

Example:

```bash
gitid switch --help
```

---

## 12.9 Doctor Command

### Command

```bash
gitid doctor
```

### Purpose

Diagnose auth/config issues.

Checks:

* git config
* SSH agent
* GitHub auth
* remote URLs
* SSH connectivity
* conflicting configs

Example Output:

```text
✔ Git configured
✔ SSH key loaded
✖ GitHub auth expired
⚠ Remote uses HTTPS instead of SSH

Suggested Fix:
  gitid repair github
```

This command is critical for trust.

---

# 13. Safety Features

## 13.1 Automatic Backup

Before modifications:

* `.gitconfig`
* `.ssh/config`

must be backed up.

---

## 13.2 Restore Command

```bash
gitid restore
```

---

## 13.3 Dry Run Mode

```bash
gitid switch work --dry-run
```

Shows changes without applying.

---

# 14. SSH Management

GitID should:

* generate SSH keys
* manage SSH config entries
* avoid duplicate host collisions
* support multiple providers

Example generated config:

```ssh
Host github-work
  HostName github.com
  User git
  IdentityFile ~/.ssh/work_ed25519
```

---

# 15. GitHub CLI Integration

GitID integrates with:

```bash
gh auth
```

Features:

* login
* account switching
* auth validation

---

# 16. Auto Switching Strategies

## V1

Folder-based switching.

## V2

Remote-based switching:

* GitHub org
* GitLab host
* regex matching

---

# 17. CLI Design Principles

Commands should feel:

* obvious
* discoverable
* short
* safe

Support aliases:

```bash
gitid ls
gitid list

gitid sw
gitid switch
```

---

# 18. File Locations

## macOS/Linux

```text
~/.config/gitid/
```

Contains:

* profiles.yaml
* backups/
* logs/

---

# 19. Logging

Debug mode:

```bash
gitid --verbose
```

Optional logs:

```text
~/.config/gitid/logs/
```

---

# 20. Failure Handling

Never partially apply changes.

All operations must:

* validate first
* apply atomically
* rollback on failure

---

# 21. Security Considerations

GitID must NEVER:

* upload credentials
* store passwords
* sync data externally

SSH keys remain local.

---

# 22. Future Features

## V2

* Windows support
* GitLab support
* Bitbucket support
* remote URL auto-detection
* repo-specific overrides

## V3

* VSCode extension
* TUI mode
* signed commit management
* Kubernetes context switching
* cloud profile switching

---

# 23. Open Source Strategy

## License

MIT

---

# 24. Distribution Strategy

## Channels

* npm
* Homebrew
* GitHub Releases

---

# 25. Success Metrics

## Early Success

* GitHub stars
* npm installs
* HackerNews traction
* developer feedback

## Product Metrics

* setup completion rate
* successful switches
* doctor command usage
* issue reports

---

# 26. MVP Scope

## MUST HAVE

* profile add
* profile switch
* SSH integration
* GitHub CLI integration
* attach folders
* auto switching
* help command
* doctor command
* backups

## NICE TO HAVE

* remote-based switching
* GPG support
* VSCode integration

---

# 27. Sample User Flow

## Initial Setup

```bash
npm install -g gitid

gitid setup
gitid add work
gitid add personal
```

---

## Daily Usage

```bash
cd ~/office/project
```

Auto-switches to:

```text
work
```

Then:

```bash
git commit -m "fix"
git push
```

Uses:

* correct email
* correct SSH key
* correct GitHub account

without manual switching.

---

# 28. Competitive Advantage

GitID wins through:

* exceptional UX
* simplicity
* automatic switching
* safety tooling
* terminal-first experience

NOT through technical novelty.

---

# 29. Example CLI UX

```bash
gitid current
gitid switch work
gitid attach work ~/office
gitid doctor
gitid backup
gitid restore
```

Simple.
Predictable.
Fast.
