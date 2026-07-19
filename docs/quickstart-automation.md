# Automated Quick-Start (Recorded End-to-End Run)

For automated testing of the whole lifecycle, `scripts/quickstart.sh` runs the
full quick-start workflow for one target in a single command and records the
run. It is primarily supported on **Linux + QEMU/KVM (libvirt)**.

It runs, in order:

```text
install -> build -> create -> start -> provision --check -> provision -> (stop -> destroy)
```

Cleanup (`stop` + `destroy`) always runs at the end via a shell trap, even if a
step fails, unless you pass `--keep`. The run is also self-healing: just before
`create` it removes any leftover VM from a previous run, so a stale domain does
not abort the workflow with "... already exists".

Run it through the Makefile:

```bash
make quickstart
make quickstart TYPE=desktop ARCH=amd64
make quickstart QUICKSTART_ARGS="--keep"
make quickstart QUICKSTART_ARGS="--no-cache"   # force a rebuild
```

Or call the script directly:

```bash
bash ./scripts/quickstart.sh --os ubuntu --type server --arch amd64
```

Flags:

| Flag | Meaning |
| --- | --- |
| `--os NAME` | Guest OS (default `ubuntu`) |
| `--type TYPE` | Ubuntu type `server`/`desktop` (default `server`; ignored for non-ubuntu) |
| `--arch ARCH` | `amd64`/`arm64` (default `amd64`) |
| `--skip-install` | Skip `sailwright install` when dependencies are already present |
| `--skip-build` | Skip `sailwright build` and reuse an existing/pulled artifact |
| `--no-cache` | Force a rebuild even if the build artifact already exists (passed to `sailwright build`) |
| `--keep` | Leave the VM running instead of `stop` + `destroy` |
| `--no-record` | Disable all recording |
| `--record-dir DIR` | Output directory for recordings (default `./artifact`) |

### Recordings

Unless `--no-record` is passed, the run also produces terminal and VM-graphical
recordings under `--record-dir` (default `./artifact`) — see
[Demo Recordings](demo-recordings.md#quick-start-recordings-real-run) for the
formats, file naming, and first-run caveats.

### Provisioning credentials

The Ubuntu provision step needs SSH credentials for the packer-built image. Set
them in project-root `.env` or the environment before running:

```dotenv
LIBVIRT_UBUNTU_ANSIBLE_USER=packer
LIBVIRT_UBUNTU_ANSIBLE_PASSWORD=P@ssw0rd!
LIBVIRT_UBUNTU_ANSIBLE_BECOME_PASSWORD=P@ssw0rd!
```

### CI

[.github/workflows/test-quickstart-linux.yml](../.github/workflows/test-quickstart-linux.yml)
provides two jobs: a lightweight `lint-quickstart` job (syntax + ShellCheck) on
every PR/push that touches the script, and a manual (`workflow_dispatch`)
`quickstart-e2e` job that boots a VM, runs the recorded workflow, and uploads
the `.vnc.mp4` and `.cast`/`.typescript` recordings as artifacts. The e2e job
defaults to pulling a prebuilt Ubuntu OCI artifact and running with
`--skip-build` to keep runtime bounded.
