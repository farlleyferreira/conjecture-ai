#!/bin/sh
# Installs the prebuilt conjecture binary for this machine (Linux or
# macOS) from this project's GitHub Releases, after checking its
# SHA-256 against the one published next to it.
#
#   curl -fsSL https://raw.githubusercontent.com/farlleyferreira/conjecture-ai/main/install.sh | sh
#
# Environment:
#   CONJECTURE_VERSION      a release tag (e.g. v1.3.0); default: latest
#   CONJECTURE_INSTALL_DIR  where the binary goes; default: ~/.local/bin
set -eu

repo="farlleyferreira/conjecture-ai"
version="${CONJECTURE_VERSION:-latest}"
install_dir="${CONJECTURE_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
	echo "conjecture install: $1" >&2
	exit 1
}

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "unsupported OS $(uname -s) - on Windows use install.ps1" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) fail "unsupported architecture $(uname -m)" ;;
esac

asset="conjecture_${os}_${arch}.tar.gz"
if [ "$version" = latest ]; then
	base="https://github.com/${repo}/releases/latest/download"
else
	base="https://github.com/${repo}/releases/download/${version}"
fi

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${asset} (${version})..."
curl -fsSL -o "${tmp}/${asset}" "${base}/${asset}" ||
	fail "could not download ${base}/${asset} - does that release have binaries?"
curl -fsSL -o "${tmp}/${asset}.sha256" "${base}/${asset}.sha256" ||
	fail "could not download the checksum for ${asset}"

want="$(cut -d ' ' -f 1 <"${tmp}/${asset}.sha256")"
if command -v sha256sum >/dev/null 2>&1; then
	got="$(sha256sum "${tmp}/${asset}" | cut -d ' ' -f 1)"
elif command -v shasum >/dev/null 2>&1; then
	got="$(shasum -a 256 "${tmp}/${asset}" | cut -d ' ' -f 1)"
else
	fail "sha256sum or shasum is required to verify the download"
fi
[ "$want" = "$got" ] || fail "checksum mismatch for ${asset}: want ${want}, got ${got}"

tar -xzf "${tmp}/${asset}" -C "$tmp" ./conjecture
mkdir -p "$install_dir"
mv "${tmp}/conjecture" "${install_dir}/conjecture"
chmod +x "${install_dir}/conjecture"

echo "Installed: ${install_dir}/conjecture ($("${install_dir}/conjecture" version))"
case ":${PATH}:" in
*":${install_dir}:"*) ;;
*) echo "Note: ${install_dir} is not on your PATH - add it to run 'conjecture' directly." ;;
esac
