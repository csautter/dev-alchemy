#!/usr/bin/env bash

set -euo pipefail

if [[ "$(uname -s)" != "Linux" ]]; then
	echo "This installer only supports Linux hosts." >&2
	exit 1
fi

if [[ $# -ne 1 ]]; then
	echo "Usage: $0 <project-root>" >&2
	exit 1
fi

project_root="$1"
go_mod_path="${project_root}/go.mod"

if [[ ! -f "${go_mod_path}" ]]; then
	echo "go.mod not found at ${go_mod_path}" >&2
	exit 1
fi

if [[ "${EUID}" -eq 0 ]]; then
	SUDO=""
else
	if ! command -v sudo >/dev/null 2>&1; then
		echo "sudo is required to install Go." >&2
		exit 1
	fi
	SUDO="sudo"
fi

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "Required command not found: $1" >&2
		exit 1
	fi
}

require_command curl
require_command python3
require_command tar
require_command sha256sum

go_version="$(awk '/^go [0-9]+\.[0-9]+(\.[0-9]+)?$/ { print $2; exit }' "${go_mod_path}")"
if [[ -z "${go_version}" ]]; then
	echo "Could not determine required Go version from ${go_mod_path}" >&2
	exit 1
fi

case "$(uname -m)" in
x86_64 | amd64)
	go_arch="amd64"
	;;
aarch64 | arm64)
	go_arch="arm64"
	;;
*)
	echo "Unsupported Linux architecture: $(uname -m)" >&2
	exit 1
	;;
esac

go_filename="go${go_version}.linux-${go_arch}.tar.gz"
go_archive_url="https://go.dev/dl/${go_filename}"

if command -v go >/dev/null 2>&1; then
	current_go_version="$(go version | awk '{print $3}')"
	if [[ "${current_go_version}" == "go${go_version}" ]]; then
		echo "Go ${go_version} is already installed, skipping."
		exit 0
	fi
fi

go_archive_sha256="$(
	python3 - "${go_version}" "${go_arch}" <<'PY'
import json
import sys
import urllib.request

version = "go" + sys.argv[1]
arch = sys.argv[2]

with urllib.request.urlopen("https://go.dev/dl/?mode=json&include=all") as response:
    releases = json.load(response)

for release in releases:
    if release["version"] != version:
        continue
    for file in release["files"]:
        if file["os"] == "linux" and file["arch"] == arch and file["kind"] == "archive":
            print(file["sha256"])
            raise SystemExit(0)

raise SystemExit(f"could not resolve checksum for {version} linux/{arch}")
PY
)"

if [[ -z "${go_archive_sha256}" ]]; then
	echo "Could not resolve checksum for ${go_filename}" >&2
	exit 1
fi

tmp_dir="$(mktemp -d)"
cleanup() {
	rm -rf "${tmp_dir}"
}
trap cleanup EXIT

archive_path="${tmp_dir}/${go_filename}"

echo "Downloading ${go_filename} from go.dev"
curl -fL "${go_archive_url}" -o "${archive_path}"

downloaded_sha256="$(sha256sum "${archive_path}" | awk '{print $1}')"
if [[ "${downloaded_sha256}" != "${go_archive_sha256}" ]]; then
	echo "Checksum mismatch for ${go_filename}" >&2
	echo "Expected: ${go_archive_sha256}" >&2
	echo "Actual:   ${downloaded_sha256}" >&2
	exit 1
fi

echo "Installing Go ${go_version} to /usr/local/go"
${SUDO} rm -rf /usr/local/go
${SUDO} tar -C /usr/local -xzf "${archive_path}"
${SUDO} ln -sf /usr/local/go/bin/go /usr/local/bin/go
${SUDO} ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

go version
