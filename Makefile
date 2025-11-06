test-build:
	# Prompt for sudo rights at the beginning
	sudo -v
	# sudo is required to generate a customized Windows 11 iso
	go test -parallel 4 -timeout 300m -v ./pkg/build/*.go
	# Terminate any lingering packer and qemu processes started by packer
	pkill -f "packer.*ubuntu.*" || true
	pkill -f "packer.*windows.*" || true
	pkill -f "packer-plugin-qemu.*" || true
	pkill -f "qemu-system.*packer.*" || true

test-deploy:
	go test -timeout 20m -v ./pkg/deploy/*.go

test-clean-testcache:
	go clean -testcache

test-build-specific:
	# Usage: make test-build-specific TEST_NAME=<name of the test>
	sudo -v
	go test -timeout 300m -v ./pkg/build/*.go -run $(TEST_NAME)
	# Terminate any lingering packer and qemu processes started by packer
	pkill -f "packer.*ubuntu.*" || true
	pkill -f "packer.*windows.*" || true
	pkill -f "packer-plugin-qemu.*" || true
	pkill -f "qemu-system.*packer.*" || true