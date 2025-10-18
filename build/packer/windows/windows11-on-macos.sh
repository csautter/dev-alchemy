#!/usr/bin/env bash

set +e # we want to continue on errors
set -x

# Function to keep a command alive by restarting it if it exits
# Usage: keep_alive <command>
# This is necessary because vncsnapshot sometimes cannot connect to the VNC server in the first attempt
# and exits with an error. We want to retry a few times before giving up.
keep_alive() {
	local cmd="$*"
	local max_runs=2
	local run_count=0
	while [ $run_count -lt $max_runs ]; do
		if [ $run_count -lt $max_runs ]; then
			sleep 10
		fi
		echo "=====[ $(date) ]====="
		echo "Starting command: $cmd"
		$cmd &
		cmd_exit_code=$?
		local cmd_pid=$!
		wait $cmd_pid
		if [ $run_count -lt $max_runs ]; then
			echo "Command '$cmd' exited with code $cmd_exit_code. Restarting after 10 seconds..."
		fi
		run_count=$((run_count + 1))
	done
}

# Manual argument parsing for portability
arch="arm64"
headless=false

while [[ $# -gt 0 ]]; do
	case "$1" in
	--arch)
		if [[ -n "$2" && ("$2" == "amd64" || "$2" == "arm64") ]]; then
			arch="$2"
			shift 2
		else
			echo "Invalid value for --arch: $2. Allowed values are 'amd64' or 'arm64'." >&2
			exit 1
		fi
		;;
	--headless)
		headless=true
		shift
		;;
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

echo "Using architecture: $arch"
echo "Headless mode: $headless"

script_dir=$(
	# shellcheck disable=SC2164
	cd "$(dirname "$0")"
	pwd -P
)
project_root=$(
	# shellcheck disable=SC2164
	cd "${script_dir}/../../.."
	pwd -P
)

cd "${project_root}" || exit 1

# download the Windows 11 ISO if not already present
if [ ! -d ./vendor/windows ]; then
	mkdir -p ./vendor/windows
fi
if [ ! -f "./vendor/windows/win11_25H2_english_$arch.iso" ]; then
	echo "Downloading Windows 11 $arch ISO"
	cd "${project_root}/scripts/macos/" || exit 1
	if [ ! -d .venv ]; then
		python3 -m venv .venv
	fi
	# shellcheck disable=SC1091
	source .venv/bin/activate

	if ! python -c "import playwright" &>/dev/null; then
		pip install playwright
		python -m playwright install chromium
	fi

	if [ "$arch" = "amd64" ]; then
		python playwright_win11_iso.py
	elif [ "$arch" = "arm64" ]; then
		python playwright_win11_iso.py --arm
	fi
	cd "${project_root}/vendor/windows/" || exit 1

	if [ $headless = true ]; then
		echo "Running in headless mode, skipping ISO download progress bar"
		curl -o "win11_25h2_english_$arch.iso" "$(cat "./win11_${arch}_iso_url.txt")"
	else
		echo "Running in interactive mode, showing ISO download progress bar"
		curl --progress-bar -o "win11_25h2_english_$arch.iso" "$(cat "./win11_${arch}_iso_url.txt")"
	fi

	cd "${project_root}" || exit 1
else
	echo "Windows 11 $arch ISO already exists, skipping download"
fi

bash scripts/macos/download-utm-guest-tools.sh

if [ "$arch" = "arm64" ]; then
	# download the qemu-uefi files if not already present
	bash scripts/macos/download-arm64-uefi.sh

	# builds the autounattend ISO with the current autounattend.xml file
	bash scripts/macos/create-win11-autounattend-iso.sh

	# download the virtio-win ISO if not already present
	bash scripts/macos/download-virtio-win-iso.sh

fi

# creates the qcow2 disk image and overwrites it if it already exists
bash scripts/macos/create-qemu-qcow2-disk.sh --arch $arch

packer init "build/packer/windows/windows11-$arch-on-macos.pkr.hcl"

# record video in headless mode
if [ $headless = true ]; then
	mkdir -p "$project_root/build/packer/windows/.build_tmp/windows11-$arch-on-macos-output"
	# set VNC password to "packer"
	packer_password="packer"
	expect <<EOD
spawn vncpasswd "$project_root/build/packer/windows/.build_tmp/packer-qemu.vnc.pass"
expect "Password:"
send "$packer_password\n"
expect "Verify:"
send "$packer_password\n"
expect eof
EOD

	# use different VNC ports for amd64 and arm64 builds to allow parallel execution
	if [ "$arch" = "amd64" ]; then
		# on amd64 we use the standard localhost:2 display
		echo "Using VNC display localhost:2 for amd64 build"
		echo "You can connect to it using a VNC viewer with password '$packer_password' on localhost:5902"
		vnc_port=2
	else
		# on arm64 we use the localhost:1 display
		echo "Using VNC display localhost:1 for arm64 build"
		echo "You can connect to it using a VNC viewer with password '$packer_password' on localhost:5901"
		vnc_port=1
	fi
	keep_alive "vncsnapshot -quiet -passwd $project_root/build/packer/windows/.build_tmp/packer-qemu.vnc.pass -compresslevel 9 -count 14400 -fps 1 localhost:$vnc_port $project_root/build/packer/windows/.build_tmp/windows11-$arch-on-macos-output/packer-qemu.vnc.jpg" &
	vncsnapshot_pid=$!
	echo "Started vncsnapshot with PID $vncsnapshot_pid"

	# shellcheck disable=SC2064
	trap "echo 'Stopping vncsnapshot process '$vncsnapshot_pid'; kill -SIGINT $vncsnapshot_pid; wait $vncsnapshot_pid; echo 'vncsnapshot process $vncsnapshot_pid has finished'" EXIT
fi

PACKER_LOG=1 packer build -var "iso_url=${project_root}/vendor/windows/win11_25H2_english_$arch.iso" -var "headless=$headless" "build/packer/windows/windows11-$arch-on-macos.pkr.hcl"
packer_exit_code=$?

if [ $headless = true ]; then
	# shellcheck disable=SC2086
	kill -SIGINT $vncsnapshot_pid
	# shellcheck disable=SC2086
	wait $vncsnapshot_pid
	echo "vncsnapshot process $vncsnapshot_pid has finished"
	# create mp4 video from jpg images
	ffmpeg -framerate 1 -i "$project_root/build/packer/windows/.build_tmp/windows11-$arch-on-macos-output/packer-qemu.vnc%05d.jpg" -c:v libx264 -pix_fmt yuv420p "$project_root/build/packer/windows/.build_tmp/windows11-$arch-on-macos-output/packer-qemu.vnc.mp4"
	echo "Created video $project_root/build/packer/windows/.build_tmp/windows11-$arch-on-macos-output/packer-qemu.vnc.mp4"
	find "$project_root/build/packer/windows/.build_tmp/windows11-$arch-on-macos-output" -name 'packer-qemu.vnc*.jpg' -delete
fi

exit $packer_exit_code
