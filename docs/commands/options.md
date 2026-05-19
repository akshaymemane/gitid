# Global Options

Global options work with every command.

## `--plain`

Force plain output without colors or symbols.

```bash
gitid --plain doctor
```

Use this for:

- scripts
- CI logs
- terminals that do not render styled output well
- accessibility workflows that prefer simple text

## `--verbose`

Print extra diagnostics.

```bash
gitid --verbose switch work
```

Alias:

```bash
gitid -v switch work
```

## `NO_COLOR`

GitID also honors the `NO_COLOR` environment variable:

```bash
NO_COLOR=1 gitid doctor
```

This is equivalent to asking for unstyled output.

## `--color`

Control color output explicitly.

```bash
gitid --color=auto doctor
gitid --color=always doctor
gitid --color=never doctor
```

Values:

| Value | Behavior |
| --- | --- |
| `auto` | Use colors only when GitID detects an interactive color-capable terminal. |
| `always` | Force ANSI color output. Useful when terminal detection is too conservative. |
| `never` | Disable color output. |

GitID also honors `FORCE_COLOR=1`:

```bash
FORCE_COLOR=1 gitid doctor
```
