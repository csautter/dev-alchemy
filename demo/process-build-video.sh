#!/usr/bin/env bash
#
# Turn a raw VM-build screen recording (the 1 fps VNC capture produced during
# `sailwright build`) into short, web-ready assets:
#   - demo/vm-build.mp4  (for the website <video>)
#   - demo/vm-build.gif  (for the GitHub README, which can't inline mp4)
#
# The raw recording is 1 fps (a slideshow of the multi-minute autoinstall), so we
# speed it up, scale it down, and re-encode into a smooth ~20-30s time-lapse.
#
# Source the raw recording by running `make quickstart` on a Linux+KVM host; it
# saves artifact/build-<slug>.vnc.mp4. (The KVM CI job also uploads *.vnc.mp4.)
#
# Requirements: ffmpeg; gifsicle to shrink the GIF (skipped with a warning if absent).

set -euo pipefail

# --- Defaults ---------------------------------------------------------------
SLUG="ubuntu-server-amd64"
INPUT=""
SPEED="12"        # playback speed-up factor (1 fps source -> SPEED fps of content)
FPS="24"          # mp4 output frame rate
WIDTH="640"       # mp4 output width (height auto, keeps aspect)
GIF_FPS="10"      # gif output frame rate (lower keeps the file small)
GIF_WIDTH="520"   # gif output width
START=""          # optional trim start (seconds or HH:MM:SS)
DURATION=""       # optional trim duration (seconds), applied to the SOURCE

usage() {
	cat <<'EOF'
Usage: demo/process-build-video.sh [options]

Turns a raw VM-build VNC recording into demo/vm-build.mp4 and demo/vm-build.gif.

Options:
  --input PATH     Source recording (default: artifact/build-<slug>.vnc.mp4)
  --slug SLUG      Target slug for the default input path (default: ubuntu-server-amd64)
  --speed N        Speed-up factor (default: 12)
  --fps N          mp4 frame rate (default: 24)
  --width N        mp4 width in px (default: 640)
  --gif-fps N      gif frame rate (default: 10)
  --gif-width N    gif width in px (default: 520)
  --start T        Trim: start offset on the source (seconds or HH:MM:SS)
  --duration N     Trim: seconds of source to use (before speed-up)
  -h, --help       Show this help
EOF
}

# --- Args -------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
	case "$1" in
	--input) INPUT="$2"; shift 2 ;;
	--slug) SLUG="$2"; shift 2 ;;
	--speed) SPEED="$2"; shift 2 ;;
	--fps) FPS="$2"; shift 2 ;;
	--width) WIDTH="$2"; shift 2 ;;
	--gif-fps) GIF_FPS="$2"; shift 2 ;;
	--gif-width) GIF_WIDTH="$2"; shift 2 ;;
	--start) START="$2"; shift 2 ;;
	--duration) DURATION="$2"; shift 2 ;;
	-h | --help) usage; exit 0 ;;
	*) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
	esac
done

script_dir=$(
	cd "$(dirname "$0")"
	pwd -P
)
project_root=$(
	cd "${script_dir}/.."
	pwd -P
)

if [[ -z "${INPUT}" ]]; then
	INPUT="${project_root}/artifact/build-${SLUG}.vnc.mp4"
fi

mp4_out="${script_dir}/vm-build.mp4"
gif_out="${script_dir}/vm-build.gif"

if ! command -v ffmpeg >/dev/null 2>&1; then
	echo "error: ffmpeg is required (install it or run 'sailwright install')." >&2
	exit 1
fi
if [[ ! -f "${INPUT}" ]]; then
	echo "error: source recording not found: ${INPUT}" >&2
	echo "       Produce one with 'make quickstart' on a Linux+KVM host (it saves" >&2
	echo "       artifact/build-${SLUG}.vnc.mp4), or pass --input PATH." >&2
	exit 1
fi

# Trim options apply to the SOURCE (before the speed-up filter).
trim_args=()
[[ -n "${START}" ]] && trim_args+=(-ss "${START}")
[[ -n "${DURATION}" ]] && trim_args+=(-t "${DURATION}")

echo "Processing ${INPUT}"
echo "  speed x${SPEED}, mp4 ${WIDTH}px@${FPS}fps, gif ${GIF_WIDTH}px@${GIF_FPS}fps"

# --- mp4 --------------------------------------------------------------------
# setpts=PTS/SPEED speeds playback; -r resamples to a smooth output frame rate.
ffmpeg -hide_banner -loglevel warning -y \
	"${trim_args[@]}" -i "${INPUT}" \
	-vf "setpts=PTS/${SPEED},scale=${WIDTH}:-2:flags=lanczos" \
	-r "${FPS}" \
	-c:v libx264 -preset veryfast -crf 28 -pix_fmt yuv420p -movflags +faststart -an \
	"${mp4_out}"
echo "Wrote ${mp4_out} ($(wc -c <"${mp4_out}") bytes)."

# --- gif (two-pass palette for quality) -------------------------------------
# mktemp creates the extensionless file; keep that name too so the trap cleans
# up both it and the suffixed path ffmpeg actually writes.
tmp_palette_base="$(mktemp "${TMPDIR:-/tmp}/vm-build-palette.XXXXXX")"
tmp_palette="${tmp_palette_base}.png"
tmp_gif_base="$(mktemp "${TMPDIR:-/tmp}/vm-build.XXXXXX")"
tmp_gif="${tmp_gif_base}.gif"
trap 'rm -f "${tmp_palette_base}" "${tmp_palette}" "${tmp_gif_base}" "${tmp_gif}"' EXIT

gif_filters="setpts=PTS/${SPEED},scale=${GIF_WIDTH}:-2:flags=lanczos,fps=${GIF_FPS}"
ffmpeg -hide_banner -loglevel warning -y \
	"${trim_args[@]}" -i "${INPUT}" \
	-vf "${gif_filters},palettegen=stats_mode=diff" \
	-frames:v 1 -update 1 \
	"${tmp_palette}"
ffmpeg -hide_banner -loglevel warning -y \
	"${trim_args[@]}" -i "${INPUT}" -i "${tmp_palette}" \
	-lavfi "${gif_filters}[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=3" \
	"${tmp_gif}"

if command -v gifsicle >/dev/null 2>&1; then
	raw_size=$(wc -c <"${tmp_gif}")
	gifsicle -O3 --colors 256 "${tmp_gif}" -o "${gif_out}"
	echo "Wrote ${gif_out} ($(wc -c <"${gif_out}") bytes, gifsicle from ${raw_size})."
else
	cp -f "${tmp_gif}" "${gif_out}"
	echo "Wrote ${gif_out} ($(wc -c <"${gif_out}") bytes; install gifsicle to shrink it)."
fi
