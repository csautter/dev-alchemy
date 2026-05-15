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
#   GITHUB_CANCEL_RUN_ON_SHUTDOWN       - cancel active workflow run before VM teardown (default: true)
#   GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN - force-cancel if normal cancel does not settle (default: false)
#   RUNNER_SHUTDOWN_GRACE_SECONDS       - seconds to wait for cancel/deregister before VM stop (default: 120)
#   RUNNER_FORCE_CANCEL_AFTER_SECONDS   - seconds to wait before optional force-cancel (default: 30)
#   RUNNER_SHUTDOWN_POLL_SECONDS        - seconds between runner-state polls during shutdown (default: 5)

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
GITHUB_CANCEL_RUN_ON_SHUTDOWN="${GITHUB_CANCEL_RUN_ON_SHUTDOWN:-true}"
GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN="${GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN:-false}"
RUNNER_SHUTDOWN_GRACE_SECONDS="${RUNNER_SHUTDOWN_GRACE_SECONDS:-120}"
RUNNER_FORCE_CANCEL_AFTER_SECONDS="${RUNNER_FORCE_CANCEL_AFTER_SECONDS:-30}"
RUNNER_SHUTDOWN_POLL_SECONDS="${RUNNER_SHUTDOWN_POLL_SECONDS:-5}"
# Optional: path on the HOST for the general-purpose build cache (ISOs, toolchain
# archives, and other large build dependencies). When set, the directory is shared
# into each VM as /Volumes/My Shared Files/build-cache/ via VirtioFS so workflows
# can symlink cached files instead of downloading them from Azure Blob Storage.
BUILD_CACHE_DIR="${BUILD_CACHE_DIR:-}"
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

# ─── GitHub Actions shutdown helpers ──────────────────────────────────────────
env_truthy() {
	case "$1" in
		1|true|TRUE|yes|YES|on|ON) return 0 ;;
		*) return 1 ;;
	esac
}

shell_number_or_default() {
	local value="$1"
	local default_value="$2"
	case "$value" in
		''|*[!0-9]*) printf '%s\n' "$default_value" ;;
		*) printf '%s\n' "$value" ;;
	esac
}

RUNNER_SHUTDOWN_GRACE_SECONDS="$(shell_number_or_default "$RUNNER_SHUTDOWN_GRACE_SECONDS" 120)"
RUNNER_FORCE_CANCEL_AFTER_SECONDS="$(shell_number_or_default "$RUNNER_FORCE_CANCEL_AFTER_SECONDS" 30)"
RUNNER_SHUTDOWN_POLL_SECONDS="$(shell_number_or_default "$RUNNER_SHUTDOWN_POLL_SECONDS" 5)"

escape_jq_string() {
	local value="$1"
	value=${value//\\/\\\\}
	value=${value//\"/\\\"}
	printf '%s' "$value"
}

runner_list_api_path() {
	if [[ "$GITHUB_SCOPE" == "org" ]]; then
		printf '/orgs/%s/actions/runners\n' "$GITHUB_ORG"
	else
		printf '/repos/%s/actions/runners\n' "$GITHUB_REPO"
	fi
}

runner_delete_api_path() {
	local runner_id="$1"
	if [[ "$GITHUB_SCOPE" == "org" ]]; then
		printf '/orgs/%s/actions/runners/%s\n' "$GITHUB_ORG" "$runner_id"
	else
		printf '/repos/%s/actions/runners/%s\n' "$GITHUB_REPO" "$runner_id"
	fi
}

find_runner_info() {
	local runner_name="$1"
	local runner_name_jq
	runner_name_jq=$(escape_jq_string "$runner_name")

	gh api "$(runner_list_api_path)" --paginate \
		--jq ".runners[] | select(.name == \"${runner_name_jq}\") | [.id, .busy, .status] | @tsv" \
		2>/dev/null | head -n 1 || true
}

find_active_runner_job() {
	local runner_name="$1"
	if [[ "$GITHUB_SCOPE" != "repo" ]]; then
		return 1
	fi
	if [[ -z "$GITHUB_REPO" ]]; then
		return 1
	fi

	local runner_name_jq run_ids run_id job_line
	runner_name_jq=$(escape_jq_string "$runner_name")
	run_ids=$(gh api "/repos/${GITHUB_REPO}/actions/runs?status=in_progress&per_page=100" --paginate \
		--jq '.workflow_runs[].id' 2>/dev/null || true)
	[[ -z "$run_ids" ]] && return 1

	while IFS= read -r run_id; do
		[[ -z "$run_id" ]] && continue
		job_line=$(gh api "/repos/${GITHUB_REPO}/actions/runs/${run_id}/jobs?filter=latest&per_page=100" --paginate \
			--jq ".jobs[] | select(.status == \"in_progress\" and .runner_name == \"${runner_name_jq}\") | [(.run_id // ${run_id}), .id, .name, .html_url] | @tsv" \
			2>/dev/null | head -n 1 || true)
		if [[ -n "$job_line" ]]; then
			printf '%s\n' "$job_line"
			return 0
		fi
	done <<< "$run_ids"

	return 1
}

REQUESTED_CANCEL_RUN_ID=""

cancel_active_runner_job() {
	local vm="$1"
	local runner_name="$2"
	REQUESTED_CANCEL_RUN_ID=""

	if ! env_truthy "$GITHUB_CANCEL_RUN_ON_SHUTDOWN"; then
		echo "[${vm}] Active workflow cancellation disabled by GITHUB_CANCEL_RUN_ON_SHUTDOWN=${GITHUB_CANCEL_RUN_ON_SHUTDOWN}."
		return 1
	fi
	if [[ "$GITHUB_SCOPE" != "repo" ]]; then
		echo "[${vm}] Warning: automatic workflow cancellation currently requires GITHUB_SCOPE=repo."
		echo "[${vm}]          Set GITHUB_REPO to the repository that owns the job for clean Ctrl+C cancellation."
		return 1
	fi

	local job_line run_id job_id job_name job_url
	job_line=$(find_active_runner_job "$runner_name" || true)
	if [[ -z "$job_line" ]]; then
		echo "[${vm}] No in-progress GitHub Actions job found for runner '${runner_name}'."
		return 1
	fi

	IFS=$'\t' read -r run_id job_id job_name job_url <<< "$job_line"
	REQUESTED_CANCEL_RUN_ID="$run_id"

	echo "[${vm}] Canceling workflow run ${run_id} for active job ${job_id} (${job_name})..."
	if gh api --method POST -H "Accept: application/vnd.github+json" \
		"/repos/${GITHUB_REPO}/actions/runs/${run_id}/cancel" >/dev/null 2>&1; then
		echo "[${vm}] Cancellation requested: ${job_url}"
		return 0
	fi

	echo "[${vm}] Warning: GitHub did not accept cancellation for workflow run ${run_id}."
	return 1
}

force_cancel_workflow_run() {
	local vm="$1"
	local run_id="$2"

	[[ -z "$run_id" ]] && return 1
	if ! env_truthy "$GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN"; then
		return 1
	fi
	if [[ "$GITHUB_SCOPE" != "repo" ]]; then
		return 1
	fi

	echo "[${vm}] Workflow run ${run_id} is still active; requesting force-cancel..."
	gh api --method POST -H "Accept: application/vnd.github+json" \
		"/repos/${GITHUB_REPO}/actions/runs/${run_id}/force-cancel" >/dev/null 2>&1
}

signal_runner_processes() {
	local vm="$1"
	local vm_ip="$2"
	[[ -z "$vm_ip" ]] && return

	echo "[${vm}] Sending SIGINT to runner process inside VM..."
	if ! vm_ssh "$vm_ip" bash -s -- "$RUNNER_DIR" <<'EOF' >/dev/null 2>&1; then
set +e
runner_dir="$1"
for pattern in \
	"${runner_dir}/bin/Runner.Listener" \
	"${runner_dir}/bin/Runner.Worker" \
	"${runner_dir}/run.sh"; do
	pkill -INT -f "$pattern" 2>/dev/null || true
done
EOF
		echo "[${vm}] Warning: could not signal runner process over SSH; continuing with GitHub-side cancellation."
	fi
}

wait_for_runner_to_settle() {
	local vm="$1"
	local runner_name="$2"
	local run_id="$3"
	local start_ts now deadline force_after_ts forced runner_info runner_id runner_busy runner_status

	start_ts=$(date +%s)
	deadline=$(( start_ts + RUNNER_SHUTDOWN_GRACE_SECONDS ))
	force_after_ts=$(( start_ts + RUNNER_FORCE_CANCEL_AFTER_SECONDS ))
	forced=false

	while true; do
		now=$(date +%s)
		runner_info=$(find_runner_info "$runner_name")
		if [[ -z "$runner_info" ]]; then
			echo "[${vm}] Runner '${runner_name}' is no longer registered."
			return 0
		fi

		IFS=$'\t' read -r runner_id runner_busy runner_status <<< "$runner_info"
		if [[ "$runner_busy" != "true" ]]; then
			echo "[${vm}] Runner '${runner_name}' is no longer busy (status: ${runner_status})."
			return 0
		fi

		if [[ "$forced" != "true" && -n "$run_id" && "$RUNNER_FORCE_CANCEL_AFTER_SECONDS" -gt 0 && "$now" -ge "$force_after_ts" ]]; then
			force_cancel_workflow_run "$vm" "$run_id" || true
			forced=true
		fi

		if [[ "$now" -ge "$deadline" ]]; then
			echo "[${vm}] Warning: runner '${runner_name}' is still busy after ${RUNNER_SHUTDOWN_GRACE_SECONDS}s."
			return 1
		fi

		sleep "$RUNNER_SHUTDOWN_POLL_SECONDS"
	done
}

# ─── VM cleanup ────────────────────────────────────────────────────────────────
# Usage: cleanup_vm <vm-name> [runner-name] [vm-ip] [cancel-active-job]
cleanup_vm() {
	local vm="$1"
	local runner_name="${2:-}"
	local vm_ip="${3:-}"
	local cancel_active_job="${4:-false}"
	[[ -z "$vm" ]] && return

	# Deregister the runner before stopping the VM (handles Ctrl+C interruption).
	# Ephemeral runners deregister themselves when run.sh receives SIGINT, so we
	# wait briefly and only attempt an explicit removal if the runner is still listed.
	if [[ -n "$runner_name" ]]; then
		if [[ "$cancel_active_job" == "true" ]]; then
			cancel_active_runner_job "$vm" "$runner_name" || true
			signal_runner_processes "$vm" "$vm_ip"
			wait_for_runner_to_settle "$vm" "$runner_name" "$REQUESTED_CANCEL_RUN_ID" || true
		else
			# Give run.sh a moment to finish its own graceful shutdown / self-deregistration.
			sleep 3
		fi

		echo "[${vm}] Deregistering runner '${runner_name}'..."

		# Use the GitHub API directly via the local gh CLI as the last-resort removal
		# path if the ephemeral runner did not already deregister itself.
		local runner_info runner_id runner_busy runner_status
		runner_info=$(find_runner_info "$runner_name")
		IFS=$'\t' read -r runner_id runner_busy runner_status <<< "$runner_info"

		if [[ -n "$runner_id" ]]; then
			local api_path
			api_path=$(runner_delete_api_path "$runner_id")
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
	local current_runner_ssh_pid=""
	local shutdown_requested=false
	local run_count=0

	# shellcheck disable=SC2317 # Invoked by the EXIT trap.
	_worker_cleanup() {
		[[ -z "$current_vm_name" ]] && return
		local cancel_active_job=false
		[[ "$shutdown_requested" == "true" ]] && cancel_active_job=true
		cleanup_vm "$current_vm_name" "$current_runner_name" "$current_vm_ip" "$cancel_active_job"
		if [[ -n "$current_runner_ssh_pid" ]]; then
			kill -TERM "$current_runner_ssh_pid" 2>/dev/null || true
			wait "$current_runner_ssh_pid" 2>/dev/null || true
		fi
		current_vm_name=""
		current_runner_name=""
		current_vm_ip=""
		current_runner_ssh_pid=""
	}
	# EXIT runs the actual cleanup.
	# INT/TERM call exit so the EXIT trap fires and the worker subshell terminates;
	# without an explicit exit here, set +e causes the loop to continue after Ctrl+C.
	trap '_worker_cleanup' EXIT
	trap 'shutdown_requested=true; exit 130' INT
	trap 'shutdown_requested=true; exit 143' TERM

	# Disable set -e within the worker so that a single cycle failure (e.g. an SSH
	# authentication error, a failed clone, or a dropped VM) does not kill the
	# entire worker process.  Each critical step checks its own exit code and uses
	# `continue` to restart the cycle cleanly after VM cleanup.
	set +e

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
			if ! tart clone "${VM_BASE_IMAGE}" "${vm_name}"; then
				echo "[worker-${worker_id}] ERROR: tart clone failed. Retrying in 15s..."
				sleep 15
				continue
			fi
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
		if [[ -z "$runner_token" ]]; then
			echo "[worker-${worker_id}] ERROR: Failed to obtain registration token. Cleaning up and retrying in 30s..."
			cleanup_vm "$current_vm_name"
			current_vm_name=""
			sleep 30
			continue
		fi
		echo "[worker-${worker_id}] Registration token obtained."

		# ── Start VM ────────────────────────────────────────────────
		# WARNING: exposing ssh port with bridged networking and an insecure password is
		#          only suitable for local/testing use.
		# NOTE: on some systems with strict firewall rules tart VMs may need --net-bridged
		#       to reach the internet.
		echo "[worker-${worker_id}] Starting VM '${vm_name}'..."
		local tart_dir_flag=()
		if [[ -n "$BUILD_CACHE_DIR" ]]; then
			echo "[worker-${worker_id}] Sharing build cache '${BUILD_CACHE_DIR}' → /Volumes/My Shared Files/build-cache/ inside VM"
			tart_dir_flag=("--dir=build-cache:${BUILD_CACHE_DIR}")
		fi
		tart run --no-graphics --net-bridged="Wi-Fi" "${tart_dir_flag[@]}" "${vm_name}" &

		# ── Wait for an IP (90 s timeout) ───────────────────────────
		current_vm_ip=""
		local ip_attempts=0
		while [[ -z "$current_vm_ip" ]]; do
			current_vm_ip=$(tart ip --resolver=arp "${vm_name}" 2>/dev/null || true)
			if [[ -z "$current_vm_ip" ]]; then
				ip_attempts=$(( ip_attempts + 1 ))
				if [[ $ip_attempts -ge 90 ]]; then
					echo "[worker-${worker_id}] ERROR: Timed out waiting for VM IP after ${ip_attempts}s. Cleaning up and retrying..."
					break
				fi
				sleep 1
			fi
		done
		if [[ -z "$current_vm_ip" ]]; then
			cleanup_vm "$current_vm_name" "$current_runner_name"
			current_vm_name=""
			current_runner_name=""
			continue
		fi
		echo "[worker-${worker_id}] VM IP: $current_vm_ip"

		# ── Wait for SSH (60 s timeout) ──────────────────────────────
		local ssh_attempts=0
		until vm_ssh "$current_vm_ip" "true" 2>/dev/null; do
			echo "[worker-${worker_id}] Waiting for SSH to become available..."
			ssh_attempts=$(( ssh_attempts + 1 ))
			if [[ $ssh_attempts -ge 30 ]]; then
				echo "[worker-${worker_id}] ERROR: Timed out waiting for SSH after $(( ssh_attempts * 2 ))s. Cleaning up and retrying..."
				current_vm_ip=""
				break
			fi
			sleep 2
		done
		if [[ -z "$current_vm_ip" ]]; then
			cleanup_vm "$current_vm_name" "$current_runner_name"
			current_vm_name=""
			current_runner_name=""
			continue
		fi
		echo "[worker-${worker_id}] SSH connection successful."

		# ── Configure and run the runner ────────────────────────────
		# The runner binary is pre-installed in the golden image by prepare-tart-base.sh.
		# The tracked SSH process blocks this cycle until the ephemeral runner picks
		# up a job, completes it, and deregisters itself. Keeping it as a background
		# process lets TERM/Ctrl+C interrupt wait and run the cleanup trap promptly.
		current_runner_name="$runner_name"
		echo "[worker-${worker_id}] Configuring and starting GitHub Actions runner '${runner_name}'..."

		local runner_exit=0
		vm_ssh "$current_vm_ip" bash <<EOF &
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
		current_runner_ssh_pid=$!
		wait "$current_runner_ssh_pid" || runner_exit=$?
		current_runner_ssh_pid=""

		if [[ $runner_exit -ne 0 ]]; then
			echo "[worker-${worker_id}] ERROR: Runner SSH session failed (exit ${runner_exit}). Cleaning up and retrying in 10s..."
			cleanup_vm "$current_vm_name" "$current_runner_name" "$current_vm_ip" "true"
			current_vm_name=""
			current_runner_name=""
			current_vm_ip=""
			sleep 10
			continue
		fi

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
WORKER_CLEANUP_STARTED=false

cleanup_all_workers() {
	if [[ "$WORKER_CLEANUP_STARTED" == "true" ]]; then
		return
	fi
	WORKER_CLEANUP_STARTED=true
	[[ ${#WORKER_PIDS[@]} -eq 0 ]] && return

	echo "Shutting down all workers..."
	for pid in "${WORKER_PIDS[@]}"; do
		kill -TERM "$pid" 2>/dev/null || true
	done
	wait "${WORKER_PIDS[@]}" 2>/dev/null || true
	WORKER_PIDS=()
}

handle_parent_signal() {
	local exit_code="$1"
	cleanup_all_workers
	exit "$exit_code"
}

trap 'cleanup_all_workers' EXIT
trap 'handle_parent_signal 130' INT
trap 'handle_parent_signal 143' TERM

echo "Starting ${RUNNER_POOL_SIZE} parallel runner worker(s)..."
for i in $(seq 1 "$RUNNER_POOL_SIZE"); do
	run_worker "$i" &
	worker_pid=$!
	WORKER_PIDS+=("$worker_pid")
	echo "Worker #${i} started (PID: ${worker_pid})"
	# Stagger VM boots slightly to avoid simultaneous resource contention.
	[[ "$i" -lt "$RUNNER_POOL_SIZE" ]] && sleep 3
done

# Wait for all workers to finish.
wait "${WORKER_PIDS[@]}" || true
WORKER_PIDS=()
echo "All ${RUNNER_POOL_SIZE} worker(s) have exited."
