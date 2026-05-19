# `gitid switch`

Apply a saved identity profile.

```bash
gitid switch work
```

Alias:

```bash
gitid sw work
```

## What It Does

- applies Git `user.name`
- applies Git `user.email`
- configures SSH command behavior when an SSH key is set
- attempts GitHub CLI account switching when a GitHub username is set
- marks the profile as active in GitID config

## Dry Run

Preview a switch without applying it:

```bash
gitid switch work --dry-run
```

Dry runs are useful before changing identity inside an important repository.

## Git Scope

When run inside a Git repository, GitID applies identity locally to that repository. Outside a repository, it applies global Git identity.

