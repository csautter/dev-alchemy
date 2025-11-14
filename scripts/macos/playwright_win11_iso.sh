#!/bin/bash

set -e

# Manual argument parsing for portability
arch="arm64"

script_dir=$(
	# shellcheck disable=SC2164
	cd "$(dirname "$0")"
	pwd -P
)
project_root=$(
	# shellcheck disable=SC2164
	cd "${script_dir}/../.."
	pwd -P
)

while [[ $# -gt 0 ]]; do
	case "$1" in
	--arch)
		if [[ -n "$2" && ("$2" == "amd64" || "$2" == "arm64") ]]; then
			arch="$2"
			shift 2
		else
			echo "Invalid value for --arch: $2. Allowed values are 'amd64' or 'arm64'." >&2
			exit 1
		fi
		;;
	--project-root)
		if [[ -n "$2" ]]; then
			project_root="$2"
			shift 2
		else
			echo "Invalid value for --project-root: $2." >&2
			exit 1
		fi
		;;
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

# This script loads the python venv and runs the playwright script to download the Windows 11 ISO.
cd "${project_root}/scripts/macos/" || exit 1
if [ ! -d .venv ]; then
	python3 -m venv .venv
fi
# shellcheck disable=SC1091
source .venv/bin/activate

if ! python -c "import playwright" &>/dev/null; then
	pip install playwright
	python -m playwright install chromium
fi

if [ "$arch" = "amd64" ]; then
	python playwright_win11_iso.py
elif [ "$arch" = "arm64" ]; then
	python playwright_win11_iso.py --arm
fi
