# `gitid backup` And `gitid restore`

GitID backs up important local config before modifying it.

## Create Backups

```bash
gitid backup
```

This backs up:

- `~/.gitconfig`
- `~/.ssh/config`

Backups are stored in:

```text
~/.config/gitid/backups
```

## Restore Backups

```bash
gitid restore
```

This restores the latest backup for each supported config file.

Use restore if you want to undo GitID-managed changes to Git or SSH config.

