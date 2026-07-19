#!/usr/bin/env bash
#
# Download a pinned asciinema-player release into demo/vendor/ so demo/index.html
# works locally/offline. The vendored assets are gitignored — for the public
# website, host these two files yourself (or bump the pinned version here).

set -euo pipefail

# Pinned asciinema-player version. Bump when you want a newer player.
VERSION="${ASCIINEMA_PLAYER_VERSION:-3.8.0}"

script_dir=$(
	cd "$(dirname "$0")"
	pwd -P
)
vendor_dir="${script_dir}/vendor"
base_url="https://github.com/asciinema/asciinema-player/releases/download/v${VERSION}"

mkdir -p "${vendor_dir}"

for asset in asciinema-player.min.js asciinema-player.css; do
	echo "Downloading ${asset} (v${VERSION}) ..."
	curl -fL --retry 3 -o "${vendor_dir}/${asset}" "${base_url}/${asset}"
done

echo "Done. Vendored asciinema-player v${VERSION} into ${vendor_dir}."
echo "Open demo/index.html in a browser to preview the demo."
