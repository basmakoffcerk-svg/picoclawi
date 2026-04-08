#!/usr/bin/env sh

set -eu

OWNER="${PICOCLAW_REPO_OWNER:-basmakoffcerk-svg}"
REPO="${PICOCLAW_REPO_NAME:-picoclawi}"
INSTALL_DIR="${PICOCLAW_INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_FROM_SOURCE="${PICOCLAW_INSTALL_FROM_SOURCE:-0}"
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

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

resolve_sudo() {
  if [ "$(id -u)" -eq 0 ]; then
    echo ""
    return 0
  fi
  if has_cmd sudo; then
    echo "sudo"
    return 0
  fi
  return 1
}

install_build_deps() {
  missing=""
  for cmd in git make go; do
    if ! has_cmd "$cmd"; then
      missing="$missing $cmd"
    fi
  done

  [ -z "$missing" ] && return 0

  echo "Missing build dependencies:$missing"
  echo "Attempting automatic dependency installation..."

  sudo_cmd="$(resolve_sudo || true)"

  if has_cmd apt-get; then
    ${sudo_cmd:+$sudo_cmd }apt-get update -y
    ${sudo_cmd:+$sudo_cmd }apt-get install -y git make golang-go ca-certificates
  elif has_cmd apk; then
    ${sudo_cmd:+$sudo_cmd }apk add --no-cache git make go ca-certificates
  elif has_cmd dnf; then
    ${sudo_cmd:+$sudo_cmd }dnf install -y git make golang ca-certificates
  elif has_cmd yum; then
    ${sudo_cmd:+$sudo_cmd }yum install -y git make golang ca-certificates
  elif has_cmd pacman; then
    ${sudo_cmd:+$sudo_cmd }pacman -Sy --noconfirm git make go ca-certificates
  elif has_cmd zypper; then
    ${sudo_cmd:+$sudo_cmd }zypper --non-interactive install git make go ca-certificates
  elif has_cmd pkg; then
    ${sudo_cmd:+$sudo_cmd }pkg update -y
    ${sudo_cmd:+$sudo_cmd }pkg install -y git make golang
  elif has_cmd brew; then
    brew install git make go
  else
    echo "No supported package manager found for auto-install." >&2
  fi

  for cmd in git make go; do
    need_cmd "$cmd"
  done
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
  tag="$(curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    -H "User-Agent: picoclaw-installer" \
    "$api_url" 2>/dev/null | awk -F '"' '/"tag_name":/ { print $4; exit }' || true)"
  if [ -n "$tag" ]; then
    echo "$tag"
    return 0
  fi

  # Fallback for API rate limits / 403: resolve redirects from /releases/latest.
  latest_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/$OWNER/$REPO/releases/latest" 2>/dev/null || true)"
  case "$latest_url" in
    *"/releases/tag/"*)
      echo "${latest_url##*/}"
      return 0
      ;;
  esac

  return 1
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

install_from_source() {
  install_build_deps
  need_cmd git
  need_cmd make
  need_cmd go

  src_dir="$TMP_DIR/src"
  repo_url="https://github.com/$OWNER/$REPO.git"

  echo "Release artifacts unavailable. Falling back to source build."
  echo "Cloning $repo_url"
  git clone --depth=1 "$repo_url" "$src_dir"
  (
    cd "$src_dir"
    make build
    install -m 0755 "build/picoclaw" "$INSTALL_DIR/picoclaw"
    # Launcher binaries are optional in fallback mode.
    make build-launcher >/dev/null 2>&1 || true
    [ -f "build/picoclaw-launcher" ] && install -m 0755 "build/picoclaw-launcher" "$INSTALL_DIR/picoclaw-launcher" || true
    make build-launcher-tui >/dev/null 2>&1 || true
    [ -f "build/picoclaw-launcher-tui" ] && install -m 0755 "build/picoclaw-launcher-tui" "$INSTALL_DIR/picoclaw-launcher-tui" || true
  )
  echo "Installed $INSTALL_DIR/picoclaw (source build)"
}

main() {
  need_cmd install

  mkdir -p "$INSTALL_DIR"

  if [ "$INSTALL_FROM_SOURCE" = "1" ]; then
    install_from_source
    print_next_steps
    return 0
  fi

  need_cmd curl
  need_cmd tar

  os="$(detect_os)"
  arch="$(detect_arch)"
  tag="$(download_latest_tag || true)"

  if [ -n "$tag" ]; then
    echo "Latest release: $tag"
    if download_asset "$tag" "$os" "$arch"; then
      install_binaries
      print_next_steps
      return 0
    fi
    echo "Release asset for $os/$arch not found under tag $tag." >&2
  else
    echo "Could not determine latest release tag (possibly API rate limit or no release yet)." >&2
  fi

  install_from_source
  print_next_steps
}

main "$@"
