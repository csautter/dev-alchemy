#!/usr/bin/env bash

set +e
set -x

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
		local cmd_pid=$!
		wait $cmd_pid
		if [ $run_count -lt $max_runs ]; then
			echo "Command '$cmd' exited with code $?. Restarting after 10 seconds..."
		fi
		run_count=$((run_count + 1))
	done
}

HEADLESS=false

while [[ $# -gt 0 ]]; do
	case "$1" in
	--headless)
		HEADLESS=true
		shift
		;;
	*)
		shift
		;;
	esac
done
SCRIPT_DIR=$(
	cd $(dirname $0)
	pwd -P
)
PROJECT_ROOT=$(
	cd ${SCRIPT_DIR}/../../..
	pwd -P
)

cd ${PROJECT_ROOT}

# download the Windows 11 ARM64 ISO if not already present
if [ ! -d ./vendor/windows ]; then
	mkdir -p ./vendor/windows
fi
if [ ! -f ./vendor/windows/win11_25H2_english_arm64.iso ]; then
	echo "Downloading Windows 11 ARM64 ISO"
	cd $PROJECT_ROOT/scripts/macos/
	if [ ! -d .venv ]; then
		python3 -m venv .venv
	fi
	source .venv/bin/activate

	if ! python -c "import playwright" &>/dev/null; then
		pip install playwright
		python -m playwright install chromium
	fi

	python playwright_win11_iso.py --arm
	cd $PROJECT_ROOT/vendor/windows/

	if [ $HEADLESS = true ]; then
		echo "Running in headless mode, skipping ISO download progress bar"
		curl -o win11_25h2_english_arm64.iso $(cat ./win11_arm_iso_url.txt)
	else
		echo "Running in interactive mode, showing ISO download progress bar"
		curl --progress-bar -o win11_25h2_english_arm64.iso $(cat ./win11_arm_iso_url.txt)
	fi

	cd ${PROJECT_ROOT}
else
	echo "Windows 11 ARM64 ISO already exists, skipping download"
fi

# download the qemu-uefi files if not already present
bash scripts/macos/download-arm64-uefi.sh

# builds the autounattend ISO with the current autounattend.xml file
bash scripts/macos/create-win11-autounattend-iso.sh

# download the virtio-win ISO if not already present
bash scripts/macos/download-virtio-win-iso.sh

# creates the qcow2 disk image and overwrites it if it already exists
bash scripts/macos/create-qemu-qcow2-disk.sh

packer init build/packer/windows/windows11-arm64-on-macos.pkr.hcl

# record video in headless mode
if [ $HEADLESS = true ]; then
	mkdir -p $PROJECT_ROOT/build/packer/windows/.build_tmp/windows11-arm64-on-macos-output
	# set VNC password to "packer"
	packer_password="packer"
	expect <<EOD
spawn vncpasswd $PROJECT_ROOT/build/packer/windows/.build_tmp/packer-qemu.vnc.pass
expect "Password:"
send "$packer_password\n"
expect "Verify:"
send "$packer_password\n"
expect eof
EOD
	# https://manpages.ubuntu.com/manpages/jammy/man1/vncsnapshot.1.html
	keep_alive "vncsnapshot -quiet -passwd $PROJECT_ROOT/build/packer/windows/.build_tmp/packer-qemu.vnc.pass -compresslevel 9 -count 14400 -fps 1 localhost:1 $PROJECT_ROOT/build/packer/windows/.build_tmp/windows11-arm64-on-macos-output/packer-qemu.vnc.jpg" &
	VNCSNAPSHOT_PID=$!
	echo "Started vncsnapshot with PID $VNCSNAPSHOT_PID"

	trap "echo 'Stopping vncsnapshot process $VNCSNAPSHOT_PID'; kill -SIGINT $VNCSNAPSHOT_PID; wait $VNCSNAPSHOT_PID; echo 'vncsnapshot process $VNCSNAPSHOT_PID has finished'" EXIT
fi

PACKER_LOG=1 packer build -var "iso_url=./vendor/windows/win11_25H2_english_arm64.iso" -var "headless=$HEADLESS" build/packer/windows/windows11-arm64-on-macos.pkr.hcl
PACKER_EXIT_CODE=$?

if [ $HEADLESS = true ]; then
	kill -SIGINT $VNCSNAPSHOT_PID
	wait $VNCSNAPSHOT_PID
	echo "vncsnapshot process $VNCSNAPSHOT_PID has finished"
	# create mp4 video from jpg images
	ffmpeg -framerate 1 -i $PROJECT_ROOT/build/packer/windows/.build_tmp/windows11-arm64-on-macos-output/packer-qemu.vnc%05d.jpg -c:v libx264 -pix_fmt yuv420p $PROJECT_ROOT/build/packer/windows/.build_tmp/windows11-arm64-on-macos-output/packer-qemu.vnc.mp4
	echo "Created video $PROJECT_ROOT/build/packer/windows/.build_tmp/windows11-arm64-on-macos-output/packer-qemu.vnc.mp4"
	find "$PROJECT_ROOT/build/packer/windows/.build_tmp/windows11-arm64-on-macos-output" -name 'packer-qemu.vnc*.jpg' -delete
fi

exit $PACKER_EXIT_CODE
