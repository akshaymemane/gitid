---
layout: home

hero:
  name: GitID
  text: Fast Git identity switching for developers.
  tagline: A terminal-native CLI for moving between work, personal, and open-source Git identities without second-guessing every commit.
  image:
    src: /logo.svg
    alt: GitID
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: Command Reference
      link: /commands/

features:
  - title: Switch With Confidence
    details: Apply the right Git author, SSH identity, and GitHub CLI account from one profile.
  - title: Built For Real Workflows
    details: Map folders to profiles, preview changes with dry runs, and keep script-friendly plain output.
  - title: Local And Safe
    details: Profiles stay on your machine. GitID backs up Git and SSH config before it changes them.
---

## Why GitID Exists

Developers rarely have just one identity anymore. One laptop might hold a company GitHub account, a personal account, a consulting identity, and a few SSH keys for different providers. Git supports that world, but the workflow is scattered across `git config`, SSH config, `gh auth`, shell hooks, and memory.

GitID is inspired by a simple feeling: before you commit or push, you should know which identity is active. No guessing. No awkward commit email cleanup. No hand-editing SSH config at midnight.

## What It Handles

```bash
npm install -g @akshaymemane/git-id

gitid add work
gitid switch work
gitid doctor
```

GitID helps you manage:

- Git commit name and email
- profile-specific SSH keys
- GitHub CLI account switching
- folder-based automatic profile switching
- backups and restore points for local config files

## The Design Principle

GitID is not trying to be an enterprise identity platform. It is a small, fast CLI that makes everyday Git identity work predictable.

The goal is not cleverness. The goal is calm.

