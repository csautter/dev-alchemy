#!/usr/bin/env bash

set -euo pipefail

RUNNER_VERSION="latest"
RUNNER_DIR="/actions-runner"

fail() {
	echo "FAILURE: $*" >&2
	exit 1
}

usage() {
	echo "Usage: $0 [-v runner-version] [-d runner-dir]" >&2
	exit 1
}

while getopts ":v:d:" opt; do
	case "$opt" in
		v)
			RUNNER_VERSION="$OPTARG"
			;;
		d)
			RUNNER_DIR="$OPTARG"
			;;
		*)
			usage
			;;
	esac
done

for cmd in curl gzip jq sed tar; do
	command -v "$cmd" >/dev/null 2>&1 || fail "Required command '$cmd' was not found."
done

install_azure_cli() {
	if command -v az >/dev/null 2>&1; then
		echo "Azure CLI already installed: $(az version --query '\"azure-cli\"' -o tsv)"
		return 0
	fi

	echo "Installing Azure CLI from Microsoft apt repository"
	export DEBIAN_FRONTEND=noninteractive

	apt-get update
	apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release

	install -d -m 0755 /etc/apt/keyrings
	curl -sLS https://packages.microsoft.com/keys/microsoft.asc |
		gpg --dearmor |
		tee /etc/apt/keyrings/microsoft.gpg >/dev/null
	chmod go+r /etc/apt/keyrings/microsoft.gpg

	local az_dist
	az_dist="$(lsb_release -cs)"
	cat >/etc/apt/sources.list.d/azure-cli.sources <<EOF
Types: deb
URIs: https://packages.microsoft.com/repos/azure-cli/
Suites: ${az_dist}
Components: main
Architectures: $(dpkg --print-architecture)
Signed-by: /etc/apt/keyrings/microsoft.gpg
EOF

	apt-get update
	apt-get install -y azure-cli

	command -v az >/dev/null 2>&1 || fail "Azure CLI installation completed, but 'az' is still not available on PATH."
	echo "Azure CLI installed: $(az version --query '\"azure-cli\"' -o tsv)"
}

configure_keyboard_layout() {
	if [[ -f /etc/default/keyboard ]] && grep -q '^XKBLAYOUT="de"$' /etc/default/keyboard; then
		echo "Keyboard layout already configured: de"
	else
		echo "Configuring keyboard layout: de"
		cat >/etc/default/keyboard <<'EOF'
XKBMODEL="pc105"
XKBLAYOUT="de"
XKBVARIANT=""
XKBOPTIONS=""
BACKSPACE="guess"
EOF
	fi

	if command -v localectl >/dev/null 2>&1; then
		localectl set-x11-keymap de pc105
	fi

	if command -v setupcon >/dev/null 2>&1; then
		setupcon --force
	fi
}

resolve_latest_runner_version() {
	local version=""
	local latest_url=""

	version="$(
		curl -fsSL --retry 3 --connect-timeout 15 \
			-H "Accept: application/vnd.github+json" \
			"https://api.github.com/repos/actions/runner/releases/latest" |
			jq -r '.tag_name // empty' |
			sed 's/^v//'
	)" || true

	if [[ "$version" =~ ^[0-9.]+$ ]]; then
		printf '%s\n' "$version"
		return 0
	fi

	latest_url="$(
		curl -fsSIL --retry 3 --connect-timeout 15 \
			-o /dev/null \
			-w '%{url_effective}' \
			"https://github.com/actions/runner/releases/latest"
	)" || true
	version="${latest_url##*/}"
	version="${version#v}"

	if [[ "$version" =~ ^[0-9.]+$ ]]; then
		printf '%s\n' "$version"
		return 0
	fi

	fail "Failed to resolve latest runner version from GitHub."
}

case "$(uname -m)" in
	x86_64|amd64)
		RUNNER_ARCH="x64"
		;;
	aarch64|arm64)
		RUNNER_ARCH="arm64"
		;;
	*)
		fail "Unsupported architecture '$(uname -m)'."
		;;
esac

configure_keyboard_layout
install_azure_cli

if [[ "$RUNNER_VERSION" == "latest" ]]; then
	RUNNER_VERSION="$(resolve_latest_runner_version)"
fi

[[ "$RUNNER_VERSION" =~ ^[0-9.]+$ ]] || fail "Invalid runner version '${RUNNER_VERSION}'."

mkdir -p "$RUNNER_DIR"
cd "$RUNNER_DIR"

archive_name="actions-runner-linux-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz"
curl -fsSLo "$archive_name" -L "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${archive_name}"
tar xzf "$archive_name"
rm -f "$archive_name"

# Ubuntu 24.04 uses libicu74 while some runner dependency scripts still only probe for libicu72.
if [[ -f "./bin/installdependencies.sh" ]]; then
	sed -i 's/libicu72/libicu72 libicu74/' ./bin/installdependencies.sh
	./bin/installdependencies.sh
fi
