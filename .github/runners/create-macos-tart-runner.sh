#!/bin/bash
set -e

# This script creates a new ephemeral runner on macOS using tart and registers it with GitHub
# Actions. After each job completes the runner deregisters itself, the VM is shut down, and a
# fresh VM + runner is started automatically to replenish the runner pool.
#
# Required environment variables (or set defaults below):
#   GITHUB_SCOPE      - "repo" (default) or "org"
#   GITHUB_REPO       - owner/repo  (required when GITHUB_SCOPE=repo, e.g. "myorg/myrepo")
#   GITHUB_ORG        - org name    (required when GITHUB_SCOPE=org,  e.g. "myorg")
#   RUNNER_LABELS     - comma-separated runner labels/tags (default: "macos,tart,arm64,macos-16-tart")
#   RUNNER_NAME_BASE  - base name; a timestamp suffix is appended each run (default: <hostname>-tart)
#   VM_SSH_USER       - SSH user inside the tart VM        (default: admin)
#   VM_SSH_PASS       - SSH password inside the tart VM    (default: admin)
#   RUNNER_DIR        - install directory inside the VM    (default: /Users/admin/actions-runner)
#   VM_BASE_IMAGE     - tart base image name               (default: tahoe-base)
#   VM_CLONE_PER_RUN  - clone base image for every run so the VM starts clean (default: true)
#   MAX_RUNS          - maximum number of runner cycles before this script exits (default: 0 = infinite)

# ─── Configuration ─────────────────────────────────────────────────────────────
GITHUB_SCOPE="${GITHUB_SCOPE:-repo}"
GITHUB_REPO="${GITHUB_REPO:-csautter/dev-alchemy}"
GITHUB_ORG="${GITHUB_ORG:-}"
RUNNER_LABELS="${RUNNER_LABELS:-macos,tart,arm64,macos-16-tart}"
RUNNER_NAME_BASE="${RUNNER_NAME_BASE:-$(hostname -s)-tart}"
VM_SSH_USER="${VM_SSH_USER:-admin}"
VM_SSH_PASS="${VM_SSH_PASS:-admin}"
RUNNER_DIR="${RUNNER_DIR:-/Users/admin/actions-runner}"
VM_BASE_IMAGE="${VM_BASE_IMAGE:-tahoe-base}"
VM_CLONE_PER_RUN="${VM_CLONE_PER_RUN:-true}"
MAX_RUNS="${MAX_RUNS:-0}"
# ───────────────────────────────────────────────────────────────────────────────

# ─── Pre-flight checks ─────────────────────────────────────────────────────────
for cmd in tart gh sshpass; do
	if ! command -v "$cmd" &>/dev/null; then
		echo "ERROR: '$cmd' could not be found."
		case "$cmd" in
			tart)     echo "  Install: brew install tart" ;;
			gh)       echo "  Install: brew install gh" ;;
			sshpass)  echo "  Install: brew install sshpass" ;;
		esac
		exit 1
	fi
done

# ─── Resolve the latest runner version once (reused across all cycles) ─────────
echo "Resolving latest GitHub Actions runner version..."
RUNNER_VERSION=$(gh api /repos/actions/runner/releases/latest --jq '.tag_name' | sed 's/^v//')
echo "Runner version: ${RUNNER_VERSION}"
RUNNER_ARCHIVE="actions-runner-osx-arm64-${RUNNER_VERSION}.tar.gz"
RUNNER_DOWNLOAD_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_ARCHIVE}"

# ─── Ensure base image exists ──────────────────────────────────────────────────
if ! tart list | grep -q "^local.*${VM_BASE_IMAGE}"; then
	echo "Cloning base image '${VM_BASE_IMAGE}' from ghcr.io/cirruslabs/macos-tahoe-base:latest ..."
	tart clone ghcr.io/cirruslabs/macos-tahoe-base:latest "${VM_BASE_IMAGE}"
else
	echo "Base image '${VM_BASE_IMAGE}' already present."
fi

# ─── Cleanup helper ────────────────────────────────────────────────────────────
# Tracks the name of the VM that is currently running so the trap can clean up.
CURRENT_VM_NAME=""

cleanup_vm() {
	local vm="${1:-$CURRENT_VM_NAME}"
	[[ -z "$vm" ]] && return
	echo "Stopping VM '${vm}'..."
	tart stop "${vm}" 2>/dev/null || true
	if [[ "$VM_CLONE_PER_RUN" == "true" && "$vm" != "$VM_BASE_IMAGE" ]]; then
		echo "Deleting ephemeral VM clone '${vm}'..."
		tart delete "${vm}" 2>/dev/null || true
	fi
}

# Ensure the VM is stopped/deleted if this script is interrupted or exits.
trap 'cleanup_vm "$CURRENT_VM_NAME"' EXIT INT TERM

# ─── Main runner loop ──────────────────────────────────────────────────────────
RUN_COUNT=0

while true; do
	RUN_COUNT=$(( RUN_COUNT + 1 ))
	TIMESTAMP=$(date +%s)
	RUNNER_NAME="${RUNNER_NAME_BASE}-${TIMESTAMP}"

	echo ""
	echo "════════════════════════════════════════════════════════════"
	echo " Runner cycle #${RUN_COUNT}  |  name: ${RUNNER_NAME}"
	echo "════════════════════════════════════════════════════════════"

	# ── Determine VM name for this cycle ────────────────────────────
	if [[ "$VM_CLONE_PER_RUN" == "true" ]]; then
		VM_NAME="runner-${TIMESTAMP}"
		echo "Cloning '${VM_BASE_IMAGE}' → '${VM_NAME}' for a clean ephemeral VM..."
		tart clone "${VM_BASE_IMAGE}" "${VM_NAME}"
	else
		VM_NAME="${VM_BASE_IMAGE}"
	fi
	CURRENT_VM_NAME="$VM_NAME"

	# ── Fetch a fresh registration token (valid 1 hour) ─────────────
	echo "Fetching runner registration token..."
	if [[ "$GITHUB_SCOPE" == "org" ]]; then
		if [[ -z "$GITHUB_ORG" ]]; then
			echo "GITHUB_ORG must be set when GITHUB_SCOPE=org"
			exit 1
		fi
		RUNNER_TOKEN=$(gh api \
			--method POST \
			-H "Accept: application/vnd.github+json" \
			"/orgs/${GITHUB_ORG}/actions/runners/registration-token" \
			--jq '.token')
		RUNNER_REGISTRATION_URL="https://github.com/${GITHUB_ORG}"
	else
		if [[ -z "$GITHUB_REPO" ]]; then
			echo "GITHUB_REPO must be set (format: owner/repo)"
			exit 1
		fi
		RUNNER_TOKEN=$(gh api \
			--method POST \
			-H "Accept: application/vnd.github+json" \
			"/repos/${GITHUB_REPO}/actions/runners/registration-token" \
			--jq '.token')
		RUNNER_REGISTRATION_URL="https://github.com/${GITHUB_REPO}"
	fi
	echo "Registration token obtained."

	# ── Start VM ────────────────────────────────────────────────────
	# optionally disable graphics with --no-graphics
	# WARNING: exposing ssh port with bridged networking and an insecure password is
	#          only suitable for local/testing use.
	# NOTE: on some systems with strict firewall rules tart VMs may need --net-bridged
	#       to reach the internet.
	echo "Starting VM '${VM_NAME}'..."
	tart run --no-graphics --net-bridged="Wi-Fi" "${VM_NAME}" &

	# ── Wait for an IP ───────────────────────────────────────────────
	VM_IP=""
	while [[ -z "$VM_IP" ]]; do
		VM_IP=$(tart ip --resolver=arp "${VM_NAME}" 2>/dev/null || echo "")
		sleep 1
	done
	echo "VM IP: $VM_IP"

	# ── SSH helper ───────────────────────────────────────────────────
	vm_ssh() {
		sshpass -p "${VM_SSH_PASS}" ssh \
			-o "StrictHostKeyChecking=no" \
			-o "UserKnownHostsFile=/dev/null" \
			-o "ConnectTimeout=5" \
			"${VM_SSH_USER}@${VM_IP}" "$@"
	}

	# ── Wait for SSH ─────────────────────────────────────────────────
	until vm_ssh "true" 2>/dev/null; do
		echo "Waiting for SSH to become available..."
		sleep 2
	done
	echo "SSH connection successful."

	# ── Install, configure and run the runner (foreground) ──────────
	# Running ./run.sh in the foreground means this SSH session blocks until the
	# ephemeral runner picks up a job, completes it, and deregisters itself.
	# Control then returns to this host script which cleans up the VM and loops.
	echo "Installing and starting GitHub Actions runner '${RUNNER_NAME}' (version ${RUNNER_VERSION})..."

	vm_ssh bash <<EOF
set -e

mkdir -p "${RUNNER_DIR}"
cd "${RUNNER_DIR}"

echo "Downloading runner ${RUNNER_ARCHIVE}..."
curl -fsSL -o "${RUNNER_ARCHIVE}" "${RUNNER_DOWNLOAD_URL}"
tar xzf "${RUNNER_ARCHIVE}"
rm -f "${RUNNER_ARCHIVE}"

echo "Configuring runner..."
./config.sh \
	--url "${RUNNER_REGISTRATION_URL}" \
	--token "${RUNNER_TOKEN}" \
	--name "${RUNNER_NAME}" \
	--labels "${RUNNER_LABELS}" \
	--ephemeral \
	--unattended

echo "Runner configured. Waiting for a job (./run.sh)..."
./run.sh
echo "Runner finished."
EOF

	echo "Runner '${RUNNER_NAME}' completed its job and deregistered."

	# ── Shut down VM ─────────────────────────────────────────────────
	cleanup_vm "$CURRENT_VM_NAME"
	CURRENT_VM_NAME=""

	# ── Check cycle limit ────────────────────────────────────────────
	if [[ "$MAX_RUNS" -gt 0 && "$RUN_COUNT" -ge "$MAX_RUNS" ]]; then
		echo "Reached MAX_RUNS=${MAX_RUNS}. Exiting."
		break
	fi

	echo "Restarting runner cycle in 3 seconds..."
	sleep 3
done

echo "Runner pool manager exited after ${RUN_COUNT} run(s)."