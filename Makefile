test-build:
	# sudo is required to generate a customized Windows 11 iso
	cd ./pkg/build && go test -parallel 4 -timeout 300m -v .
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
	cd ./pkg/build && go test -timeout 300m -v . -run $(TEST_NAME)
	# Terminate any lingering packer and qemu processes started by packer
	pkill -f "packer.*ubuntu.*" || true
	pkill -f "packer.*windows.*" || true
	pkill -f "packer-plugin-qemu.*" || true
	pkill -f "qemu-system.*packer.*" || true

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
	# make test-gh-runner-func-delete FUNCTION_APP_NAME=<function-app-name> RESOURCE_GROUP=<name> RUNNER_NAME=<name>
	# Optional: API_CLIENT_ID=<api-client-id> TENANT_ID=<tenant-id>
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
