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
#   RUNNER_LABELS     - comma-separated runner labels/tags (default: "macos,tart,arm64,macos-26-tart")
#   RUNNER_NAME_BASE  - base name; a timestamp suffix is appended each run (default: <hostname>-tart)
#   VM_SSH_USER       - SSH user inside the tart VM        (default: admin)
#   VM_SSH_PASS       - SSH password inside the tart VM    (default: admin)
#   RUNNER_DIR        - install directory inside the VM    (default: /Users/admin/actions-runner)
#   VM_BASE_IMAGE     - tart base image name               (default: tahoe-base)
#   VM_CLONE_PER_RUN  - clone base image for every run so the VM starts clean (default: true)
#   MAX_RUNS          - maximum number of runner cycles per worker before this script exits (default: 0 = infinite)
#   RUNNER_POOL_SIZE  - number of parallel runner workers / VMs to run simultaneously (default: 1)
#   VM_CPU_COUNT      - number of vCPU cores to assign to each VM (default: image default)
#   VM_MEMORY_MB      - memory in MiB to assign to each VM (default: image default)

# ─── Configuration ─────────────────────────────────────────────────────────────
GITHUB_SCOPE="${GITHUB_SCOPE:-repo}"
GITHUB_REPO="${GITHUB_REPO:-csautter/dev-alchemy}"
GITHUB_ORG="${GITHUB_ORG:-}"
RUNNER_LABELS="${RUNNER_LABELS:-macos,tart,arm64,macos-26-tart}"
RUNNER_NAME_BASE="${RUNNER_NAME_BASE:-$(hostname -s)-tart}"
VM_SSH_USER="${VM_SSH_USER:-admin}"
VM_SSH_PASS="${VM_SSH_PASS:-admin}"
RUNNER_DIR="${RUNNER_DIR:-/Users/admin/actions-runner}"
VM_BASE_IMAGE="${VM_BASE_IMAGE:-tahoe-runner}"
VM_CLONE_PER_RUN="${VM_CLONE_PER_RUN:-true}"
MAX_RUNS="${MAX_RUNS:-0}"
# Number of parallel runner workers; each worker manages its own VM and runner cycle.
RUNNER_POOL_SIZE="${RUNNER_POOL_SIZE:-1}"
# VM resource overrides — leave empty to use the image defaults.
VM_CPU_COUNT="${VM_CPU_COUNT:-}"   # vCPU cores,   e.g. 4
VM_MEMORY_MB="${VM_MEMORY_MB:-}"   # memory (MiB), e.g. 8192
# Optional: path on the HOST where Windows ISOs are cached.
# When set the directory is shared into each VM as /Volumes/iso-cache/ via VirtioFS,
# so the workflow can symlink the ISO instead of downloading it from Azure.
ISO_CACHE_DIR="${ISO_CACHE_DIR:-}"
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

# ─── Ensure base image exists ──────────────────────────────────────────────────
if ! tart list | awk 'NR>1 && $1=="local" {print $2}' | grep -qx "${VM_BASE_IMAGE}"; then
	echo "ERROR: Base image '${VM_BASE_IMAGE}' not found."
	echo "  Run './prepare-tart-base.sh' first to build the golden image."
	echo "  Or set VM_BASE_IMAGE=tahoe-base to use the upstream image directly."
	exit 1
fi
echo "Base image '${VM_BASE_IMAGE}' present."

# ─── SSH helper ────────────────────────────────────────────────────────────────
# Usage: vm_ssh <ip> [ssh-args...]
vm_ssh() {
	local ip="$1"; shift
	sshpass -p "${VM_SSH_PASS}" ssh \
		-o "StrictHostKeyChecking=no" \
		-o "UserKnownHostsFile=/dev/null" \
		-o "ConnectTimeout=5" \
		"${VM_SSH_USER}@${ip}" "$@"
}

# ─── VM cleanup ────────────────────────────────────────────────────────────────
# Usage: cleanup_vm <vm-name> [runner-name]
cleanup_vm() {
	local vm="$1"
	local runner_name="${2:-}"
	[[ -z "$vm" ]] && return

	# Deregister the runner before stopping the VM (handles Ctrl+C interruption).
	# Ephemeral runners deregister themselves when run.sh receives SIGINT, so we
	# wait briefly and only attempt an explicit removal if the runner is still listed.
	if [[ -n "$runner_name" ]]; then
		echo "[${vm}] Deregistering runner '${runner_name}'..."
		# Give run.sh a moment to finish its own graceful shutdown / self-deregistration.
		sleep 3

		# Use the GitHub API directly via the local gh CLI. This is the only reliable
		# last-resort path: SSH into the VM never works during an unclean shutdown.
		local runner_id
		if [[ "$GITHUB_SCOPE" == "org" ]]; then
			runner_id=$(gh api "/orgs/${GITHUB_ORG}/actions/runners" \
				--jq ".runners[] | select(.name == \"${runner_name}\") | .id" 2>/dev/null || true)
		else
			runner_id=$(gh api "/repos/${GITHUB_REPO}/actions/runners" \
				--jq ".runners[] | select(.name == \"${runner_name}\") | .id" 2>/dev/null || true)
		fi

		if [[ -n "$runner_id" ]]; then
			local api_path
			if [[ "$GITHUB_SCOPE" == "org" ]]; then
				api_path="/orgs/${GITHUB_ORG}/actions/runners/${runner_id}"
			else
				api_path="/repos/${GITHUB_REPO}/actions/runners/${runner_id}"
			fi
			if gh api --method DELETE "$api_path" 2>/dev/null; then
				echo "[${vm}] Runner '${runner_name}' deregistered via GitHub API."
			else
				echo "[${vm}] Warning: Could not deregister runner '${runner_name}'; manual cleanup may be required."
			fi
		else
			echo "[${vm}] Runner '${runner_name}' not found; likely already self-deregistered."
		fi
	fi

	echo "[${vm}] Stopping VM..."
	tart stop "${vm}" 2>/dev/null || true
	if [[ "$VM_CLONE_PER_RUN" == "true" && "$vm" != "$VM_BASE_IMAGE" ]]; then
		echo "[${vm}] Deleting ephemeral VM clone..."
		tart delete "${vm}" 2>/dev/null || true
	fi
}

# ─── Single worker (runs one runner cycle at a time, loops until MAX_RUNS) ─────
run_worker() {
	local worker_id="$1"
	local current_vm_name=""
	local current_runner_name=""
	local current_vm_ip=""
	local run_count=0

	_worker_cleanup() {
		[[ -z "$current_vm_name" ]] && return
		cleanup_vm "$current_vm_name" "$current_runner_name"
		current_vm_name=""
		current_runner_name=""
	}
	trap '_worker_cleanup' EXIT INT TERM

	while true; do
		run_count=$(( run_count + 1 ))
		local timestamp
		timestamp=$(date +%s)
		local runner_name="${RUNNER_NAME_BASE}-w${worker_id}-${timestamp}"

		echo ""
		echo "════════════════════════════════════════════════════════════"
		echo " Worker #${worker_id} | Cycle #${run_count} | Runner: ${runner_name}"
		echo "════════════════════════════════════════════════════════════"

		# ── Determine VM name for this cycle ────────────────────────
		local vm_name
		if [[ "$VM_CLONE_PER_RUN" == "true" ]]; then
			vm_name="runner-w${worker_id}-${timestamp}"
			echo "[worker-${worker_id}] Cloning '${VM_BASE_IMAGE}' → '${vm_name}'..."
			tart clone "${VM_BASE_IMAGE}" "${vm_name}"
		else
			vm_name="${VM_BASE_IMAGE}"
		fi
		current_vm_name="$vm_name"

		# ── Apply CPU / memory overrides ─────────────────────────────
		if [[ -n "$VM_CPU_COUNT" || -n "$VM_MEMORY_MB" ]]; then
			local set_args=()
			[[ -n "$VM_CPU_COUNT" ]] && set_args+=(--cpu "$VM_CPU_COUNT")
			[[ -n "$VM_MEMORY_MB" ]] && set_args+=(--memory "$VM_MEMORY_MB")
			echo "[worker-${worker_id}] Applying VM resource overrides: ${set_args[*]}"
			tart set "${vm_name}" "${set_args[@]}"
		fi

		# ── Fetch a fresh registration token (valid 1 hour) ─────────
		echo "[worker-${worker_id}] Fetching runner registration token..."
		local runner_token runner_registration_url
		if [[ "$GITHUB_SCOPE" == "org" ]]; then
			if [[ -z "$GITHUB_ORG" ]]; then
				echo "GITHUB_ORG must be set when GITHUB_SCOPE=org"
				exit 1
			fi
			runner_token=$(gh api \
				--method POST \
				-H "Accept: application/vnd.github+json" \
				"/orgs/${GITHUB_ORG}/actions/runners/registration-token" \
				--jq '.token')
			runner_registration_url="https://github.com/${GITHUB_ORG}"
		else
			if [[ -z "$GITHUB_REPO" ]]; then
				echo "GITHUB_REPO must be set (format: owner/repo)"
				exit 1
			fi
			runner_token=$(gh api \
				--method POST \
				-H "Accept: application/vnd.github+json" \
				"/repos/${GITHUB_REPO}/actions/runners/registration-token" \
				--jq '.token')
			runner_registration_url="https://github.com/${GITHUB_REPO}"
		fi
		echo "[worker-${worker_id}] Registration token obtained."

		# ── Start VM ────────────────────────────────────────────────
		# WARNING: exposing ssh port with bridged networking and an insecure password is
		#          only suitable for local/testing use.
		# NOTE: on some systems with strict firewall rules tart VMs may need --net-bridged
		#       to reach the internet.
		echo "[worker-${worker_id}] Starting VM '${vm_name}'..."
		local tart_dir_flag=()
		if [[ -n "$ISO_CACHE_DIR" ]]; then
			echo "[worker-${worker_id}] Sharing ISO cache '${ISO_CACHE_DIR}' → /Volumes/My Shared Files/iso-cache/ inside VM"
			tart_dir_flag=("--dir=iso-cache:${ISO_CACHE_DIR}")
		fi
		tart run --no-graphics --net-bridged="Wi-Fi" "${tart_dir_flag[@]}" "${vm_name}" &

		# ── Wait for an IP ───────────────────────────────────────────
		current_vm_ip=""
		while [[ -z "$current_vm_ip" ]]; do
			current_vm_ip=$(tart ip --resolver=arp "${vm_name}" 2>/dev/null || echo "")
			sleep 1
		done
		echo "[worker-${worker_id}] VM IP: $current_vm_ip"

		# ── Wait for SSH ─────────────────────────────────────────────
		until vm_ssh "$current_vm_ip" "true" 2>/dev/null; do
			echo "[worker-${worker_id}] Waiting for SSH to become available..."
			sleep 2
		done
		echo "[worker-${worker_id}] SSH connection successful."

		# ── Configure and run the runner (foreground) ───────────────
		# The runner binary is pre-installed in the golden image by prepare-tart-base.sh.
		# Running ./run.sh in the foreground means this SSH session blocks until the
		# ephemeral runner picks up a job, completes it, and deregisters itself.
		# Control then returns to this worker which cleans up the VM and loops.
		current_runner_name="$runner_name"
		echo "[worker-${worker_id}] Configuring and starting GitHub Actions runner '${runner_name}'..."

		vm_ssh "$current_vm_ip" bash <<EOF
set -e

cd "${RUNNER_DIR}"

echo "Configuring runner..."
./config.sh \
	--url "${runner_registration_url}" \
	--token "${runner_token}" \
	--name "${runner_name}" \
	--labels "${RUNNER_LABELS}" \
	--ephemeral \
	--unattended

# Inject PATH into the runner .env so every job inherits Homebrew binaries.
# The runner process is a non-login shell and never sources ~/.zprofile or
# /etc/paths.d – writing to .env is the supported way to set env vars for
# self-hosted runners.
echo 'PATH=/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/go/bin:/Users/admin/go/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin' >> .env
echo 'GOPATH=/Users/admin/go' >> .env

echo "Runner configured. Waiting for a job (./run.sh)..."
./run.sh
echo "Runner finished."
EOF

		echo "[worker-${worker_id}] Runner '${runner_name}' completed its job and deregistered."

		# ── Shut down VM ─────────────────────────────────────────────
		cleanup_vm "$current_vm_name" "$current_runner_name"
		current_vm_name=""
		current_runner_name=""
		current_vm_ip=""

		# ── Check cycle limit ────────────────────────────────────────
		if [[ "$MAX_RUNS" -gt 0 && "$run_count" -ge "$MAX_RUNS" ]]; then
			echo "[worker-${worker_id}] Reached MAX_RUNS=${MAX_RUNS}. Worker exiting."
			break
		fi

		echo "[worker-${worker_id}] Restarting runner cycle in 3 seconds..."
		sleep 3
	done

	echo "[worker-${worker_id}] Worker exited after ${run_count} run(s)."
}

# ─── Launch parallel workers ───────────────────────────────────────────────────
if [[ "$VM_CLONE_PER_RUN" != "true" && "$RUNNER_POOL_SIZE" -gt 1 ]]; then
	echo "ERROR: VM_CLONE_PER_RUN=false with RUNNER_POOL_SIZE>1 — all workers would share"
	echo "       the same base VM, causing conflicts. Set VM_CLONE_PER_RUN=true."
	exit 1
fi

WORKER_PIDS=()

cleanup_all_workers() {
	echo "Shutting down all workers..."
	for pid in "${WORKER_PIDS[@]}"; do
		kill "$pid" 2>/dev/null || true
	done
	wait "${WORKER_PIDS[@]}" 2>/dev/null || true
}

trap 'cleanup_all_workers' EXIT INT TERM

echo "Starting ${RUNNER_POOL_SIZE} parallel runner worker(s)..."
for i in $(seq 1 "$RUNNER_POOL_SIZE"); do
	run_worker "$i" &
	WORKER_PIDS+=("$!")
	echo "Worker #${i} started (PID: ${WORKER_PIDS[-1]})"
	# Stagger VM boots slightly to avoid simultaneous resource contention.
	[[ "$i" -lt "$RUNNER_POOL_SIZE" ]] && sleep 3
done

# Wait for all workers to finish.
wait "${WORKER_PIDS[@]}" || true
echo "All ${RUNNER_POOL_SIZE} worker(s) have exited."