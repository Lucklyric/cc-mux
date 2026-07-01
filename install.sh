#!/usr/bin/env bash
# cc-mux installer: downloads the latest release binary for this OS/arch,
# verifies its checksum, and installs it.
#
#   curl -fsSL https://raw.githubusercontent.com/Lucklyric/cc-mux/main/install.sh | bash
#
# Environment overrides:
#   VERSION          install a specific tag (e.g. v0.1.0) instead of latest
#   CC_MUX_REPO      owner/name to install from   (default: Lucklyric/cc-mux)
#   CC_MUX_BIN_DIR   install directory            (default: $HOME/.local/bin)
set -euo pipefail

REPO="${CC_MUX_REPO:-Lucklyric/cc-mux}"
BIN_DIR="${CC_MUX_BIN_DIR:-$HOME/.local/bin}"
BIN_NAME="cc-mux"

err() { printf 'install: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || err "required tool not found: $1"; }

need curl
need tar
need uname

os="$(uname -s)"
case "$os" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) err "unsupported OS: $os" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

version="${VERSION:-}"
if [ -z "$version" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | cut -d '"' -f4)"
fi
[ -n "$version" ] || err "could not determine latest version of ${REPO}"

ver_no_v="${version#v}"
archive="${BIN_NAME}_${ver_no_v}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${version}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

printf 'install: downloading %s %s (%s/%s)\n' "$BIN_NAME" "$version" "$os" "$arch"
curl -fsSL "${base}/${archive}" -o "${tmp}/${archive}" \
  || err "download failed: ${base}/${archive}"

if curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt" 2>/dev/null; then
  want="$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')"
  if [ -n "$want" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      got="$(sha256sum "${tmp}/${archive}" | awk '{print $1}')"
    else
      got="$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')"
    fi
    [ "$want" = "$got" ] || err "checksum mismatch for ${archive}"
    printf 'install: checksum verified\n'
  fi
fi

tar -xzf "${tmp}/${archive}" -C "$tmp"
mkdir -p "$BIN_DIR"
install -m 0755 "${tmp}/${BIN_NAME}" "${BIN_DIR}/${BIN_NAME}"
printf 'install: installed %s to %s/%s\n' "$version" "$BIN_DIR" "$BIN_NAME"

# SC2016: the literal $PATH in the hint below is intentional (shown to the user).
# shellcheck disable=SC2016
case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *) printf 'install: %s is not on your PATH; add:\n  export PATH="%s:$PATH"\n' "$BIN_DIR" "$BIN_DIR" ;;
esac
