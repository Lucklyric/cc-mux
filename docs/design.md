# cc-mux — design

_Status: implemented (v0 scaffold). 2026-07-01._

## Purpose

Keep every Claude Code environment (providers, proxies, tokens, flags) as
**named profiles** in a single config file, and launch `claude` with a profile's
environment inlined. No daemon, no persistent state: each launch `exec`s the
target command with the profile's env overlaid on the current shell environment.

## Config

`~/.config/cc-mux/config.json` — honors `$XDG_CONFIG_HOME`, overridable with
`$CC_MUX_CONFIG`. Written mode `0600` (may contain secrets).

```json
{
  "version": 1,
  "default_command": "claude",
  "profiles": {
    "openrouter": {
      "description": "…",
      "env": {
        "ANTHROPIC_BASE_URL": "https://openrouter.ai/api",
        "ANTHROPIC_AUTH_TOKEN": "${OPENROUTER_API_KEY}",
        "ANTHROPIC_API_KEY": "",
        "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
      },
      "command": "claude",
      "args": []
    }
  }
}
```

- **`${VAR}` interpolation** — any `env` value may embed `${VAR}`, expanded from
  the host environment at launch. Secrets live in the shell/keychain, not the
  file. Unresolved refs are a hard error before launch.
- **Empty string** is exported as an explicit blank (`ANTHROPIC_API_KEY=""`).
- **Command precedence**: profile `command` → config `default_command` → `claude`.

## Launch model

`cc-mux <profile> [args]`:

1. Load config, look up the profile.
2. Resolve `${VAR}` refs (fail fast on any unresolved).
3. `BuildEnviron`: overlay resolved env on `os.Environ()` (overlay wins, empty preserved).
4. `BuildArgv`: `command` + profile `args` + pass-through args (`--` separates).
5. `syscall.Exec` — replace the process; only returns on failure.

Any first arg that is not a subcommand is treated as a profile name, so
`cc-mux openrouter` is sugar for `cc-mux run openrouter`.

## Commands

`run` (default), `list`/`ls`, `show` (masked; `--resolved`, `--reveal`),
`doctor [profile]`, `edit`, `set <profile> K=V…` (multi-key, `--create`),
`init` (`--force`), `path`, `update` (`--check`), `version`, plus cobra `help`.

## doctor

Validates: schema version, unresolved `${VAR}`, launch command on `$PATH`,
malformed `ANTHROPIC_BASE_URL`, profile names shadowing subcommands. Non-zero
exit on any error-level finding. All external state is injected → unit-tested.

## Distribution

- **`install.sh`** — `curl … | bash`; detects OS/arch, resolves latest release
  (or `VERSION=`), downloads, verifies `checksums.txt`, installs to
  `${CC_MUX_BIN_DIR:-$HOME/.local/bin}`. Overrides: `CC_MUX_REPO`, `VERSION`.
- **`.github/workflows/ci.yml`** — `go vet` / `go test` / `go build` + `shellcheck`.
- **`.github/workflows/release.yml`** — on tag `v*`, GoReleaser cross-compiles
  darwin/linux × amd64/arm64, publishes archives + checksums, stamps version via ldflags.
- **`cc-mux update`** — in-binary self-update from the same releases; source repo
  overridable via `CC_MUX_UPDATE_REPO`.

## Layout

```
main.go                    entry point → cli.Execute()
internal/config/           types, load/save, ${VAR} resolution
internal/launch/           BuildEnviron / BuildArgv / Exec
internal/doctor/           validation (injected deps)
internal/update/           GitHub-release self-update
internal/cli/              cobra commands + agent-usable help
install.sh · .goreleaser.yaml · .github/workflows/
```

## Non-goals (v0)

No mutation of Claude Code's own settings files, no session multiplexing, no
profile inheritance/merging. Profiles are pure env-var launch environments.
