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
	# REPO=<owner/repo> RESOURCE_GROUP=<name> RUNNER_NAME=<name> VIRTUALIZATION_FLAVOR=hyperv|virtualbox
	FUNCTION_APP_NAME="$(FUNCTION_APP_NAME)" \
	API_CLIENT_ID="$$(az ad app list --display-name gh-actions-runner-broker --query '[0].appId' -o tsv)" \
	REPO="$(REPO)" \
	RESOURCE_GROUP="$(RESOURCE_GROUP)" \
	RUNNER_NAME="$(RUNNER_NAME)" \
	VIRTUALIZATION_FLAVOR="$(VIRTUALIZATION_FLAVOR)" \
	bash ./scripts/gh-runner-func/test-endpoints.sh --request-runner

test-gh-runner-func-delete:
	# Usage:
	# make test-gh-runner-func-delete FUNCTION_APP_NAME=<function-app-name> API_CLIENT_ID=<api-client-id> RESOURCE_GROUP=<name> RUNNER_NAME=<name>
	FUNCTION_APP_NAME="$(FUNCTION_APP_NAME)" \
	API_CLIENT_ID="$$(az ad app list --display-name gh-actions-runner-broker --query '[0].appId' -o tsv)" \
	RESOURCE_GROUP="$(RESOURCE_GROUP)" \
	RUNNER_NAME="$(RUNNER_NAME)" \
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