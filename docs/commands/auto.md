# `gitid attach` And `gitid auto`

Folder-based switching maps directories to profiles.

## Attach A Folder

```bash
gitid attach work ~/office
```

When the shell hook runs inside `~/office` or a child folder, GitID switches to the `work` profile.

## Enable Auto Switching

```bash
gitid auto enable
```

This enables auto-switching in GitID config and prints the shell hook.

Install the hook automatically for supported shells:

```bash
gitid auto enable --install
```

Currently supported shell profile targets:

- zsh: `~/.zshrc`
- bash: `~/.bashrc`

## Other Auto Commands

```bash
gitid auto disable
gitid auto hook
```

`gitid auto apply` is an internal command used by the shell hook.

