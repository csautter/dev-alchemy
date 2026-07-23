#!/usr/bin/env bash
#
# Render demo/quickstart.cast -> demo/quickstart.gif using agg (the asciinema GIF
# generator). The GIF is what you embed in the GitHub README, where the JS
# asciinema-player cannot run.
#
# Requirements:
#   - a monospace font and a colour-emoji font, e.g. on Debian/Ubuntu:
#       sudo apt-get install -y fonts-dejavu-core fonts-noto-color-emoji fontconfig
#   - agg on PATH; if missing, a pinned static binary is downloaded into
#     demo/vendor/ (gitignored).

set -euo pipefail

AGG_VERSION="${AGG_VERSION:-1.9.0}"

script_dir=$(
	cd "$(dirname "$0")"
	pwd -P
)
cast="${script_dir}/quickstart.cast"
gif="${script_dir}/quickstart.gif"

if [[ ! -f "${cast}" ]]; then
	echo "error: ${cast} not found; run 'make demo' first." >&2
	exit 1
fi

# Resolve an agg binary: prefer PATH, else download a pinned release.
if command -v agg >/dev/null 2>&1; then
	AGG="$(command -v agg)"
else
	vendor_dir="${script_dir}/vendor"
	AGG="${vendor_dir}/agg"
	if [[ ! -x "${AGG}" ]]; then
		os="$(uname -s)"
		machine="$(uname -m)"
		case "${os}-${machine}" in
		Linux-x86_64) target="x86_64-unknown-linux-musl" ;;
		Linux-aarch64 | Linux-arm64) target="aarch64-unknown-linux-gnu" ;;
		Darwin-x86_64) target="x86_64-apple-darwin" ;;
		Darwin-arm64) target="aarch64-apple-darwin" ;;
		*)
			echo "error: no prebuilt agg for ${os}-${machine}; install agg manually (cargo install agg)." >&2
			exit 1
			;;
		esac
		mkdir -p "${vendor_dir}"
		url="https://github.com/asciinema/agg/releases/download/v${AGG_VERSION}/agg-${target}"
		echo "Downloading agg v${AGG_VERSION} (${target}) ..."
		curl -fL --retry 3 -o "${AGG}" "${url}"
		chmod +x "${AGG}"
	fi
fi

# mktemp creates the extensionless file; keep that name too so the trap cleans
# up both it and the suffixed path agg actually writes.
tmp_gif_base="$(mktemp "${TMPDIR:-/tmp}/quickstart-gif.XXXXXX")"
tmp_gif="${tmp_gif_base}.gif"
trap 'rm -f "${tmp_gif_base}" "${tmp_gif}"' EXIT

echo "Rendering ${cast} -> ${gif} ..."
"${AGG}" \
	--font-family "DejaVu Sans Mono" \
	--theme asciinema \
	--idle-time-limit 2 \
	--last-frame-duration 3 \
	"${cast}" "${tmp_gif}"

# Optimise with gifsicle when available. agg emits per-frame local colormaps;
# -O3 with a single 256-colour global palette roughly halves the file with no
# visible quality loss (the only differences are emoji anti-aliasing edges).
if command -v gifsicle >/dev/null 2>&1; then
	raw_size=$(wc -c <"${tmp_gif}")
	gifsicle -O3 --colors 256 "${tmp_gif}" -o "${gif}"
	opt_size=$(wc -c <"${gif}")
	echo "Optimised with gifsicle: ${raw_size} -> ${opt_size} bytes."
else
	echo "gifsicle not found; writing unoptimised GIF (install gifsicle to shrink it)."
	cp -f "${tmp_gif}" "${gif}"
fi

echo "Wrote ${gif}."
