#!/bin/bash
set -ex

# This script creates a new runner on macOS using tart and registers it with GitHub Actions.
# The runner registration token is obtained via the gh CLI and is valid for 1 hour.
#
# Required environment variables (or set defaults below):
#   GITHUB_SCOPE   - "repo" (default) or "org"
#   GITHUB_REPO    - owner/repo  (required when GITHUB_SCOPE=repo, e.g. "myorg/myrepo")
#   GITHUB_ORG     - org name    (required when GITHUB_SCOPE=org,  e.g. "myorg")
#   RUNNER_LABELS  - comma-separated runner labels/tags (default: "macos,tart,arm64,macos-16-tart")
#   RUNNER_NAME    - name to register the runner under  (default: <hostname>-tart)
#   VM_SSH_USER    - SSH user inside the tart VM        (default: admin)
#   VM_SSH_PASS    - SSH password inside the tart VM    (default: admin)
#   RUNNER_DIR     - install directory inside the VM    (default: /Users/admin/actions-runner)

# ─── Configuration ─────────────────────────────────────────────────────────────
GITHUB_SCOPE="${GITHUB_SCOPE:-repo}"
GITHUB_REPO="${GITHUB_REPO:-csautter/dev-alchemy}"
GITHUB_ORG="${GITHUB_ORG:-}"
RUNNER_LABELS="${RUNNER_LABELS:-macos,tart,arm64,macos-16-tart}"
RUNNER_NAME="${RUNNER_NAME:-$(hostname -s)-tart}"
VM_SSH_USER="${VM_SSH_USER:-admin}"
VM_SSH_PASS="${VM_SSH_PASS:-admin}"
RUNNER_DIR="${RUNNER_DIR:-/Users/admin/actions-runner}"
# ───────────────────────────────────────────────────────────────────────────────

# check if tart is installed
if ! command -v tart &>/dev/null; then
	echo "tart could not be found, please install it first."
	exit 1
fi

# check if gh CLI is installed and authenticated
if ! command -v gh &>/dev/null; then
	echo "gh CLI could not be found, please install it first (brew install gh)."
	exit 1
fi

# check if sshpass is installed
if ! command -v sshpass &>/dev/null; then
	echo "sshpass could not be found, please install it first (brew install sshpass)."
	exit 1
fi

# ─── Fetch a temporary runner registration token via gh CLI ────────────────────
# Tokens are valid for 1 hour; the gh CLI uses the currently authenticated user.
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
echo "Registration token obtained (valid for 1 hour)."

# ─── Resolve the latest runner version ─────────────────────────────────────────
RUNNER_VERSION=$(gh api /repos/actions/runner/releases/latest --jq '.tag_name' | sed 's/^v//')
echo "Latest runner version: ${RUNNER_VERSION}"
RUNNER_ARCHIVE="actions-runner-osx-arm64-${RUNNER_VERSION}.tar.gz"
RUNNER_DOWNLOAD_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_ARCHIVE}"

# ─── Start the tart VM ─────────────────────────────────────────────────────────
if ! tart list | grep -q "^local.*tahoe-base.*$"; then
	echo "Cloning tahoe-base image..."
	tart clone ghcr.io/cirruslabs/macos-tahoe-base:latest tahoe-base
else
	echo "tahoe-base image already exists."
fi

# optionally disable graphics with --no-graphics
# WARNING: exposing ssh port with bridged networking and insecure ssh password is very insecure and should only be used for testing purposes
# RECOMMENDED: try without --net-bridged first
# NOTE: on some systems with strict firewall rules tart vms might not get internet access without bridged networking
tart run --net-bridged="Wi-Fi" tahoe-base &

# Retry until VM_IP is not empty
VM_IP=""
while [[ -z "$VM_IP" ]]; do
	VM_IP=$(tart ip --resolver=arp tahoe-base 2>/dev/null || echo "")
	sleep 1
done
echo "VM IP: $VM_IP"

# Helper: run a command inside the VM over SSH
vm_ssh() {
	sshpass -p "${VM_SSH_PASS}" ssh \
		-o "StrictHostKeyChecking=no" \
		-o "UserKnownHostsFile=/dev/null" \
		-o "ConnectTimeout=5" \
		"${VM_SSH_USER}@${VM_IP}" "$@"
}

# check ssh connectivity
# Retry SSH until successful
until vm_ssh "echo 'SSH connection successful'"; do
	echo "Waiting for SSH to become available..."
	sleep 2
done

# ─── Install and register the GitHub Actions runner inside the VM ───────────────
echo "Installing GitHub Actions runner (version ${RUNNER_VERSION}) in VM..."

# All ${...} references expand on the host before the script is sent to the VM.
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

echo "Installing and starting runner as a launchd service..."
./svc.sh install
./svc.sh start

echo "Runner installed and started successfully."
EOF

echo "GitHub Actions runner setup complete. Runner '${RUNNER_NAME}' registered at ${RUNNER_REGISTRATION_URL} with labels: ${RUNNER_LABELS}"