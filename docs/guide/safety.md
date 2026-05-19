# Safety Model

GitID changes local developer configuration, so its safety model is intentionally simple.

## Local Only

GitID stores profiles in:

```text
~/.config/gitid/profiles.yaml
```

It does not upload credentials, passwords, SSH keys, or profile data.

## Backups

Before modifying Git or SSH config, GitID creates backups under:

```text
~/.config/gitid/backups
```

You can also create backups manually:

```bash
gitid backup
```

## Dry Runs

Use dry-run mode before switching:

```bash
gitid switch work --dry-run
```

This is the safest way to see the exact profile actions before GitID applies them.

## Plain Output

For scripts, CI, or terminals where styled output is unwanted:

```bash
gitid --plain doctor
gitid --color=never list
NO_COLOR=1 gitid list
```

If your terminal supports color but GitID shows plain output, force color:

```bash
gitid --color=always doctor
FORCE_COLOR=1 gitid list
```
