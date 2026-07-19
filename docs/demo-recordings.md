# Demo Recordings

This guide collects everything related to producing the terminal casts, GIFs,
and videos used for the GitHub README and the website. Unlike the rest of
[Testing Workflows](testing-workflows.md), none of this is required to
validate a build — it exists purely to generate demo assets.

## Quick-Start Recordings (Real Run)

Running [`scripts/quickstart.sh`](../scripts/quickstart.sh) (see
[Automated Quick-Start](quickstart-automation.md)) produces two complementary
recordings in the `--record-dir` (default
`./artifact`), unless `--no-record` is passed:

- **Terminal session** — `quickstart-<slug>.cast` via
  [asciinema](https://asciinema.org) (installed by
  [scripts/linux/sailwright-install-dependencies.sh](../scripts/linux/sailwright-install-dependencies.sh)).
  If asciinema is missing it falls back to a `quickstart-<slug>.typescript` via
  the `script` utility.
- **VM graphical display** — `quickstart-<slug>.vnc.mp4`, captured from the
  running libvirt domain with `vncsnapshot` + `ffmpeg`, reusing the same
  pipeline as the packer build recordings
  ([pkg/build/vnc_recording.go](../pkg/build/vnc_recording.go)). This is
  best-effort: if `virsh`/`vncsnapshot`/`ffmpeg` or the VNC endpoint is
  unavailable, it is skipped with a warning and the workflow still runs.

The packer `build` step also emits its own `*.vnc.mp4` for the build VM under
the managed cache dir; the quick-start collects any produced during the run into
the record dir as `build-<slug>.vnc.mp4`. So a full run yields up to two videos —
`build-<slug>.vnc.mp4` (OS install during build) and `quickstart-<slug>.vnc.mp4`
(create/start/provision).

> First-run note: on a machine without asciinema, the very first run records the
> terminal as `quickstart-<slug>.typescript` via `script`, because asciinema is
> only installed during that run's install step. A later run — or one with
> `--skip-install` once dependencies are present — records a cleaner
> `quickstart-<slug>.cast` instead.

## Website Terminal Demo (Generated)

A short, polished terminal demo that shows the real quick-start commands being
**typed** — for embedding on the website via
[asciinema-player](https://github.com/asciinema/asciinema-player) — lives in
[demo/](../demo/). The cast is generated (not recorded), so it stays short,
clean, and reproducible:

```bash
make demo   # regenerate demo/quickstart.cast from demo/quickstart.demo
```

Edit [demo/quickstart.demo](../demo/quickstart.demo) to change the script; CI
([.github/workflows/generate-demo.yml](../.github/workflows/generate-demo.yml))
fails if the committed cast is out of date. See [demo/README.md](../demo/README.md)
for the embed snippet and local preview instructions.

The top-level README embeds a GIF rendered from the same cast (asciinema-player
JS cannot run in a GitHub README). Refresh it with `make demo-gif` after changing
the demo.
