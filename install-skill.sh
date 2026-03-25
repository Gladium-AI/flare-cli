#!/bin/sh
set -eu

REPO="${REPO:-Gladium-AI/flare-cli}"
REF="${REF:-main}"
SKILL_NAME="${SKILL_NAME:-flare-cli}"
SKILL_PATH="${SKILL_PATH:-skills/${SKILL_NAME}}"
INSTALL_TARGETS="${INSTALL_TARGETS:-claude,codex}"
CLAUDE_SKILLS_DIR="${CLAUDE_SKILLS_DIR:-$HOME/.claude/skills}"
CODEX_HOME_DEFAULT="${HOME}/.codex"
CODEX_HOME_DIR="${CODEX_HOME:-$CODEX_HOME_DEFAULT}"
CODEX_SKILLS_DIR="${CODEX_SKILLS_DIR:-${CODEX_HOME_DIR%/}/skills}"

download() {
	url="$1"
	out="$2"

	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$out"
		return
	fi

	if command -v wget >/dev/null 2>&1; then
		wget -q "$url" -O "$out"
		return
	fi

	echo "Error: curl or wget is required" >&2
	exit 1
}

copy_skill() {
	src="$1"
	root="$2"
	dst="${root%/}/${SKILL_NAME}"

	mkdir -p "$root"
	rm -rf "$dst"
	cp -R "$src" "$dst"
	printf 'Installed skill to %s\n' "$dst"
}

install_skill() {
	src="$1"

	if [ -n "${SKILLS_DIR:-}" ]; then
		copy_skill "$src" "$SKILLS_DIR"
		return
	fi

	installed=0
	for target in $(printf '%s' "$INSTALL_TARGETS" | tr ',' ' '); do
		case "$target" in
			claude)
				copy_skill "$src" "$CLAUDE_SKILLS_DIR"
				installed=1
				;;
			codex)
				copy_skill "$src" "$CODEX_SKILLS_DIR"
				installed=1
				;;
			"")
				;;
			*)
				echo "Error: unsupported install target '$target' (use claude,codex)" >&2
				exit 1
				;;
		esac
	done

	if [ "$installed" -ne 1 ]; then
		echo "Error: no install targets resolved" >&2
		exit 1
	fi
}

if [ -n "${SOURCE_DIR:-}" ]; then
	src="${SOURCE_DIR%/}/${SKILL_PATH}"
	if [ ! -d "$src" ]; then
		echo "Error: skill directory not found at $src" >&2
		exit 1
	fi
	install_skill "$src"
	exit 0
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

archive="$tmpdir/repo.tar.gz"
url="https://codeload.github.com/${REPO}/tar.gz/refs/heads/${REF}"

download "$url" "$archive"
tar -xzf "$archive" -C "$tmpdir"

root_dir="$(find "$tmpdir" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if [ -z "$root_dir" ]; then
	echo "Error: could not extract repository archive" >&2
	exit 1
fi

src="${root_dir}/${SKILL_PATH}"
if [ ! -d "$src" ]; then
	echo "Error: skill directory not found in downloaded archive: $src" >&2
	exit 1
fi

install_skill "$src"
