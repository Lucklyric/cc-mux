# cc-mux

Profile-based launcher & config manager for [Claude Code](https://claude.com/claude-code).

`cc-mux` keeps every Claude Code environment you use — different providers, proxies,
tokens, and flags — as **named profiles** in one config file, and launches `claude`
with a profile's environment inlined. No daemon, no persistent state: each launch
simply `exec`s `claude` with the profile's env overlaid on your shell env.

```bash
cc-mux openrouter
# ⇒ ANTHROPIC_BASE_URL="http://…" ANTHROPIC_AUTH_TOKEN="…" \
#   ANTHROPIC_API_KEY="" CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1 claude
```

> **Status: scaffolding.** The design is settled; the CLI implementation is in
> progress. Commands and flags below describe the target behavior.

## Config

`~/.config/cc-mux/config.json` (honors `$XDG_CONFIG_HOME`; override with `$CC_MUX_CONFIG`).

```json
{
  "version": 1,
  "default_command": "claude",
  "profiles": {
    "openrouter": {
      "description": "OpenRouter (Anthropic-compatible endpoint)",
      "env": {
        "ANTHROPIC_BASE_URL": "https://openrouter.ai/api",
        "ANTHROPIC_AUTH_TOKEN": "${OPENROUTER_API_KEY}",
        "ANTHROPIC_API_KEY": "",
        "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
      }
    }
  }
}
```

- `env` values support `${VAR}` interpolation from the host environment, so secrets
  live in your shell/keychain rather than in the file.
- An empty string (`""`) is exported as an explicit blank.
- Unresolved `${VAR}` references are a hard error before launch.

## Commands

| Command | Purpose |
|---|---|
| `cc-mux <profile> [-- args]` | Launch `claude` with that profile (trailing args pass through) |
| `cc-mux list` | List profiles and descriptions |
| `cc-mux show <profile>` | Show a profile's resolved env (secrets masked) |
| `cc-mux doctor [profile]` | Validate config and profiles |
| `cc-mux edit` | Open the config in `$EDITOR` |
| `cc-mux set <profile> K=V [K2=V2 …]` | Add/update one or more env entries |
| `cc-mux init` | Write a starter config |
| `cc-mux path` | Print the config file path |
| `cc-mux update` | Self-update from GitHub releases |
| `cc-mux help [command]` | Full, example-driven help |

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Lucklyric/cc-mux/main/install.sh | bash
```

Downloads the latest release binary for your OS/arch, verifies its checksum, and
installs to `${CC_MUX_BIN_DIR:-$HOME/.local/bin}`. *(Available once the first release is cut.)*

## License

[MIT](./LICENSE)
