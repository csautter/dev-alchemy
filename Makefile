GOSEC ?= $(shell command -v gosec 2>/dev/null || echo "$$(go env GOPATH)/bin/gosec")
BINARY_NAME ?= alchemy
DIST_DIR ?= dist
MAIN_PACKAGE ?= ./cmd/main.go
VERSION ?= dev
GO_LDFLAGS ?= -s -w
RELEASE_GOOS ?= darwin linux windows
RELEASE_GOARCH ?= amd64 arm64

PACKAGE_VERSION = $(patsubst v%,%,$(VERSION))
TARGET_BINARY_NAME = $(BINARY_NAME)$(if $(filter windows,$(GOOS)),.exe,)
TARGET_DIST_DIR = $(DIST_DIR)/$(GOOS)-$(GOARCH)
RELEASE_ASSET_BASENAME = dev-alchemy_$(PACKAGE_VERSION)_$(GOOS)_$(GOARCH)

.PHONY: build build-cli-local build-cli-target build-cli-release package-cli-target package-cli-release clean-dist \
	test-build-runner test-build test-deploy test-provision test-deploy-windows-hyperv test-clean-testcache \
	test-build-integration test-build-specific test-gh-runner-func-request test-gh-runner-func-delete \
	deploy-plan-terraform-azure-gh-runner deploy-apply-terraform-azure-gh-runner deploy-az-func-app gosec

build:
	go build ./...

build-cli-local:
	$(MAKE) build-cli-target GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH)

build-cli-target:
	@test -n "$(GOOS)" || (echo "GOOS is required" && exit 1)
	@test -n "$(GOARCH)" || (echo "GOARCH is required" && exit 1)
	mkdir -p "$(TARGET_DIST_DIR)"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags '$(GO_LDFLAGS)' -o "$(TARGET_DIST_DIR)/$(TARGET_BINARY_NAME)" $(MAIN_PACKAGE)

build-cli-release: clean-dist
	@set -e; \
	for goos in $(RELEASE_GOOS); do \
		for goarch in $(RELEASE_GOARCH); do \
			$(MAKE) build-cli-target GOOS=$$goos GOARCH=$$goarch VERSION=$(VERSION); \
		done; \
	done

package-cli-target: build-cli-target
	@test -n "$(GOOS)" || (echo "GOOS is required" && exit 1)
	@test -n "$(GOARCH)" || (echo "GOARCH is required" && exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	rm -f "$(DIST_DIR)/$(RELEASE_ASSET_BASENAME).tar.gz" "$(DIST_DIR)/$(RELEASE_ASSET_BASENAME).zip"
	if [ "$(GOOS)" = "windows" ]; then \
		cd "$(TARGET_DIST_DIR)" && zip -q -j "../$(RELEASE_ASSET_BASENAME).zip" "$(TARGET_BINARY_NAME)"; \
	else \
		tar -C "$(TARGET_DIST_DIR)" -czf "$(DIST_DIR)/$(RELEASE_ASSET_BASENAME).tar.gz" "$(TARGET_BINARY_NAME)"; \
	fi

package-cli-release: clean-dist
	@set -e; \
	for goos in $(RELEASE_GOOS); do \
		for goarch in $(RELEASE_GOARCH); do \
			$(MAKE) package-cli-target GOOS=$$goos GOARCH=$$goarch VERSION=$(VERSION); \
		done; \
	done

clean-dist:
	rm -rf "$(DIST_DIR)"

test-build-runner:
	go test ./cmd/cmd/... -run "TestParallelBuilds|TestSequentialBuilds" -v -timeout 60s

test-build:
	# sudo is required to generate a customized Windows 11 iso
	cd ./pkg/build && go test -parallel 4 -timeout 300m -v .

test-deploy:
	go test -timeout 20m -v ./pkg/deploy/...

test-provision:
	go test -timeout 20m -v ./pkg/provision/...

test-deploy-windows-hyperv:
	RUN_INTEGRATION_TESTS=1 RUN_WINDOWS_HYPERV_DEPLOY_SMOKE=1 go test -timeout 60m -v ./pkg/deploy -run "TestGetHypervWindowsBoxPath_UsesExpectedBuildArtifact|TestGetHypervWindowsBoxPath_FallsBackToCachePath|TestRunHypervVagrantDeployOnWindows_Smoke"

test-clean-testcache:
	go clean -testcache

test-build-integration:
	cd ./pkg/build && RUN_INTEGRATION_TESTS=1 go test -timeout 300m -v . -run "TestIntegrationDependencyReconciliation|TestGetWindows11DownloadAmd64|TestGetWindows11DownloadArm64|TestResolveDebianPackageURL|TestResolveAndDownloadQemuEfiAarch64"

test-build-specific:
	# Usage: make test-build-specific TEST_NAME=<name of the test>
	cd ./pkg/build && go test -timeout 300m -v . -run $(TEST_NAME)

test-gh-runner-func-request:
	# Usage:
	# make test-gh-runner-func-request FUNCTION_APP_NAME=<function-app-name>
	# Example:
	# make test-gh-runner-func-request FUNCTION_APP_NAME=my-func-app
	# Optional:
	# API_CLIENT_ID=<api-client-id> TENANT_ID=<tenant-id>
	# REPO=<owner/repo> RESOURCE_GROUP=<name> RUNNER_NAME=<name> VIRTUALIZATION_FLAVOR=hyperv|virtualbox
	# AZURE_BROKER_CONFIG_DIR=~/.azure-broker AZURE_ADMIN_CONFIG_DIR=~/.azure
	# TERRAGRUNT_OUTPUT_DIR=deployments/terraform/env/azure_dev/azure_gh_runner
	FUNCTION_APP_NAME="$(FUNCTION_APP_NAME)" \
	API_CLIENT_ID="$(API_CLIENT_ID)" \
	TENANT_ID="$(TENANT_ID)" \
	REPO="$(REPO)" \
	RESOURCE_GROUP="$(RESOURCE_GROUP)" \
	RUNNER_NAME="$(RUNNER_NAME)" \
	VIRTUALIZATION_FLAVOR="$(VIRTUALIZATION_FLAVOR)" \
	AZURE_BROKER_CONFIG_DIR="$(AZURE_BROKER_CONFIG_DIR)" \
	AZURE_ADMIN_CONFIG_DIR="$(AZURE_ADMIN_CONFIG_DIR)" \
	TERRAGRUNT_OUTPUT_DIR="$(TERRAGRUNT_OUTPUT_DIR)" \
	bash ./scripts/gh-runner-func/test-endpoints.sh --request-runner

test-gh-runner-func-delete:
	# Usage:
	# make test-gh-runner-func-delete FUNCTION_APP_NAME=<function-app-name> RESOURCE_GROUP=<name>
	# Optional: API_CLIENT_ID=<api-client-id> TENANT_ID=<tenant-id> RUNNER_NAME=<name>
	FUNCTION_APP_NAME="$(FUNCTION_APP_NAME)" \
	API_CLIENT_ID="$(API_CLIENT_ID)" \
	TENANT_ID="$(TENANT_ID)" \
	RESOURCE_GROUP="$(RESOURCE_GROUP)" \
	RUNNER_NAME="$(RUNNER_NAME)" \
	AZURE_BROKER_CONFIG_DIR="$(AZURE_BROKER_CONFIG_DIR)" \
	AZURE_ADMIN_CONFIG_DIR="$(AZURE_ADMIN_CONFIG_DIR)" \
	TERRAGRUNT_OUTPUT_DIR="$(TERRAGRUNT_OUTPUT_DIR)" \
	bash ./scripts/gh-runner-func/test-endpoints.sh --delete-resource-group

deploy-plan-terraform-azure-gh-runner:
	cd deployments/terraform/env/azure_dev/azure_gh_runner && terragrunt plan -out tf.plan

deploy-apply-terraform-azure-gh-runner:
	cd deployments/terraform/env/azure_dev/azure_gh_runner && terragrunt apply tf.plan

deploy-az-func-app:
	# Usage:
	# make deploy-az-func-app FUNCTION_APP_NAME=<function-app-name>
	cd ./scripts/gh-runner-func/ && \
	func azure functionapp publish $(FUNCTION_APP_NAME)

gosec:
	@if [ ! -x "$(GOSEC)" ]; then \
		echo "gosec not found at $(GOSEC). Rebuild the dev container or run: go install github.com/securego/gosec/v2/cmd/gosec@v2.25.0"; \
		exit 1; \
	fi
	$(GOSEC) -no-fail -fmt sarif -out gosec-results.sarif -exclude-generated ./...
