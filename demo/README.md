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
