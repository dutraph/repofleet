#!/usr/bin/env bash
# install.sh — build and install fleet into PREFIX/bin.
#
# Usage:
#   ./install.sh                  # installs to /usr/local/bin (sudo if needed)
#   PREFIX=$HOME/.local ./install.sh
#   ./install.sh --prefix /opt/local
#   ./install.sh --ref v1.0.0     # install a specific tag

set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
REPO_URL="${REPO_URL:-https://github.com/dutraph/repofleet.git}"
REF="${REF:-main}"

while [ $# -gt 0 ]; do
    case "$1" in
        --prefix) PREFIX="$2"; shift 2 ;;
        --prefix=*) PREFIX="${1#*=}"; shift ;;
        --repo) REPO_URL="$2"; shift 2 ;;
        --ref) REF="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,12p' "$0"
            exit 0
            ;;
        *)
            echo "unknown option: $1" >&2
            exit 2
            ;;
    esac
done

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m!!\033[0m %s\n' "$*" >&2; exit 1; }

command -v go >/dev/null 2>&1 || die "go (>= 1.22) not found on PATH. Install Go first."
command -v git >/dev/null 2>&1 || die "git not found on PATH."

INSTALL_DIR="$PREFIX/bin"
SUDO=""
mkdir -p "$INSTALL_DIR" 2>/dev/null || SUDO="sudo"
if [ -n "$SUDO" ]; then
    log "creating $INSTALL_DIR (requires sudo)"
    sudo mkdir -p "$INSTALL_DIR"
fi
if [ ! -w "$INSTALL_DIR" ]; then SUDO="sudo"; fi

src_dir=""
if [ -f "./go.mod" ] && grep -q "^module github.com/dutraph/repofleet" ./go.mod 2>/dev/null; then
    log "building from current directory"
    src_dir="$(pwd)"
else
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT
    log "cloning $REPO_URL ($REF) into $tmp"
    git clone --depth 1 --branch "$REF" "$REPO_URL" "$tmp/repofleet"
    src_dir="$tmp/repofleet"
fi

version=""
if ( cd "$src_dir" && git rev-parse --is-inside-work-tree >/dev/null 2>&1 ); then
    version="$(cd "$src_dir" && git describe --tags --always --dirty 2>/dev/null || true)"
fi
if [ -z "$version" ] && [ -f "$src_dir/VERSION" ]; then
    version="$(tr -d '[:space:]' < "$src_dir/VERSION")"
fi
[ -n "$version" ] || version="dev"

log "compiling fleet $version"
mkdir -p "$src_dir/bin"
( cd "$src_dir" && go build \
    -trimpath \
    -ldflags "-s -w -X github.com/dutraph/repofleet/internal/version.Version=${version#v}" \
    -o ./bin/fleet ./cmd/fleet )

log "installing to $INSTALL_DIR/fleet"
$SUDO install -m 0755 "$src_dir/bin/repos" "$INSTALL_DIR/fleet"

log "done."
