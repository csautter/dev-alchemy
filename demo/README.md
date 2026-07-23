# Sailwright terminal demo

A short, polished terminal demo that shows the real Sailwright quick-start
commands being **typed** at a prompt, for embedding on the website with
[asciinema-player](https://github.com/asciinema/asciinema-player).

The cast is **generated**, not recorded — so it is short, clean, and
byte-for-byte reproducible. CI verifies the committed cast stays in sync with the
source (see [../.github/workflows/generate-demo.yml](../.github/workflows/generate-demo.yml)).

## Files

| File | Purpose |
| --- | --- |
| [`quickstart.demo`](quickstart.demo) | Source of truth: the commands + representative output (edit this) |
| [`generate_cast.py`](generate_cast.py) | Deterministic generator: `.demo` → `.cast` (Python stdlib only) |
| `quickstart.cast` | Generated, committed asciinema v2 cast (the embeddable asset) |
| [`index.html`](index.html) | Example page embedding the player |
| [`fetch-player.sh`](fetch-player.sh) | Download a pinned player into `vendor/` (gitignored) |
| `quickstart.gif` | Rendered GIF for the GitHub README (JS players don't run there) |
| [`render-gif.sh`](render-gif.sh) | Render `quickstart.cast` → `quickstart.gif` via `agg` |
| `vm-build.mp4` / `vm-build.gif` | Curated clip of the **real** VM build (shown beside the terminal) |
| [`process-build-video.sh`](process-build-video.sh) | Turn a raw build recording into `vm-build.mp4` + `.gif` |

The page ([`index.html`](index.html)) shows the terminal cast and the VM-build
video **side by side** — the commands on the left, the unattended VM build on the
right.

## Regenerate

After editing `quickstart.demo`:

```bash
make demo          # from the repo root
# or: python3 demo/generate_cast.py
```

The generator is deterministic (fixed seed + fixed header timestamp), so
regenerating without content changes produces an identical file. CI fails if the
committed `quickstart.cast` is stale.

To also refresh the GitHub-README GIF after regenerating the cast:

```bash
make demo-gif      # renders demo/quickstart.gif via agg
```

`render-gif.sh` downloads a pinned `agg` into `vendor/` if it isn't on `PATH`. It
needs a monospace + colour-emoji font, plus `gifsicle` to optimise the result
(`gifsicle -O3 --colors 256` roughly halves the file with no visible quality
loss; it is skipped with a warning if `gifsicle` is missing). On Debian/Ubuntu:

```bash
sudo apt-get install -y fonts-dejavu-core fonts-noto-color-emoji fontconfig gifsicle
```

## VM-build video

`vm-build.mp4` / `vm-build.gif` are a curated clip of the **real** VM build — the
Ubuntu autoinstall running unattended during `sailwright build`. Unlike the
terminal cast (which is generated), this is a real recording, so it must be
produced on a Linux + KVM host:

```bash
# 1) Produce the raw recording on a KVM host. `sailwright build` records the
#    QEMU screen at 1 fps; make quickstart collects it into artifact/.
make quickstart
#    -> artifact/build-ubuntu-server-amd64.vnc.mp4
#    (the KVM CI job in test-quickstart-linux.yml also uploads *.vnc.mp4)

# 2) Turn that 1 fps slideshow into a short, smooth, web-ready clip:
make demo-build-video
#    -> demo/vm-build.mp4  (website <video>)
#    -> demo/vm-build.gif  (README)
```

`process-build-video.sh` speeds up, scales, and trims the source, then encodes an
H.264 mp4 and a palette-optimised GIF (`gifsicle -O3 --colors 256`). Requires
`ffmpeg` (a project dependency) and `gifsicle`. Tune it via `DEMO_BUILD_ARGS`,
e.g. to trim the boot tail and speed it up more:

```bash
make demo-build-video DEMO_BUILD_ARGS="--speed 16 --start 8 --width 720"
```

Run `bash demo/process-build-video.sh --help` for all flags.

## Preview locally

```bash
bash demo/fetch-player.sh          # downloads the player into demo/vendor/
python3 -m http.server -d demo 8000
# open http://localhost:8000/
```

## Embed on the website

Self-host the two player assets (`asciinema-player.min.js`,
`asciinema-player.css`) and `quickstart.cast`, then:

```html
<link rel="stylesheet" href="/path/to/asciinema-player.css" />
<div id="sailwright-demo"></div>
<script src="/path/to/asciinema-player.min.js"></script>
<script>
  AsciinemaPlayer.create(
    "/path/to/quickstart.cast",
    document.getElementById("sailwright-demo"),
    { autoPlay: true, loop: true, idleTimeLimit: 1.5, cols: 100, rows: 28 }
  );
</script>
```

> asciinema-player is a JS widget, so it works on your own site but **not** inside
> a GitHub README (GitHub strips `<script>`). The committed `quickstart.gif`
> (rendered with [`agg`](https://github.com/asciinema/agg) via `make demo-gif`) is
> what the top-level README embeds.
