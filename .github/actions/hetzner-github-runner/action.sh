#!/usr/bin/env bash

set -euo pipefail

WAIT_SECONDS=10
CREATED_SERVER_ID=""

fail() {
	echo "FAILURE: $*" >&2
	exit 1
}

require_command() {
	local cmd="$1"
	command -v "$cmd" >/dev/null 2>&1 || fail "Required command '$cmd' was not found."
}

require_integer() {
	local value="$1"
	local name="$2"
	[[ "$value" =~ ^[0-9]+$ ]] || fail "$name must be an integer."
}

require_positive_integer() {
	local value="$1"
	local name="$2"
	require_integer "$value" "$name"
	(( value > 0 )) || fail "$name must be greater than zero."
}

require_boolean() {
	local value="$1"
	local name="$2"
	[[ "$value" == "true" || "$value" == "false" ]] || fail "$name must be 'true' or 'false'."
}

validate_server_type_location() {
	local metadata_file="${RUNNER_STATE_DIR}/server-types.json"
	local http_code
	local supported_locations
	local available_locations
	local recommended_locations

	http_code="$(hetzner_api "GET" "/server_types?per_page=100" "$metadata_file")"
	if [[ "$http_code" != "200" ]]; then
		echo "Hetzner server types response (${http_code}):" >&2
		cat "$metadata_file" >&2 || true
		fail "Failed to validate the Hetzner server-type/location combination."
	fi

	if ! jq -e --arg server_type "$MY_SERVER_TYPE" '.server_types[]? | select((.name | ascii_downcase) == $server_type)' "$metadata_file" >/dev/null; then
		fail "Hetzner server-type '${MY_SERVER_TYPE}' was not found in the current Cloud API response."
	fi

	if jq -e --arg server_type "$MY_SERVER_TYPE" --arg location "$MY_LOCATION" '
		any(
			.server_types[]?
			| select((.name | ascii_downcase) == $server_type)
			| .locations[]?;
			((.name // .location.name // "") | ascii_downcase) == $location
		)
	' "$metadata_file" >/dev/null; then
		return 0
	fi

	supported_locations="$(
		jq -r --arg server_type "$MY_SERVER_TYPE" '
			[
				.server_types[]?
				| select((.name | ascii_downcase) == $server_type)
				| .locations[]?
				| (.name // .location.name // empty)
				| ascii_downcase
			] | unique | join(", ")
		' "$metadata_file"
	)"

	available_locations="$(
		jq -r --arg server_type "$MY_SERVER_TYPE" '
			[
				.server_types[]?
				| select((.name | ascii_downcase) == $server_type)
				| .locations[]?
				| select((.available // false) == true)
				| (.name // .location.name // empty)
				| ascii_downcase
			] | unique | join(", ")
		' "$metadata_file"
	)"

	recommended_locations="$(
		jq -r --arg server_type "$MY_SERVER_TYPE" '
			[
				.server_types[]?
				| select((.name | ascii_downcase) == $server_type)
				| .locations[]?
				| select((.recommended // false) == true)
				| (.name // .location.name // empty)
				| ascii_downcase
			] | unique | join(", ")
		' "$metadata_file"
	)"

	[[ -n "$supported_locations" ]] || supported_locations="none reported by Hetzner"
	if [[ -n "$recommended_locations" ]]; then
		fail "Hetzner server-type '${MY_SERVER_TYPE}' is not supported in location '${MY_LOCATION}'. Supported locations: ${supported_locations}. Recommended locations: ${recommended_locations}."
	fi
	if [[ -n "$available_locations" ]]; then
		fail "Hetzner server-type '${MY_SERVER_TYPE}' is not supported in location '${MY_LOCATION}'. Supported locations: ${supported_locations}. Currently available locations: ${available_locations}."
	fi
	fail "Hetzner server-type '${MY_SERVER_TYPE}' is not supported in location '${MY_LOCATION}'. Supported locations: ${supported_locations}."
}

github_api() {
	local method="$1"
	local endpoint="$2"
	local output_file="$3"
	local data_file="${4:-}"
	local http_code
	local curl_args=(
		-sS
		-L
		-X "$method"
		-o "$output_file"
		-w "%{http_code}"
		-H "Accept: application/vnd.github+json"
		-H "Authorization: Bearer ${MY_GITHUB_TOKEN}"
		-H "X-GitHub-Api-Version: 2022-11-28"
	)

	if [[ -n "$data_file" ]]; then
		curl_args+=(-H "Content-Type: application/json" --data @"$data_file")
	fi

	http_code="$(curl "${curl_args[@]}" "https://api.github.com${endpoint}")"
	echo "$http_code"
}

request_runner_registration_token_via_broker() {
	local output_file="$1"
	local request_file="${RUNNER_STATE_DIR}/registration-token-request.json"
	local access_token
	local http_code

	require_command az
	[[ -n "$MY_FUNCTION_APP_NAME" ]] || fail "function-app-name is required to create a runner registration token via the broker."
	[[ -n "$MY_AZURE_CLIENT_ID" ]] || fail "azure-client-id is required to create a runner registration token via the broker."

	jq -n --arg repo "$MY_GITHUB_REPOSITORY" '{repo: $repo}' > "$request_file"

	access_token="$(
		az account get-access-token \
			--resource "api://${MY_AZURE_CLIENT_ID}" \
			--query accessToken \
			--output tsv
	)" || fail "Failed to obtain an Azure access token for the runner broker."

	http_code="$(
		curl \
			-sS \
			-L \
			-X POST \
			-o "$output_file" \
			-w "%{http_code}" \
			-H "Authorization: Bearer ${access_token}" \
			-H "Content-Type: application/json" \
			--data @"$request_file" \
			"https://${MY_FUNCTION_APP_NAME}.azurewebsites.net/api/request_runner_registration_token"
	)"

	echo "$http_code"
}

create_runner_registration_token() {
	local output_file="$1"

	if [[ -n "$MY_FUNCTION_APP_NAME" || -n "$MY_AZURE_CLIENT_ID" ]]; then
		request_runner_registration_token_via_broker "$output_file"
		return
	fi

	[[ -n "$MY_GITHUB_TOKEN" ]] || fail "Either runner-registration-token, function-app-name with azure-client-id, or github-token is required in create mode."
	github_api "POST" "/repos/${MY_GITHUB_REPOSITORY}/actions/runners/registration-token" "$output_file"
}

hetzner_api() {
	local method="$1"
	local endpoint="$2"
	local output_file="$3"
	local data_file="${4:-}"
	local http_code
	local curl_args=(
		-sS
		-X "$method"
		-o "$output_file"
		-w "%{http_code}"
		-H "Content-Type: application/json"
		-H "Authorization: Bearer ${MY_HCLOUD_TOKEN}"
	)

	if [[ -n "$data_file" ]]; then
		curl_args+=(--data @"$data_file")
	fi

	http_code="$(curl "${curl_args[@]}" "https://api.hetzner.cloud/v1${endpoint}")"
	echo "$http_code"
}

delete_server_best_effort() {
	local server_id="$1"
	local body_file="${RUNNER_STATE_DIR}/delete-server.json"
	local http_code

	http_code="$(hetzner_api "DELETE" "/servers/${server_id}" "$body_file")"
	if [[ "$http_code" == "204" || "$http_code" == "404" ]]; then
		return 0
	fi

	echo "Hetzner delete response (${http_code}):" >&2
	cat "$body_file" >&2 || true
	return 1
}

cleanup_tmp() {
	local exit_code=$?
	if [[ $exit_code -ne 0 && -n "${CREATED_SERVER_ID}" ]]; then
		echo "Create flow failed after server '${CREATED_SERVER_ID}' was created. Attempting best-effort cleanup..." >&2
		delete_server_best_effort "${CREATED_SERVER_ID}" || true
	fi
	rm -rf "$RUNNER_STATE_DIR"
	return "$exit_code"
}

for cmd in base64 curl jq mktemp python3; do
	require_command "$cmd"
done

SCRIPT_DIR="$(
	cd "$(dirname "$0")"
	pwd -P
)"
INSTALL_SCRIPT="${SCRIPT_DIR}/install.sh"
[[ -f "$INSTALL_SCRIPT" ]] || fail "install.sh was not found."

RUNNER_STATE_DIR="$(mktemp -d)"
trap cleanup_tmp EXIT

MY_MODE="${INPUT_MODE:-}"
[[ "$MY_MODE" == "create" || "$MY_MODE" == "delete" ]] || fail "mode must be 'create' or 'delete'."

MY_AZURE_CLIENT_ID="${INPUT_AZURE_CLIENT_ID:-}"
MY_FUNCTION_APP_NAME="${INPUT_FUNCTION_APP_NAME:-}"
MY_GITHUB_TOKEN="${INPUT_GITHUB_TOKEN:-}"
MY_GITHUB_RUNNER_REGISTRATION_TOKEN="${INPUT_RUNNER_REGISTRATION_TOKEN:-}"

MY_HCLOUD_TOKEN="${INPUT_HCLOUD_TOKEN:-}"
[[ -n "$MY_HCLOUD_TOKEN" ]] || fail "hcloud-token is required."

MY_IMAGE="${INPUT_IMAGE:-ubuntu-24.04}"
[[ "$MY_IMAGE" =~ ^[a-zA-Z0-9._-]{1,63}$ ]] || fail "Invalid image '${MY_IMAGE}'."

MY_LOCATION="${INPUT_LOCATION:-nbg1}"
MY_LOCATION="${MY_LOCATION,,}"
[[ "$MY_LOCATION" =~ ^[a-zA-Z0-9._-]{1,32}$ ]] || fail "Invalid location '${MY_LOCATION}'."

MY_NAME="${INPUT_NAME:-gh-runner-$RANDOM}"
[[ "$MY_NAME" =~ ^[a-zA-Z0-9_-]{1,64}$ ]] || fail "Invalid runner name '${MY_NAME}'."

MY_RUNNER_DIR="${INPUT_RUNNER_DIR:-/actions-runner}"
[[ "$MY_RUNNER_DIR" =~ ^/([^/]+/)*[^/]+$ ]] || fail "runner-dir must be an absolute path without a trailing slash."

MY_RUNNER_COUNT="${INPUT_RUNNER_COUNT:-1}"
require_positive_integer "$MY_RUNNER_COUNT" "runner-count"

MY_RUNNER_VERSION="${INPUT_RUNNER_VERSION:-latest}"
if [[ "$MY_RUNNER_VERSION" != "latest" && ! "$MY_RUNNER_VERSION" =~ ^[0-9.]+$ ]]; then
	fail "runner-version must be 'latest' or a version number without the 'v' prefix."
fi

MY_SERVER_TYPE="${INPUT_SERVER_TYPE:-cpx32}"
MY_SERVER_TYPE="${MY_SERVER_TYPE,,}"
[[ "$MY_SERVER_TYPE" =~ ^[a-zA-Z0-9]+$ ]] || fail "Invalid server-type '${MY_SERVER_TYPE}'."

MY_SERVER_ID="${INPUT_SERVER_ID:-}"
MY_PRE_RUNNER_SCRIPT="${INPUT_PRE_RUNNER_SCRIPT:-}"

MY_CREATE_WAIT="${INPUT_CREATE_WAIT:-36}"
MY_DELETE_WAIT="${INPUT_DELETE_WAIT:-18}"
MY_RUNNER_WAIT="${INPUT_RUNNER_WAIT:-36}"
MY_SERVER_WAIT="${INPUT_SERVER_WAIT:-30}"
require_integer "$MY_CREATE_WAIT" "create-wait"
require_integer "$MY_DELETE_WAIT" "delete-wait"
require_integer "$MY_RUNNER_WAIT" "runner-wait"
require_integer "$MY_SERVER_WAIT" "server-wait"

MY_ENABLE_IPV4="${INPUT_ENABLE_IPV4:-true}"
MY_ENABLE_IPV6="${INPUT_ENABLE_IPV6:-false}"
require_boolean "$MY_ENABLE_IPV4" "enable-ipv4"
require_boolean "$MY_ENABLE_IPV6" "enable-ipv6"

MY_GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-}"
[[ -n "$MY_GITHUB_REPOSITORY" ]] || fail "GITHUB_REPOSITORY is required."

MY_GITHUB_REPOSITORY_ID="${GITHUB_REPOSITORY_ID:-0}"
MY_GITHUB_RUN_ID="${GITHUB_RUN_ID:-0}"
MY_GITHUB_RUN_ATTEMPT="${GITHUB_RUN_ATTEMPT:-0}"

export MY_ENABLE_IPV4
export MY_ENABLE_IPV6
export MY_GITHUB_REPOSITORY_ID
export MY_GITHUB_RUN_ATTEMPT
export MY_GITHUB_RUN_ID
export MY_IMAGE
export MY_LOCATION
export MY_NAME
export MY_SERVER_TYPE

if [[ "$MY_MODE" == "delete" ]]; then
	[[ -n "$MY_SERVER_ID" ]] || fail "server-id is required in delete mode."
	require_integer "$MY_SERVER_ID" "server-id"

	echo "Deleting runner host '${MY_NAME}' and server '${MY_SERVER_ID}'..."

	if [[ -n "$MY_GITHUB_TOKEN" ]]; then
		runner_list_file="${RUNNER_STATE_DIR}/github-runners.json"
		http_code="$(github_api "GET" "/repos/${MY_GITHUB_REPOSITORY}/actions/runners" "$runner_list_file")"
		if [[ "$http_code" == "200" ]]; then
			runner_ids="$(
				jq -r --arg name "$MY_NAME" '
					.runners[]?
					| select(.name == $name or (.name | startswith($name + "-slot-")))
					| .id
				' "$runner_list_file"
			)"

			while IFS= read -r runner_id; do
				[[ -n "$runner_id" && "$runner_id" != "null" ]] || continue
				delete_runner_file="${RUNNER_STATE_DIR}/delete-runner.json"
				http_code="$(github_api "DELETE" "/repos/${MY_GITHUB_REPOSITORY}/actions/runners/${runner_id}" "$delete_runner_file")"
				if [[ "$http_code" != "204" && "$http_code" != "404" ]]; then
					echo "GitHub delete runner response (${http_code}):" >&2
					cat "$delete_runner_file" >&2 || true
					echo "Warning: failed to delete GitHub runner id '${runner_id}' for host '${MY_NAME}'. Continuing with server cleanup." >&2
				fi
			done <<< "$runner_ids"
		else
			echo "GitHub list runners response (${http_code}):" >&2
			cat "$runner_list_file" >&2 || true
			echo "Warning: failed to list GitHub runners. Continuing with server cleanup." >&2
		fi
	else
		echo "GitHub token not provided; skipping GitHub runner deregistration and continuing with server cleanup." >&2
	fi

	delete_attempt=1
	while (( delete_attempt <= MY_DELETE_WAIT )); do
		if delete_server_best_effort "$MY_SERVER_ID"; then
			echo "Deleted Hetzner server '${MY_SERVER_ID}'."
			exit 0
		fi

		if (( delete_attempt == MY_DELETE_WAIT )); then
			fail "Failed to delete Hetzner server '${MY_SERVER_ID}'."
		fi

		echo "Server delete attempt ${delete_attempt}/${MY_DELETE_WAIT} failed. Retrying in ${WAIT_SECONDS}s..."
		sleep "$WAIT_SECONDS"
		delete_attempt=$((delete_attempt + 1))
	done
fi

declare -a MY_RUNNER_NAMES=()
declare -a MY_RUNNER_TOKENS=()
for (( slot = 1; slot <= MY_RUNNER_COUNT; slot++ )); do
	MY_RUNNER_NAMES+=("${MY_NAME}-slot-${slot}")
done

if [[ -n "$MY_GITHUB_RUNNER_REGISTRATION_TOKEN" ]]; then
	if (( MY_RUNNER_COUNT > 1 )); then
		echo "A single runner-registration-token was provided for ${MY_RUNNER_COUNT} runners; reusing it for every slot." >&2
	fi
	for (( slot = 1; slot <= MY_RUNNER_COUNT; slot++ )); do
		MY_RUNNER_TOKENS+=("${MY_GITHUB_RUNNER_REGISTRATION_TOKEN}")
	done
else
	for (( slot = 1; slot <= MY_RUNNER_COUNT; slot++ )); do
		registration_file="${RUNNER_STATE_DIR}/registration-token-${slot}.json"
		http_code="$(create_runner_registration_token "$registration_file")"
		if [[ "$http_code" != "201" ]]; then
			if [[ -n "$MY_FUNCTION_APP_NAME" || -n "$MY_AZURE_CLIENT_ID" ]]; then
				echo "Runner broker registration token response (${http_code}):" >&2
			else
				echo "GitHub registration token response (${http_code}):" >&2
			fi
			cat "$registration_file" >&2 || true
			fail "Failed to create GitHub runner registration token for slot ${slot}."
		fi

		runner_token="$(jq -r '.token' "$registration_file")"
		[[ -n "$runner_token" && "$runner_token" != "null" ]] || fail "Runner registration token response did not include a token for slot ${slot}."
		MY_RUNNER_TOKENS+=("${runner_token}")
	done
fi

for runner_token in "${MY_RUNNER_TOKENS[@]}"; do
	echo "::add-mask::${runner_token}"
done

MY_INSTALL_SH_BASE64="$(base64 --wrap=0 < "$INSTALL_SCRIPT")"
MY_PRE_RUNNER_SCRIPT_BASE64="$(printf '%s' "$MY_PRE_RUNNER_SCRIPT" | base64 --wrap=0)"
RUNNER_BOOTSTRAP_FILE="${RUNNER_STATE_DIR}/runner-bootstrap.sh"
{
	echo "#!/usr/bin/env bash"
	echo "set -euo pipefail"
	printf 'RUNNER_VERSION=%q\n' "$MY_RUNNER_VERSION"
	printf 'RUNNER_DIR_BASE=%q\n' "$MY_RUNNER_DIR"
	printf 'RUNNER_COUNT=%q\n' "$MY_RUNNER_COUNT"
	printf 'GITHUB_REPOSITORY=%q\n' "$MY_GITHUB_REPOSITORY"
	printf 'SHARED_LABEL=%q\n' "$MY_NAME"
	printf 'RUNNER_NAMES=('
	for runner_name in "${MY_RUNNER_NAMES[@]}"; do
		printf '%q ' "$runner_name"
	done
	echo ")"
	printf 'RUNNER_TOKENS=('
	for runner_token in "${MY_RUNNER_TOKENS[@]}"; do
		printf '%q ' "$runner_token"
	done
	echo ")"
	cat <<'EOF'
runner_dir_for_slot() {
	local slot="$1"
	printf '%s-%s' "$RUNNER_DIR_BASE" "$slot"
}

if getent group kvm >/dev/null; then
	usermod -aG kvm github-runner
fi
if getent group libvirt >/dev/null; then
	usermod -aG libvirt github-runner
fi

for (( slot = 1; slot <= RUNNER_COUNT; slot++ )); do
	slot_index=$((slot - 1))
	slot_dir="$(runner_dir_for_slot "$slot")"
	slot_name="${RUNNER_NAMES[$slot_index]}"
	slot_token="${RUNNER_TOKENS[$slot_index]}"
	slot_labels="hetzner,${SHARED_LABEL},${slot_name}"

	bash "${RUNNER_DIR_BASE}/install.sh" -v "${RUNNER_VERSION}" -d "${slot_dir}"
	chown -R github-runner:github-runner "${slot_dir}"

	sudo -u github-runner \
		env GITHUB_REPOSITORY="${GITHUB_REPOSITORY}" RUNNER_DIR="${slot_dir}" RUNNER_LABELS="${slot_labels}" RUNNER_NAME="${slot_name}" RUNNER_TOKEN="${slot_token}" \
		bash -lc 'cd "$RUNNER_DIR" && ./config.sh --url "https://github.com/${GITHUB_REPOSITORY}" --token "$RUNNER_TOKEN" --name "$RUNNER_NAME" --labels "$RUNNER_LABELS" --ephemeral --disableupdate --unattended'
	sudo -u github-runner env RUNNER_DIR="${slot_dir}" bash -lc 'cd "$RUNNER_DIR" && nohup ./run.sh > runner.log 2>&1 &'
done
EOF
} > "${RUNNER_BOOTSTRAP_FILE}"
RUNNER_BOOTSTRAP_BASE64="$(base64 --wrap=0 < "${RUNNER_BOOTSTRAP_FILE}")"

cloud_init_file="${RUNNER_STATE_DIR}/cloud-init.yml"
cat >"$cloud_init_file" <<EOF
#cloud-config
package_update: true
package_upgrade: false
users:
  - default
  - name: github-runner
    shell: /bin/bash
    groups:
      - sudo
    sudo:
      - ALL=(ALL) NOPASSWD:ALL
packages:
  - ca-certificates
  - curl
  - git
  - gzip
  - jq
  - sudo
  - tar
write_files:
  - path: ${MY_RUNNER_DIR}/install.sh
    permissions: "0755"
    encoding: b64
    content: ${MY_INSTALL_SH_BASE64}
  - path: ${MY_RUNNER_DIR}/pre-runner-script.sh
    permissions: "0700"
    encoding: b64
    content: ${MY_PRE_RUNNER_SCRIPT_BASE64}
  - path: ${MY_RUNNER_DIR}/runner-bootstrap.sh
    permissions: "0755"
    encoding: b64
    content: ${RUNNER_BOOTSTRAP_BASE64}
runcmd:
  - mkdir -p ${MY_RUNNER_DIR}
  - chmod 0755 ${MY_RUNNER_DIR}
  - bash ${MY_RUNNER_DIR}/pre-runner-script.sh
  - bash ${MY_RUNNER_DIR}/runner-bootstrap.sh
EOF

create_server_file="${RUNNER_STATE_DIR}/create-server.json"
validate_server_type_location
python3 - "$cloud_init_file" "$create_server_file" <<'PY'
import json
import os
import pathlib
import sys

cloud_init_path = pathlib.Path(sys.argv[1])
output_path = pathlib.Path(sys.argv[2])
payload = {
    "name": os.environ["MY_NAME"],
    "location": os.environ["MY_LOCATION"],
    "server_type": os.environ["MY_SERVER_TYPE"],
    "start_after_create": True,
    "image": os.environ["MY_IMAGE"],
    "labels": {
        "type": "github-runner",
        "gh-repo-id": os.environ.get("MY_GITHUB_REPOSITORY_ID", "0"),
        "gh-run-id": os.environ.get("MY_GITHUB_RUN_ID", "0"),
        "gh-run-attempt": os.environ.get("MY_GITHUB_RUN_ATTEMPT", "0"),
    },
    "public_net": {
        "enable_ipv4": os.environ["MY_ENABLE_IPV4"] == "true",
        "enable_ipv6": os.environ["MY_ENABLE_IPV6"] == "true",
    },
    "user_data": cloud_init_path.read_text(),
}
output_path.write_text(json.dumps(payload))
PY

create_attempt=1
create_response_file="${RUNNER_STATE_DIR}/create-server-response.json"
while (( create_attempt <= MY_CREATE_WAIT )); do
	http_code="$(hetzner_api "POST" "/servers" "$create_response_file" "$create_server_file")"
	if [[ "$http_code" == "201" ]]; then
		break
	fi

	if jq -e '.error.code | select(. == "resource_unavailable" or . == "resource_limit_exceeded")' "$create_response_file" >/dev/null 2>&1; then
		if (( create_attempt == MY_CREATE_WAIT )); then
			cat "$create_response_file" >&2 || true
			fail "Hetzner capacity was unavailable after ${MY_CREATE_WAIT} attempts."
		fi
		echo "Hetzner capacity unavailable. Retrying in ${WAIT_SECONDS}s (${create_attempt}/${MY_CREATE_WAIT})..."
		sleep "$WAIT_SECONDS"
		create_attempt=$((create_attempt + 1))
		continue
	fi

	echo "Hetzner create server response (${http_code}):" >&2
	cat "$create_response_file" >&2 || true
	fail "Failed to create Hetzner server."
done

server_id="$(jq -r '.server.id' "$create_response_file")"
[[ "$server_id" =~ ^[0-9]+$ ]] || fail "Hetzner create response did not include a server ID."
CREATED_SERVER_ID="$server_id"

echo "Created Hetzner server '${server_id}' for runner '${MY_NAME}'."

server_status_file="${RUNNER_STATE_DIR}/server-status.json"
server_attempt=1
while (( server_attempt <= MY_SERVER_WAIT )); do
	http_code="$(hetzner_api "GET" "/servers/${server_id}" "$server_status_file")"
	if [[ "$http_code" != "200" ]]; then
		echo "Hetzner server status response (${http_code}):" >&2
		cat "$server_status_file" >&2 || true
		fail "Failed to read Hetzner server status."
	fi

	server_status="$(jq -r '.server.status' "$server_status_file")"
	if [[ "$server_status" == "running" ]]; then
		break
	fi

	if (( server_attempt == MY_SERVER_WAIT )); then
		delete_server_best_effort "$server_id" || true
		fail "Hetzner server '${server_id}' did not reach the running state."
	fi

	echo "Server '${server_id}' status is '${server_status}'. Waiting ${WAIT_SECONDS}s..."
	sleep "$WAIT_SECONDS"
	server_attempt=$((server_attempt + 1))
done

runner_attempt=1
runner_list_file="${RUNNER_STATE_DIR}/github-runners.json"
if [[ -n "$MY_GITHUB_TOKEN" ]]; then
	while (( runner_attempt <= MY_RUNNER_WAIT )); do
		http_code="$(github_api "GET" "/repos/${MY_GITHUB_REPOSITORY}/actions/runners" "$runner_list_file")"
		if [[ "$http_code" != "200" ]]; then
			echo "GitHub list runners response (${http_code}):" >&2
			cat "$runner_list_file" >&2 || true
			delete_server_best_effort "$server_id" || true
			fail "Failed to list GitHub runners."
		fi

		ready_runner_count="$(
			jq -r --arg name "$MY_NAME" '
				[
					.runners[]?
					| select((.name == $name or (.name | startswith($name + "-slot-"))) and (.status == "online" or .status == "busy"))
				] | length
			' "$runner_list_file"
		)"

		if (( ready_runner_count >= MY_RUNNER_COUNT )); then
			break
		fi

		if (( runner_attempt == MY_RUNNER_WAIT )); then
			delete_server_best_effort "$server_id" || true
			fail "Runner host '${MY_NAME}' did not register ${MY_RUNNER_COUNT} runner slot(s) with GitHub in time."
		fi

		echo "Runner host '${MY_NAME}' has ${ready_runner_count}/${MY_RUNNER_COUNT} runner slot(s) online. Waiting ${WAIT_SECONDS}s..."
		sleep "$WAIT_SECONDS"
		runner_attempt=$((runner_attempt + 1))
	done
else
	echo "GitHub token not provided; skipping GitHub runner registration polling after server boot."
fi

echo "label=${MY_NAME}" >> "${GITHUB_OUTPUT}"
echo "server_id=${server_id}" >> "${GITHUB_OUTPUT}"
CREATED_SERVER_ID=""

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
	{
		echo "Hetzner runner ready"
		echo
		echo "- Shared runner label: \`${MY_NAME}\`"
		echo "- Runner slots: \`${MY_RUNNER_COUNT}\`"
		echo "- Server ID: \`${server_id}\`"
		echo "- Server type: \`${MY_SERVER_TYPE}\`"
		echo "- Location: \`${MY_LOCATION}\`"
	} >> "${GITHUB_STEP_SUMMARY}"
fi
