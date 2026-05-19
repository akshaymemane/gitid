# GitID

Fast Git identity switching for developers using multiple accounts.

[![npm version](https://img.shields.io/npm/v/@akshaymemane/git-id.svg)](https://www.npmjs.com/package/@akshaymemane/git-id)
[![GitHub](https://img.shields.io/badge/github-akshaymemane%2Fgitid-black)](https://github.com/akshaymemane/gitid)

## Install

```bash
npm install -g @akshaymemane/git-id
```

The npm package installs the `gitid` command. macOS and Linux binaries are bundled for arm64 and x64.

## Demo

![GitID terminal demo](docs/public/demo.gif)

## What's New in 0.2

GitID now uses Charm libraries for a cleaner terminal experience:

- styled success, warning, and failure states
- a readable profiles table
- a grouped `gitid doctor` report
- a guided interactive form for `gitid add`
- `--plain` and `NO_COLOR=1` support for scripts and simple terminals

## Quick Start

```bash
gitid setup
gitid add work --user "Work Name" --email work@example.com --github work-github --ssh-key ~/.ssh/work_ed25519
gitid add personal --user "Personal Name" --email me@example.com --github personal-github --ssh-key ~/.ssh/personal_ed25519
gitid switch work
gitid current
```

Interactive setup is also supported:

```bash
gitid add work
```

Generate a new SSH key while adding a profile:

```bash
gitid add work --user "Work Name" --email work@example.com --generate-ssh-key
```

## Commands

```bash
gitid add <profile>        # create a profile
gitid list                 # list profiles
gitid switch <profile>     # apply Git, SSH, and GitHub CLI identity
gitid current              # show active profile and Git config
gitid remove <profile>     # remove a profile
gitid attach <profile> DIR # map a folder to a profile
gitid auto enable          # enable auto-switching config and print shell hook
gitid auto enable --install # append shell hook to ~/.zshrc or ~/.bashrc
gitid doctor               # diagnose local setup
gitid backup               # back up ~/.gitconfig and ~/.ssh/config
gitid restore              # restore latest backups
```

Plain output for automation:

```bash
gitid --plain doctor
gitid --color=never list
NO_COLOR=1 gitid list
```

Force color when your terminal supports it but detection is too conservative:

```bash
gitid --color=always doctor
FORCE_COLOR=1 gitid list
```

## Documentation

The full VitePress documentation lives in `docs/`.

```bash
npm run docs:dev
npm run docs:build
```

Useful aliases:

```bash
gitid ls
gitid sw work
gitid status
```

## Auto Switching

Attach a folder and install the shell hook:

```bash
gitid attach work ~/office
gitid auto enable --install
```

Restart your shell. When you `cd` into `~/office` or any child folder, GitID applies the attached profile.

## Safety

GitID keeps all data local in `~/.config/gitid/profiles.yaml`. It does not store passwords or upload credentials.

Before modifying Git or SSH config, GitID creates backups under `~/.config/gitid/backups`.

Use dry-run mode to preview a switch:

```bash
gitid switch work --dry-run
```

## Notes

The product and binary are named `gitid`. The npm package is scoped as `@akshaymemane/git-id` because unscoped names close to `gitid` are blocked by the public npm registry similarity guard.
