test-build:
	go test -parallel 4 -timeout 300m -v ./pkg/build/*_test.go
	# Terminate any lingering packer and qemu processes started by packer
	pkill -f "packer.*ubuntu.*"
	pkill -f "packer-plugin-qemu.*"
	pkill -f "qemu-system.*ubuntu.*packer.*"