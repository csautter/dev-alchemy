#!/bin/bash
set -e

# One-time provisioning script that builds a golden tart VM image with
# pre-installed tooling (Homebrew, Azure CLI, gh CLI, ...).
#
# Run this once – or whenever you need to update the toolset.
# The resulting image is used as VM_BASE_IMAGE by create-macos-tart-runner.sh.
#
# Optional environment variables:
#   TART_SOURCE_IMAGE  - local name for the upstream base  (default: tahoe-base)
#   TART_SOURCE_REMOTE - remote OCI image to pull if missing (default: ghcr.io/cirruslabs/macos-tahoe-base:latest)
#   TART_GOLDEN_IMAGE  - name of the golden image to produce (default: tahoe-runner)
#   VM_SSH_USER        - SSH user inside the VM             (default: admin)
#   VM_SSH_PASS        - SSH password inside the VM         (default: admin)
#   NET_INTERFACE      - bridged network interface          (default: Wi-Fi)
#   RUNNER_DIR         - install directory inside the VM    (default: /Users/admin/actions-runner)
#   RUNNER_VERSION     - pinned runner version (default: resolve latest via gh api)

# ─── Configuration ─────────────────────────────────────────────────────────────
TART_SOURCE_IMAGE="${TART_SOURCE_IMAGE:-tahoe-base}"
TART_SOURCE_REMOTE="${TART_SOURCE_REMOTE:-ghcr.io/cirruslabs/macos-tahoe-base:latest}"
TART_GOLDEN_IMAGE="${TART_GOLDEN_IMAGE:-tahoe-runner}"
VM_SSH_USER="${VM_SSH_USER:-admin}"
VM_SSH_PASS="${VM_SSH_PASS:-admin}"
NET_INTERFACE="${NET_INTERFACE:-Wi-Fi}"
RUNNER_DIR="${RUNNER_DIR:-/Users/admin/actions-runner}"
BUILD_VM="build-${TART_GOLDEN_IMAGE}"
# RUNNER_VERSION resolved below after pre-flight
# ───────────────────────────────────────────────────────────────────────────────

# ─── Pre-flight checks ─────────────────────────────────────────────────────────
for cmd in tart sshpass gh; do
	if ! command -v "$cmd" &>/dev/null; then
		echo "ERROR: '$cmd' not found."
		case "$cmd" in
			tart)    echo "  Install: brew install tart" ;;
			sshpass) echo "  Install: brew install sshpass" ;;
			gh)      echo "  Install: brew install gh" ;;
		esac
		exit 1
	fi
done

# ─── Resolve runner version ────────────────────────────────────────────────────
if [[ -z "${RUNNER_VERSION:-}" ]]; then
	echo "Resolving latest GitHub Actions runner version..."
	RUNNER_VERSION=$(gh api /repos/actions/runner/releases/latest --jq '.tag_name' | sed 's/^v//')
fi
echo "Runner version: ${RUNNER_VERSION}"
RUNNER_ARCHIVE="actions-runner-osx-arm64-${RUNNER_VERSION}.tar.gz"
RUNNER_DOWNLOAD_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_ARCHIVE}"

# ─── Ensure upstream base image is present ────────────────────────────────────
if ! tart list | awk 'NR>1 && $1=="local" {print $2}' | grep -qx "${TART_SOURCE_IMAGE}"; then
	echo "Pulling upstream base image '${TART_SOURCE_IMAGE}' from ${TART_SOURCE_REMOTE} ..."
	tart clone "${TART_SOURCE_REMOTE}" "${TART_SOURCE_IMAGE}"
else
	echo "Upstream base image '${TART_SOURCE_IMAGE}' already present."
fi

# ─── Remove any stale build VM ────────────────────────────────────────────────
if tart list | awk 'NR>1 && $1=="local" {print $2}' | grep -qx "${BUILD_VM}"; then
	echo "Removing stale build VM '${BUILD_VM}'..."
	tart stop "${BUILD_VM}" 2>/dev/null || true
	tart delete "${BUILD_VM}"
fi

# ─── Clone source → ephemeral build VM ───────────────────────────────────────
echo "Cloning '${TART_SOURCE_IMAGE}' → '${BUILD_VM}' ..."
tart clone "${TART_SOURCE_IMAGE}" "${BUILD_VM}"

# ─── Cleanup trap (only deletes build VM on failure) ─────────────────────────
BUILD_SUCCESS=false

cleanup_build_vm() {
	echo "Stopping build VM '${BUILD_VM}'..."
	tart stop "${BUILD_VM}" 2>/dev/null || true
	if [[ "${BUILD_SUCCESS}" != "true" ]]; then
		echo "Provisioning failed – removing build VM."
		tart delete "${BUILD_VM}" 2>/dev/null || true
	fi
}
trap cleanup_build_vm EXIT INT TERM

# ─── Start build VM ───────────────────────────────────────────────────────────
echo "Starting build VM '${BUILD_VM}'..."
tart run --no-graphics --net-bridged="${NET_INTERFACE}" "${BUILD_VM}" &

# ─── Wait for IP ──────────────────────────────────────────────────────────────
VM_IP=""
echo "Waiting for VM IP..."
while [[ -z "$VM_IP" ]]; do
	VM_IP=$(tart ip --resolver=arp "${BUILD_VM}" 2>/dev/null || echo "")
	sleep 1
done
echo "VM IP: ${VM_IP}"

# ─── SSH helper ───────────────────────────────────────────────────────────────
vm_ssh() {
	sshpass -p "${VM_SSH_PASS}" ssh \
		-o "StrictHostKeyChecking=no" \
		-o "UserKnownHostsFile=/dev/null" \
		-o "ConnectTimeout=5" \
		"${VM_SSH_USER}@${VM_IP}" "$@"
}

# ─── Wait for SSH ─────────────────────────────────────────────────────────────
until vm_ssh "true" 2>/dev/null; do
	echo "Waiting for SSH to become available..."
	sleep 2
done
echo "SSH connection successful."

# ─── Provision tools ──────────────────────────────────────────────────────────
echo "Starting provisioning..."

vm_ssh bash <<'PROVISION'
set -e

BREW="/opt/homebrew/bin/brew"
BREW_PROFILE_LINE='eval "$(/opt/homebrew/bin/brew shellenv)"'

# ── Homebrew ──────────────────────────────────────────────────────────────────
if [[ ! -x "$BREW" ]]; then
	echo "Installing Homebrew..."
	NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
else
	echo "Homebrew already installed."
fi

# Ensure brew is on PATH for login shells
if ! grep -qF "$BREW_PROFILE_LINE" ~/.zprofile 2>/dev/null; then
	echo "$BREW_PROFILE_LINE" >> ~/.zprofile
fi
eval "$(/opt/homebrew/bin/brew shellenv)"

# Make Homebrew binaries available system-wide (non-login shells, GH Actions runner)
if [[ ! -f /etc/paths.d/homebrew ]]; then
	echo "Registering Homebrew in /etc/paths.d/homebrew..."
	printf '/opt/homebrew/bin\n/opt/homebrew/sbin\n' | sudo tee /etc/paths.d/homebrew > /dev/null
fi

# ── Azure CLI ─────────────────────────────────────────────────────────────────
if ! command -v az &>/dev/null; then
	echo "Installing Azure CLI..."
	brew install azure-cli
else
	echo "Azure CLI already installed."
fi

# ── GitHub CLI ────────────────────────────────────────────────────────────────
if ! command -v gh &>/dev/null; then
	echo "Installing GitHub CLI..."
	brew install gh
else
	echo "GitHub CLI already installed."
fi

# ── Additional tools (uncomment as needed) ────────────────────────────────────
# brew install terraform
# brew install packer
# brew install ansible
# brew install jq

echo ""
echo "Provisioning complete. Installed versions:"
"$BREW" --version | head -1
az version --query '"azure-cli"' -o tsv 2>/dev/null | xargs echo "  azure-cli:"
gh --version 2>/dev/null | head -1 | xargs echo "  gh:"
PROVISION

echo "Provisioning finished successfully."

# ─── Download and install GitHub Actions runner ───────────────────────────────
echo "Installing GitHub Actions runner ${RUNNER_VERSION} into golden image..."

vm_ssh bash <<EOF
set -e

mkdir -p "${RUNNER_DIR}"
cd "${RUNNER_DIR}"

echo "Downloading runner ${RUNNER_ARCHIVE}..."
curl -fsSL -o "${RUNNER_ARCHIVE}" "${RUNNER_DOWNLOAD_URL}"
tar xzf "${RUNNER_ARCHIVE}"
rm -f "${RUNNER_ARCHIVE}"

echo "Runner ${RUNNER_VERSION} installed to ${RUNNER_DIR}."
EOF

echo "Runner installation complete."

# ─── Graceful shutdown ────────────────────────────────────────────────────────
echo "Shutting down build VM..."
vm_ssh "sudo shutdown -h now" 2>/dev/null || true
sleep 8
tart stop "${BUILD_VM}" 2>/dev/null || true

# Mark success before the trap fires
BUILD_SUCCESS=true

# ─── Promote build VM → golden image ─────────────────────────────────────────
if tart list | awk 'NR>1 && $1=="local" {print $2}' | grep -qx "${TART_GOLDEN_IMAGE}"; then
	echo "Removing previous golden image '${TART_GOLDEN_IMAGE}'..."
	tart delete "${TART_GOLDEN_IMAGE}"
fi

echo "Cloning '${BUILD_VM}' → '${TART_GOLDEN_IMAGE}' (golden image)..."
tart clone "${BUILD_VM}" "${TART_GOLDEN_IMAGE}"
tart delete "${BUILD_VM}"

echo ""
echo "════════════════════════════════════════════════════════════"
echo " Golden image '${TART_GOLDEN_IMAGE}' is ready."
echo " Runner ${RUNNER_VERSION} pre-installed at ${RUNNER_DIR}."
echo " Use it with: VM_BASE_IMAGE=${TART_GOLDEN_IMAGE} ./create-macos-tart-runner.sh"
echo "════════════════════════════════════════════════════════════"
