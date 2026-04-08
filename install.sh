#!/usr/bin/env sh

set -eu

OWNER="${PICOCLAW_REPO_OWNER:-basmakoffcerk-svg}"
REPO="${PICOCLAW_REPO_NAME:-picoclawi}"
INSTALL_DIR="${PICOCLAW_INSTALL_DIR:-$HOME/.local/bin}"
TMP_DIR="${TMPDIR:-/tmp}/picoclaw-install.$$"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

detect_os() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux) echo "Linux" ;;
    darwin) echo "Darwin" ;;
    freebsd) echo "Freebsd" ;;
    netbsd) echo "Netbsd" ;;
    *)
      echo "Unsupported OS: $os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "x86_64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l|armv7) echo "armv7" ;;
    armv6l|armv6) echo "armv6" ;;
    riscv64) echo "riscv64" ;;
    loongarch64|loong64) echo "loong64" ;;
    s390x) echo "s390x" ;;
    mipsel|mipsle) echo "mipsle" ;;
    *)
      echo "Unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

download_latest_tag() {
  api_url="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
  curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    -H "User-Agent: picoclaw-installer" \
    "$api_url" | awk -F '"' '/"tag_name":/ { print $4; exit }'
}

download_asset() {
  tag="$1"
  os="$2"
  arch="$3"
  archive="picoclaw_${os}_${arch}.tar.gz"
  url="https://github.com/$OWNER/$REPO/releases/download/$tag/$archive"
  mkdir -p "$TMP_DIR"
  echo "Downloading $archive from $url"
  curl -fL "$url" -o "$TMP_DIR/$archive"
  tar -xzf "$TMP_DIR/$archive" -C "$TMP_DIR"
}

install_binaries() {
  mkdir -p "$INSTALL_DIR"

  for bin in picoclaw picoclaw-launcher picoclaw-launcher-tui; do
    if [ -f "$TMP_DIR/$bin" ]; then
      install -m 0755 "$TMP_DIR/$bin" "$INSTALL_DIR/$bin"
      echo "Installed $INSTALL_DIR/$bin"
    fi
  done
}

print_next_steps() {
  echo
  echo "Installation complete."
  echo "Add to PATH if needed:"
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
  echo
  echo "Quick start:"
  echo "  picoclaw onboard"
  echo "  picoclaw login"
  echo "  picoclaw agent -m \"hello\""
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd install

  os="$(detect_os)"
  arch="$(detect_arch)"
  tag="$(download_latest_tag)"

  if [ -z "$tag" ]; then
    echo "Failed to determine latest release tag." >&2
    echo "Make sure this repository has at least one published GitHub Release." >&2
    exit 1
  fi

  echo "Latest release: $tag"
  download_asset "$tag" "$os" "$arch"
  install_binaries
  print_next_steps
}

main "$@"
