#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_UNDER_TEST="${SCRIPT_DIR}/create-macos-tart-runner.sh"
ORIGINAL_PATH="${PATH}"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/macos-tart-runner-tests.XXXXXX")"
FAKE_BIN="${TEST_ROOT}/fake-bin"
TEST_LOG_DIR=""
CURRENT_PARENT_PID=""

cleanup() {
	if [[ -n "${CURRENT_PARENT_PID}" ]]; then
		kill "${CURRENT_PARENT_PID}" 2>/dev/null || true
	fi

	if [[ -d "${TEST_ROOT}" ]]; then
		while IFS= read -r pid_file; do
			kill "$(<"${pid_file}")" 2>/dev/null || true
		done < <(find "${TEST_ROOT}" -name remote_pid -type f 2>/dev/null || true)
		rm -rf "${TEST_ROOT}"
	fi
}
trap cleanup EXIT

fail() {
	printf 'not ok - %s\n' "$*" >&2
	exit 1
}

assert_contains() {
	local file="$1"
	local expected="$2"

	if ! grep -Fq -- "$expected" "$file"; then
		printf 'Expected to find:\n%s\n\nin %s:\n' "$expected" "$file" >&2
		sed -n '1,220p' "$file" >&2 || true
		fail "missing expected text"
	fi
}

assert_not_contains() {
	local file="$1"
	local unexpected="$2"

	if grep -Fq -- "$unexpected" "$file"; then
		printf 'Did not expect to find:\n%s\n\nin %s:\n' "$unexpected" "$file" >&2
		sed -n '1,220p' "$file" >&2 || true
		fail "unexpected text found"
	fi
}

assert_count() {
	local file="$1"
	local expected_text="$2"
	local expected_count="$3"
	local actual_count

	actual_count=$(grep -F -c -- "$expected_text" "$file" || true)
	if [[ "$actual_count" != "$expected_count" ]]; then
		printf 'Expected %s occurrence(s) of %s in %s, got %s:\n' \
			"$expected_count" "$expected_text" "$file" "$actual_count" >&2
		sed -n '1,220p' "$file" >&2 || true
		fail "unexpected occurrence count"
	fi
}

wait_for_file() {
	local file="$1"
	local attempts=0

	while [[ ! -e "$file" ]]; do
		attempts=$((attempts + 1))
		if [[ "$attempts" -gt 100 ]]; then
			fail "timed out waiting for ${file}"
		fi
		/bin/sleep 0.05
	done
}

write_fake_commands() {
	mkdir -p "${FAKE_BIN}"

	cat >"${FAKE_BIN}/gh" <<'EOF_GH'
#!/usr/bin/env bash
set -euo pipefail

log_dir="${TEST_LOG_DIR:?TEST_LOG_DIR is required}"
mkdir -p "${log_dir}"
{
	printf 'gh'
	for arg in "$@"; do
		printf '\t%s' "$arg"
	done
	printf '\n'
} >>"${log_dir}/gh.log"

if [[ "${1:-}" != "api" ]]; then
	printf 'fake gh only supports gh api\n' >&2
	exit 1
fi
shift

method="GET"
path=""
expect_method=false
for arg in "$@"; do
	if [[ "${expect_method}" == "true" ]]; then
		method="$arg"
		expect_method=false
		continue
	fi
	if [[ "$arg" == "--method" ]]; then
		expect_method=true
		continue
	fi
	if [[ "$arg" == /* ]]; then
		path="$arg"
	fi
done

if [[ "$path" == *"/registration-token" ]]; then
	printf '%s\n' "${FAKE_GH_REGISTRATION_TOKEN:-test-token}"
	exit 0
fi

if [[ "$method" == "GET" && "$path" == /repos/*/actions/runs*status=in_progress* ]]; then
	if [[ "${FAKE_GH_RUN_LIST_FAIL:-false}" == "true" ]]; then
		printf 'run list failed\n' >&2
		exit "${FAKE_GH_RUN_LIST_EXIT:-22}"
	fi
	if [[ -n "${FAKE_GH_RUN_IDS:-}" ]]; then
		printf '%s\n' "${FAKE_GH_RUN_IDS}"
	fi
	exit 0
fi

if [[ "$method" == "GET" && "$path" == /repos/*/actions/runs/*/jobs* ]]; then
	if [[ "${FAKE_GH_JOB_LIST_FAIL:-false}" == "true" ]]; then
		printf 'job list failed\n' >&2
		exit "${FAKE_GH_JOB_LIST_EXIT:-22}"
	fi

	line="${FAKE_GH_JOB_LINE:-}"
	queue="${FAKE_GH_JOB_LINE_QUEUE:-}"
	if [[ -n "$queue" && -f "$queue" ]]; then
		if IFS= read -r line <"$queue"; then
			tmp="${queue}.$$"
			tail -n +2 "$queue" >"$tmp" || true
			mv "$tmp" "$queue"
		else
			line=""
		fi
	fi

	case "$line" in
		__EMPTY__|"")
			;;
		*)
			printf '%s\n' "$line"
			;;
	esac
	exit 0
fi

if [[ "$method" == "POST" && "$path" == */actions/runs/*/cancel ]]; then
	exit "${FAKE_GH_CANCEL_EXIT:-0}"
fi

if [[ "$method" == "POST" && "$path" == */actions/runs/*/force-cancel ]]; then
	force_cancel_exit="${FAKE_GH_FORCE_CANCEL_EXIT:-0}"
	if [[ "$force_cancel_exit" != "0" ]]; then
		printf '%s\n' "${FAKE_GH_FORCE_CANCEL_ERROR:-force cancel failed}" >&2
	fi
	exit "$force_cancel_exit"
fi

if [[ "$method" == "GET" && "$path" == */actions/runners ]]; then
	if [[ "${FAKE_GH_RUNNER_LIST_FAIL:-false}" == "true" ]]; then
		printf 'runner list failed\n' >&2
		exit "${FAKE_GH_RUNNER_LIST_EXIT:-22}"
	fi

	line="${FAKE_GH_RUNNER_INFO:-}"
	queue="${FAKE_GH_RUNNER_INFO_QUEUE:-}"
	if [[ -n "$queue" && -f "$queue" ]]; then
		if IFS= read -r line <"$queue"; then
			tmp="${queue}.$$"
			tail -n +2 "$queue" >"$tmp" || true
			mv "$tmp" "$queue"
		else
			line=""
		fi
	fi

	case "$line" in
		__FAIL__:*)
			printf '%s\n' "${line#__FAIL__:}" >&2
			exit "${FAKE_GH_RUNNER_LIST_EXIT:-22}"
			;;
		__EMPTY__|"")
			exit 0
			;;
		*)
			printf '%s\n' "$line"
			exit 0
			;;
	esac
fi

if [[ "$method" == "DELETE" && "$path" == */actions/runners/* ]]; then
	exit "${FAKE_GH_DELETE_EXIT:-0}"
fi

printf 'unexpected gh api call: method=%s path=%s args=%s\n' "$method" "$path" "$*" >&2
exit 1
EOF_GH

	cat >"${FAKE_BIN}/tart" <<'EOF_TART'
#!/usr/bin/env bash
set -euo pipefail

log_dir="${TEST_LOG_DIR:?TEST_LOG_DIR is required}"
mkdir -p "${log_dir}"
{
	printf 'tart'
	for arg in "$@"; do
		printf '\t%s' "$arg"
	done
	printf '\n'
} >>"${log_dir}/tart.log"

case "${1:-}" in
	list)
		printf 'SOURCE\tNAME\n'
		printf 'local\t%s\n' "${VM_BASE_IMAGE:-tahoe-runner}"
		;;
	clone|set|stop|delete)
		exit 0
		;;
	run)
		exit 0
		;;
	ip)
		printf '%s\n' "${FAKE_TART_IP:-127.0.0.1}"
		;;
	*)
		printf 'unexpected tart call: %s\n' "$*" >&2
		exit 1
		;;
esac
EOF_TART

	cat >"${FAKE_BIN}/sshpass" <<'EOF_SSHPASS'
#!/usr/bin/env bash
set -euo pipefail

log_dir="${TEST_LOG_DIR:?TEST_LOG_DIR is required}"
mkdir -p "${log_dir}"
{
	printf 'sshpass'
	for arg in "$@"; do
		printf '\t%s' "$arg"
	done
	printf '\n'
} >>"${log_dir}/sshpass.log"

args=" $* "
if [[ "$args" == *" true"* ]]; then
	exit 0
fi

if [[ "$args" == *" bash -s -- "* ]]; then
	cat >/dev/null || true
	printf 'signal-runner\n' >>"${log_dir}/sshpass.log"
	exit 0
fi

if [[ "$args" == *" bash"* ]]; then
	cat >/dev/null || true
	touch "${log_dir}/remote_started"
	printf '%s\n' "$$" >"${log_dir}/remote_pid"
	if [[ "${FAKE_SSH_RUNNER_BLOCK:-false}" == "true" ]]; then
		trap 'printf "remote-term\n" >>"${log_dir}/sshpass.log"; exit 143' TERM INT
		while true; do
			/bin/sleep 1
		done
	fi
	exit "${FAKE_SSH_RUNNER_EXIT:-0}"
fi

exit 0
EOF_SSHPASS

	cat >"${FAKE_BIN}/date" <<'EOF_DATE'
#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" == "+%s" ]]; then
	sequence="${FAKE_DATE_SEQUENCE:-}"
	if [[ -n "$sequence" && -f "$sequence" ]]; then
		if IFS= read -r first <"$sequence"; then
			tmp="${sequence}.$$"
			tail -n +2 "$sequence" >"$tmp" || true
			mv "$tmp" "$sequence"
			printf '%s\n' "$first"
			exit 0
		fi
	fi
	if [[ -n "${FAKE_DATE_VALUE:-}" ]]; then
		printf '%s\n' "${FAKE_DATE_VALUE}"
		exit 0
	fi
fi

/bin/date "$@"
EOF_DATE

	cat >"${FAKE_BIN}/sleep" <<'EOF_SLEEP'
#!/usr/bin/env bash
set -euo pipefail

log_dir="${TEST_LOG_DIR:-}"
if [[ -n "$log_dir" ]]; then
	mkdir -p "$log_dir"
	{
		printf 'sleep'
		for arg in "$@"; do
			printf '\t%s' "$arg"
		done
		printf '\n'
	} >>"${log_dir}/sleep.log"
fi
exit 0
EOF_SLEEP

	chmod +x "${FAKE_BIN}/gh" "${FAKE_BIN}/tart" "${FAKE_BIN}/sshpass" "${FAKE_BIN}/date" "${FAKE_BIN}/sleep"
}

reset_script_state() {
	GITHUB_SCOPE="repo"
	GITHUB_REPO="owner/repo"
	GITHUB_ORG="octo-org"
	RUNNER_NAME_BASE="test-runner"
	VM_BASE_IMAGE="tahoe-runner"
	VM_CLONE_PER_RUN="true"
	VM_SSH_USER="admin"
	VM_SSH_PASS="admin"
	RUNNER_DIR="/Users/admin/actions-runner"
	RUNNER_POOL_SIZE="1"
	MAX_RUNS="0"
	GITHUB_CANCEL_RUN_ON_SHUTDOWN="true"
	GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN="false"
	RUNNER_SHUTDOWN_GRACE_SECONDS="0"
	RUNNER_FORCE_CANCEL_AFTER_SECONDS="30"
	RUNNER_SHUTDOWN_POLL_SECONDS="0"
	REQUESTED_CANCEL_RUN_ID=""
	export GITHUB_SCOPE GITHUB_REPO GITHUB_ORG VM_BASE_IMAGE VM_CLONE_PER_RUN
}

begin_test() {
	local name="$1"

	TEST_LOG_DIR="${TEST_ROOT}/${name}"
	mkdir -p "${TEST_LOG_DIR}"
	: >"${TEST_LOG_DIR}/gh.log"
	: >"${TEST_LOG_DIR}/tart.log"
	: >"${TEST_LOG_DIR}/sshpass.log"
	: >"${TEST_LOG_DIR}/sleep.log"
	export TEST_LOG_DIR

	unset FAKE_GH_RUN_IDS FAKE_GH_JOB_LINE FAKE_GH_JOB_LINE_QUEUE FAKE_GH_RUNNER_INFO FAKE_GH_RUNNER_INFO_QUEUE
	unset FAKE_GH_RUN_LIST_FAIL FAKE_GH_JOB_LIST_FAIL FAKE_GH_RUNNER_LIST_FAIL
	unset FAKE_GH_CANCEL_EXIT FAKE_GH_FORCE_CANCEL_EXIT FAKE_GH_FORCE_CANCEL_ERROR FAKE_GH_DELETE_EXIT
	unset FAKE_DATE_SEQUENCE FAKE_DATE_VALUE FAKE_SSH_RUNNER_BLOCK FAKE_SSH_RUNNER_EXIT
	unset FAKE_TART_IP

	reset_script_state
}

set_date_sequence() {
	local sequence_file="${TEST_LOG_DIR}/date-sequence"

	: >"${sequence_file}"
	local value
	for value in "$@"; do
		printf '%s\n' "$value" >>"${sequence_file}"
	done
	export FAKE_DATE_SEQUENCE="${sequence_file}"
}

set_job_line_queue() {
	local queue_file="${TEST_LOG_DIR}/job-line-queue"

	: >"${queue_file}"
	local value
	for value in "$@"; do
		printf '%s\n' "$value" >>"${queue_file}"
	done
	export FAKE_GH_JOB_LINE_QUEUE="${queue_file}"
}

set_runner_info_queue() {
	local queue_file="${TEST_LOG_DIR}/runner-info-queue"

	: >"${queue_file}"
	local value
	for value in "$@"; do
		printf '%s\n' "$value" >>"${queue_file}"
	done
	export FAKE_GH_RUNNER_INFO_QUEUE="${queue_file}"
}

run_test() {
	local name="$1"

	printf 'running %s\n' "$name"
	"$name"
	printf 'ok - %s\n' "$name"
}

write_fake_commands
PATH="${FAKE_BIN}:${ORIGINAL_PATH}"
export PATH
RUNNER_NAME_BASE="test-runner"
GITHUB_REPO="owner/repo"
VM_BASE_IMAGE="tahoe-runner"

# shellcheck source=.github/runners/create-macos-tart-runner.sh
source "${SCRIPT_UNDER_TEST}"

test_successful_cancel_signals_runner_and_stops_vm() {
	begin_test "successful-cancel"
	export FAKE_GH_RUN_IDS="9001"
	export FAKE_GH_JOB_LINE=$'9001\t501\tbuild\t"https://example.test/job"'
	export FAKE_GH_RUNNER_INFO=$'123\tfalse\tonline'

	cleanup_vm "vm-1" "runner-1" "10.0.0.2" "true" >"${TEST_LOG_DIR}/output" 2>&1

	assert_contains "${TEST_LOG_DIR}/output" "[vm-1] Canceling workflow run 9001"
	assert_contains "${TEST_LOG_DIR}/output" "[vm-1] Sending SIGINT to runner process inside VM"
	assert_contains "${TEST_LOG_DIR}/output" "Runner 'runner-1' is no longer busy"
	assert_contains "${TEST_LOG_DIR}/gh.log" "/repos/owner/repo/actions/runs/9001/cancel"
	assert_contains "${TEST_LOG_DIR}/sshpass.log" "signal-runner"
	assert_contains "${TEST_LOG_DIR}/tart.log" $'tart\tstop\tvm-1'
	assert_contains "${TEST_LOG_DIR}/tart.log" $'tart\tdelete\tvm-1'
}

test_runner_list_api_failure_remains_unknown() {
	begin_test "runner-list-api-failure"
	RUNNER_SHUTDOWN_GRACE_SECONDS="1"
	export FAKE_GH_RUNNER_LIST_FAIL="true"
	set_date_sequence 100 101

	if wait_for_runner_to_settle "vm-1" "runner-1" "" >"${TEST_LOG_DIR}/output" 2>&1; then
		fail "wait_for_runner_to_settle unexpectedly succeeded"
	fi

	assert_contains "${TEST_LOG_DIR}/output" "Runner 'runner-1' state is unknown"
	assert_contains "${TEST_LOG_DIR}/output" "could not confirm runner 'runner-1' state"
	assert_not_contains "${TEST_LOG_DIR}/output" "Runner 'runner-1' is no longer registered"
}

test_no_active_job_waits_until_busy_grace_deadline() {
	begin_test "no-active-job-busy-until-deadline"
	RUNNER_SHUTDOWN_GRACE_SECONDS="1"
	export FAKE_GH_RUN_IDS=""
	export FAKE_GH_RUNNER_INFO=$'123\ttrue\tonline'
	set_date_sequence 100 101

	cleanup_vm "vm-1" "runner-1" "10.0.0.2" "true" >"${TEST_LOG_DIR}/output" 2>&1 || true

	assert_contains "${TEST_LOG_DIR}/output" "No in-progress GitHub Actions job found for runner 'runner-1'"
	assert_contains "${TEST_LOG_DIR}/output" "runner 'runner-1' is still busy after 1s"
	assert_contains "${TEST_LOG_DIR}/output" "runner 'runner-1' is still busy; skipping GitHub runner deletion"
	assert_contains "${TEST_LOG_DIR}/output" "leaving VM running because GitHub still reports runner 'runner-1' as busy or unknown"
	assert_not_contains "${TEST_LOG_DIR}/gh.log" "/cancel"
	assert_not_contains "${TEST_LOG_DIR}/tart.log" $'tart\tstop\tvm-1'
	assert_not_contains "${TEST_LOG_DIR}/tart.log" $'tart\tdelete\tvm-1'
}

test_zero_shutdown_grace_waits_until_runner_releases() {
	begin_test "zero-shutdown-grace-waits"
	RUNNER_SHUTDOWN_GRACE_SECONDS="0"
	RUNNER_SHUTDOWN_POLL_SECONDS="0"
	set_runner_info_queue $'123\ttrue\tonline' $'123\tfalse\tonline'
	set_date_sequence 100 100 100

	wait_for_runner_to_settle "vm-1" "runner-1" "9001" >"${TEST_LOG_DIR}/output" 2>&1

	assert_contains "${TEST_LOG_DIR}/output" "Runner 'runner-1' is no longer busy"
	assert_not_contains "${TEST_LOG_DIR}/output" "runner 'runner-1' is still busy after 0s"
}

test_busy_runner_retries_job_lookup_and_cancels_race() {
	begin_test "busy-runner-retry-cancel"
	RUNNER_SHUTDOWN_GRACE_SECONDS="1"
	export FAKE_GH_RUN_IDS="9001"
	set_job_line_queue "__EMPTY__" $'9001\t501\tbuild\t"https://example.test/job"'
	set_runner_info_queue $'123\ttrue\tonline' $'123\tfalse\tonline' $'123\tfalse\tonline'
	set_date_sequence 100 100 100

	cleanup_vm "vm-1" "runner-1" "10.0.0.2" "true" >"${TEST_LOG_DIR}/output" 2>&1

	assert_contains "${TEST_LOG_DIR}/output" "Runner 'runner-1' is busy; retrying active job lookup before teardown"
	assert_contains "${TEST_LOG_DIR}/output" "Canceling workflow run 9001"
	assert_contains "${TEST_LOG_DIR}/output" "Runner 'runner-1' is no longer busy"
	assert_contains "${TEST_LOG_DIR}/gh.log" "/repos/owner/repo/actions/runs/9001/cancel"
	assert_contains "${TEST_LOG_DIR}/tart.log" $'tart\tstop\tvm-1'
	assert_contains "${TEST_LOG_DIR}/tart.log" $'tart\tdelete\tvm-1'
}

test_force_cancel_requires_opt_in() {
	begin_test "force-cancel-disabled"
	GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN="false"
	RUNNER_SHUTDOWN_GRACE_SECONDS="1"
	RUNNER_FORCE_CANCEL_AFTER_SECONDS="1"
	export FAKE_GH_RUNNER_INFO=$'123\ttrue\tonline'
	set_date_sequence 100 101

	if wait_for_runner_to_settle "vm-1" "runner-1" "9001" >"${TEST_LOG_DIR}/output" 2>&1; then
		fail "wait_for_runner_to_settle unexpectedly succeeded with busy runner"
	fi
	assert_not_contains "${TEST_LOG_DIR}/gh.log" "/force-cancel"

	begin_test "force-cancel-enabled"
	GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN="true"
	RUNNER_SHUTDOWN_GRACE_SECONDS="1"
	RUNNER_FORCE_CANCEL_AFTER_SECONDS="1"
	export FAKE_GH_RUNNER_INFO=$'123\ttrue\tonline'
	set_date_sequence 100 101

	if wait_for_runner_to_settle "vm-1" "runner-1" "9001" >"${TEST_LOG_DIR}/output" 2>&1; then
		fail "wait_for_runner_to_settle unexpectedly succeeded with busy runner"
	fi
	assert_contains "${TEST_LOG_DIR}/output" "requesting force-cancel"
	assert_contains "${TEST_LOG_DIR}/gh.log" "/repos/owner/repo/actions/runs/9001/force-cancel"
}

test_force_cancel_failure_is_reported() {
	begin_test "force-cancel-failure"
	GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN="true"
	RUNNER_SHUTDOWN_GRACE_SECONDS="1"
	RUNNER_FORCE_CANCEL_AFTER_SECONDS="1"
	export FAKE_GH_RUNNER_INFO=$'123\ttrue\tonline'
	export FAKE_GH_FORCE_CANCEL_EXIT="22"
	export FAKE_GH_FORCE_CANCEL_ERROR="force cancel denied"
	set_date_sequence 100 101

	if wait_for_runner_to_settle "vm-1" "runner-1" "9001" >"${TEST_LOG_DIR}/output" 2>&1; then
		fail "wait_for_runner_to_settle unexpectedly succeeded with busy runner"
	fi

	assert_contains "${TEST_LOG_DIR}/output" "requesting force-cancel"
	assert_contains "${TEST_LOG_DIR}/output" "GitHub API force-cancel for workflow run 9001 failed (exit 22): force cancel denied"
	assert_contains "${TEST_LOG_DIR}/output" "force-cancel for workflow run 9001 was not accepted; manual cancellation may be required"
	assert_contains "${TEST_LOG_DIR}/gh.log" "/repos/owner/repo/actions/runs/9001/force-cancel"
}

test_zero_force_cancel_delay_disables_force_cancel() {
	begin_test "zero-force-cancel-delay"
	GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN="true"
	RUNNER_SHUTDOWN_GRACE_SECONDS="1"
	RUNNER_FORCE_CANCEL_AFTER_SECONDS="0"
	export FAKE_GH_RUNNER_INFO=$'123\ttrue\tonline'
	set_date_sequence 100 101

	if wait_for_runner_to_settle "vm-1" "runner-1" "9001" >"${TEST_LOG_DIR}/output" 2>&1; then
		fail "wait_for_runner_to_settle unexpectedly succeeded with busy runner"
	fi

	assert_not_contains "${TEST_LOG_DIR}/gh.log" "/force-cancel"
	assert_contains "${TEST_LOG_DIR}/output" "runner 'runner-1' is still busy after 1s"
}

test_org_scope_does_not_call_repo_workflow_run_endpoints() {
	begin_test "org-scope"
	GITHUB_SCOPE="org"
	GITHUB_ORG="octo-org"
	GITHUB_REPO="owner/repo"
	export FAKE_GH_RUNNER_INFO=$'123\tfalse\tonline'

	cleanup_vm "vm-1" "runner-1" "10.0.0.2" "true" >"${TEST_LOG_DIR}/output" 2>&1

	assert_contains "${TEST_LOG_DIR}/output" "automatic workflow cancellation currently requires GITHUB_SCOPE=repo"
	assert_not_contains "${TEST_LOG_DIR}/gh.log" "/repos/owner/repo/actions/runs"
	assert_contains "${TEST_LOG_DIR}/gh.log" "/orgs/octo-org/actions/runners"
	assert_contains "${TEST_LOG_DIR}/gh.log" "/orgs/octo-org/actions/runners/123"
}

test_parent_term_reaches_worker_and_cleanup_is_idempotent() {
	begin_test "parent-term"
	export FAKE_GH_RUN_IDS="9001"
	export FAKE_GH_JOB_LINE=$'9001\t501\tbuild\t"https://example.test/job"'
	export FAKE_GH_RUNNER_INFO=$'123\tfalse\tonline'
	export FAKE_DATE_VALUE="1700000000"
	export FAKE_SSH_RUNNER_BLOCK="true"

	PATH="${FAKE_BIN}:${ORIGINAL_PATH}" \
	TEST_LOG_DIR="${TEST_LOG_DIR}" \
	FAKE_GH_RUN_IDS="${FAKE_GH_RUN_IDS}" \
	FAKE_GH_JOB_LINE="${FAKE_GH_JOB_LINE}" \
	FAKE_GH_RUNNER_INFO="${FAKE_GH_RUNNER_INFO}" \
	FAKE_DATE_VALUE="${FAKE_DATE_VALUE}" \
	FAKE_SSH_RUNNER_BLOCK="${FAKE_SSH_RUNNER_BLOCK}" \
	GITHUB_SCOPE="repo" \
	GITHUB_REPO="owner/repo" \
	RUNNER_NAME_BASE="test-runner" \
	RUNNER_POOL_SIZE="1" \
	MAX_RUNS="0" \
	VM_BASE_IMAGE="tahoe-runner" \
	VM_CLONE_PER_RUN="true" \
	GITHUB_CANCEL_RUN_ON_SHUTDOWN="true" \
	GITHUB_FORCE_CANCEL_RUN_ON_SHUTDOWN="false" \
	RUNNER_SHUTDOWN_GRACE_SECONDS="5" \
	RUNNER_FORCE_CANCEL_AFTER_SECONDS="30" \
	RUNNER_SHUTDOWN_POLL_SECONDS="0" \
	bash "${SCRIPT_UNDER_TEST}" >"${TEST_LOG_DIR}/output" 2>&1 &
	CURRENT_PARENT_PID=$!

	wait_for_file "${TEST_LOG_DIR}/remote_started"
	kill -TERM "${CURRENT_PARENT_PID}"

	local status=0
	if wait "${CURRENT_PARENT_PID}"; then
		status=0
	else
		status=$?
	fi
	CURRENT_PARENT_PID=""

	if [[ "$status" != "143" ]]; then
		sed -n '1,260p' "${TEST_LOG_DIR}/output" >&2 || true
		fail "expected parent to exit 143 after TERM, got ${status}"
	fi

	assert_contains "${TEST_LOG_DIR}/output" "Shutting down all workers"
	assert_contains "${TEST_LOG_DIR}/sshpass.log" "signal-runner"
	assert_contains "${TEST_LOG_DIR}/gh.log" "/repos/owner/repo/actions/runs/9001/cancel"
	assert_count "${TEST_LOG_DIR}/tart.log" $'tart\tstop\trunner-w1-1700000000' "1"
	assert_count "${TEST_LOG_DIR}/tart.log" $'tart\tdelete\trunner-w1-1700000000' "1"
	assert_count "${TEST_LOG_DIR}/output" "Shutting down all workers" "1"
}

run_test test_successful_cancel_signals_runner_and_stops_vm
run_test test_runner_list_api_failure_remains_unknown
run_test test_no_active_job_waits_until_busy_grace_deadline
run_test test_zero_shutdown_grace_waits_until_runner_releases
run_test test_busy_runner_retries_job_lookup_and_cancels_race
run_test test_force_cancel_requires_opt_in
run_test test_force_cancel_failure_is_reported
run_test test_zero_force_cancel_delay_disables_force_cancel
run_test test_org_scope_does_not_call_repo_workflow_run_endpoints
run_test test_parent_term_reaches_worker_and_cleanup_is_idempotent

printf 'All macOS Tart runner shutdown tests passed.\n'
