# Commands

GitID commands are intentionally short and predictable.

| Command | Purpose |
| --- | --- |
| `gitid setup` | Initialize local GitID config |
| `gitid add <profile>` | Create an identity profile |
| `gitid list` | List saved profiles |
| `gitid current` | Show the active profile and Git config |
| `gitid switch <profile>` | Apply an identity profile |
| `gitid attach <profile> <folder>` | Map a folder to a profile |
| `gitid auto enable` | Enable folder-based switching |
| `gitid doctor` | Diagnose Git, SSH, and GitHub auth |
| `gitid backup` | Back up Git and SSH config |
| `gitid restore` | Restore latest backups |

Aliases:

| Alias | Command |
| --- | --- |
| `gitid init` | `gitid setup` |
| `gitid ls` | `gitid list` |
| `gitid status` | `gitid current` |
| `gitid sw` | `gitid switch` |
| `gitid rm` | `gitid remove` |

