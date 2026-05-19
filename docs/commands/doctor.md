# `gitid doctor`

Diagnose local GitID, Git, SSH, and GitHub CLI state.

```bash
gitid doctor
```

## Checks

`doctor` reports on:

- GitID config readability
- Git installation
- current Git identity
- active GitID profile
- SSH key presence
- SSH agent state
- GitHub CLI auth
- repository remote URL
- auto-switching state

## Suggested Fixes

When GitID sees common issues, it prints suggested next commands. For example:

```bash
gitid switch work
gitid auto enable
```

Use `doctor` after setup, after changing SSH keys, or any time Git pushes do not use the expected identity.

