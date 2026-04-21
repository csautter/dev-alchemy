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

if [[ "$RUNNER_VERSION" == "latest" ]]; then
	RUNNER_VERSION="$(
		curl -fsSL "https://api.github.com/repos/actions/runner/releases/latest" | jq -r '.tag_name' | sed 's/^v//'
	)"
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
