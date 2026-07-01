#!/usr/bin/env bash
# Local end-to-end test: builds cc-mux and drives it through real scenarios
# against an isolated config — including the actual inline-env launch (exec),
# which in-process unit tests cannot cover.
#
#   ./scripts/e2e.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

BIN="$WORK/cc-mux"
export CC_MUX_CONFIG="$WORK/config.json"
pass=0

# check <description> <expected-substring> <actual-output>
check() {
  if printf '%s' "$3" | grep -qF -- "$2"; then
    printf 'ok   %s\n' "$1"
    pass=$((pass + 1))
  else
    printf 'FAIL %s\n  want substring: %s\n  got:\n%s\n' "$1" "$2" "$3" >&2
    exit 1
  fi
}

printf '== building ==\n'
(cd "$ROOT" && go build -o "$BIN" .)

printf '== init / list ==\n'
check "init writes config" "wrote starter config" "$("$BIN" init)"
check "list shows example profile" "openrouter" "$("$BIN" list)"

# Replace with a controlled profile whose command is `env`, so we can prove the
# launch end-to-end without needing claude installed.
cat >"$CC_MUX_CONFIG" <<'JSON'
{
  "version": 1,
  "default_command": "claude",
  "profiles": {
    "smoke": {
      "description": "e2e — execs env to prove inline-env launch",
      "command": "env",
      "env": {
        "ANTHROPIC_BASE_URL": "http://host:8788/",
        "ANTHROPIC_AUTH_TOKEN": "${SMOKE_TOKEN}",
        "ANTHROPIC_API_KEY": "",
        "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
      }
    }
  }
}
JSON

printf '== doctor: unresolved var must fail ==\n'
if "$BIN" doctor >/dev/null 2>&1; then
  printf 'FAIL doctor should exit non-zero when SMOKE_TOKEN is unset\n' >&2
  exit 1
fi
printf 'ok   doctor fails on unresolved SMOKE_TOKEN\n'
pass=$((pass + 1))

printf '== doctor: var set is ok ==\n'
check "doctor ok when var resolves" "ok" "$(SMOKE_TOKEN=x "$BIN" doctor)"

printf '== inline-env launch (real exec of env) ==\n'
launched="$(SMOKE_TOKEN=abc123 "$BIN" smoke)"
check "token resolved from host env" "ANTHROPIC_AUTH_TOKEN=abc123" "$launched"
check "explicit empty api key preserved" "ANTHROPIC_API_KEY=" "$launched"
check "base url inlined verbatim" "ANTHROPIC_BASE_URL=http://host:8788/" "$launched"
check "static flag inlined" "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1" "$launched"

printf '== pass-through args ==\n'
# `env PASSTHROUGH=ok` sets a var and prints the environment, so the arg both
# reaches the exec'd command and shows up in its output.
argout="$(SMOKE_TOKEN=x "$BIN" smoke -- PASSTHROUGH=ok)"
check "pass-through arg reached the command" "PASSTHROUGH=ok" "$argout"
check "profile env still applied alongside args" "ANTHROPIC_BASE_URL=http://host:8788/" "$argout"

printf '== version ==\n'
check "version prints tool name" "cc-mux" "$("$BIN" version)"

printf '\nall %d checks passed\n' "$pass"
