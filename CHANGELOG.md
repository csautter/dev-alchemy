# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0](https://github.com/csautter/dev-alchemy/compare/v0.3.0...v0.4.0) (2026-03-22)


### Added

* **build:** add --no-cache to force rebuilding existing VM artifacts ([13c1783](https://github.com/csautter/dev-alchemy/commit/13c17833328fb144700dc9193552509b71ebd0a2))
* **build:** add list command for available VM build combinations ([023d67f](https://github.com/csautter/dev-alchemy/commit/023d67f9673905b469b03cebe6402d3a2ecebe67))
* **cli:** add install command for host dependencies ([0ffd087](https://github.com/csautter/dev-alchemy/commit/0ffd0877a9f9dec95a39d298f4cb26844849c61d))
* **create:** add create list command and artifact readiness checks ([1fc0bed](https://github.com/csautter/dev-alchemy/commit/1fc0bedfd180ae9b33957d57fea5982892f1d157))
* **provision:** add supported target listing and filter unsupported VMs ([f378c06](https://github.com/csautter/dev-alchemy/commit/f378c064aa14e8cc1a148237ef1ce5b1ee8a50d1))
* **provision:** add UTM Windows 11 provisioning support ([556ce74](https://github.com/csautter/dev-alchemy/commit/556ce749c58af9f2998f9f3f53840e5e1f75c6a0))
* **provision:** add UTM Windows 11 provisioning support ([baacca3](https://github.com/csautter/dev-alchemy/commit/baacca35df2846beb061057978ba17bc3bd4304e))


### Fixed

* address multiple gosec findings ([b3a2cf1](https://github.com/csautter/dev-alchemy/commit/b3a2cf1582e2a5b5060f34cc81115b8744dde930))
* **ci:** restore artifact directory access after gosec hardening ([2855f00](https://github.com/csautter/dev-alchemy/commit/2855f000245dcd7dd1c885b07d831b093f3bec48))
* **ci:** restore cache ownership after sudo macos builds ([fc9e4e2](https://github.com/csautter/dev-alchemy/commit/fc9e4e2edbb7b5445c9ada621e9e9a77931b26e4))
* **deploy:** re-prime UTM ARP cache while waiting for guest IPv4 ([47206e9](https://github.com/csautter/dev-alchemy/commit/47206e9e71426e12e7c7f66b082089997d142b55))
* **deploy:** retry UTM IP discovery and warm ARP cache ([1f40162](https://github.com/csautter/dev-alchemy/commit/1f40162297960058b3c74a269990a32aa87a7e79))
* **windows:** keep WinRM reachable across network profile changes ([00d0dcf](https://github.com/csautter/dev-alchemy/commit/00d0dcfdcdad107945566e8d6321a6edd22d7465))


### Changed

* deduplicate build/create/provision VM list rendering ([75af05e](https://github.com/csautter/dev-alchemy/commit/75af05e8f9c6947018967bf44da1941f2e00a2b8))


### CI

* align Windows and macOS build job timeouts with packer timeouts ([2d72d46](https://github.com/csautter/dev-alchemy/commit/2d72d4659e8523c02e6ed48fdbb42dd44ecf5f2e))
* run cmd and pkg/build unit tests in GitHub Actions ([630b74e](https://github.com/csautter/dev-alchemy/commit/630b74e11932df822c755efa5200f47af25c1181))

## [0.3.0](https://github.com/csautter/dev-alchemy/compare/v0.2.0...v0.3.0) (2026-03-17)


### Added

* **ansible:** add configurable Debian Spotify install with apt/snap fallback ([c5ae866](https://github.com/csautter/dev-alchemy/commit/c5ae866242d7c378f5f054e10a983a25f5c84b80))
* **build:** add Hyper-V Ubuntu support on Windows and restructure cloud-init configs ([eaef86d](https://github.com/csautter/dev-alchemy/commit/eaef86df0f0c57f87fe8830c867a92e536b9ca52))
* **build:** add Hyper-V Ubuntu support on Windows and restructure cloud-init configs ([f99f622](https://github.com/csautter/dev-alchemy/commit/f99f62293c4f02e4c69df7046134d0713b6ba729))
* **hyperv:** parameterize Vagrant VM resources from build config ([ec52f11](https://github.com/csautter/dev-alchemy/commit/ec52f113796b11bb63be6751139f5378dd165a1c))


### Fixed

* **ansible:** make installer roles check-mode safe ([f0b1f37](https://github.com/csautter/dev-alchemy/commit/f0b1f373a1e547723cbbd7daa0d83336d8d484c4))
* **kubectl:** avoid Windows Chocolatey failure when newer kubernetes-cli is already installed ([174b1fd](https://github.com/csautter/dev-alchemy/commit/174b1fdce174a12e89050f9758e8d42cf4055da8))


### CI

* add deploy smoke tests to macOS and Windows build workflows ([93951cf](https://github.com/csautter/dev-alchemy/commit/93951cfbe62fd6bc20fbb0271c76f5f677419749))
* extend cmd unit-test workflow to cover provision and deploy tests ([95a4e50](https://github.com/csautter/dev-alchemy/commit/95a4e50be12377eb164e914ff54fea3cbc9cd8aa))
* **windows:** set test-hyperv-build job timeout to 60 minutes ([d1086c3](https://github.com/csautter/dev-alchemy/commit/d1086c3709e606b0c0de28739cd6011520010e89))

## [Unreleased]

## [0.2.0](https://github.com/csautter/dev-alchemy/compare/v0.1.1...v0.2.0) (2026-03-14)

### Added

#### Windows Hyper-V Vagrant Deploy
- Hyper-V Vagrant deployment path wired into the `create` command when the VM config uses the Hyper-V virtualization engine.
- Hyper-V Vagrantfile now pins a switch via `VAGRANT_HYPERV_SWITCH` to avoid interactive network selection.

#### Provision Command
- Added a unified provisioning command for VM targets: `go run cmd/main.go provision <osname>`.
- Added Windows 11 Hyper-V provisioning flow through the Go wrapper (`pkg/deploy/provision.go`), including `--check` support.

#### Devcontainer
- Added a Go devcontainer definition with Python and Packer features (`.devcontainer/devcontainer.json`).

#### Azure Runner Broker (Function App)
- New Azure Function App (`scripts/gh-runner-func/`) that provisions and tears down self-hosted GitHub runner VMs on demand.
- HTTP endpoint `POST /api/request_runner` — creates runner resource group, network, and VM.
- HTTP endpoint `POST /api/delete_resource_group` — deletes a named resource group after the job completes.
- Terraform/Terragrunt module (`deployments/terraform/modules/azure_gh_runner/`) that deploys the Function App, Key Vault, storage account, virtual network, and all supporting Azure resources.
- Terraform/Terragrunt base structure (`deployments/terraform/`) with root `hcl`, local and Azure backends, and per-environment overlays.
- Function App reads runner registration token and secrets from Azure Key Vault at runtime.
- OIDC-based authentication for GitHub Actions workflows calling Azure — eliminates long-lived `AZURE_CREDENTIALS` secrets.
- Operator README (`scripts/gh-runner-func/README.md`) documenting API contracts, authentication requirements, required app settings, and deployment steps.
- Terraform module outputs for downstream consumers.
- gitleaks secret-scanning CI job to catch credentials in source.

#### Windows Host Build Automation
- Automated VirtualBox Windows 11 VM build path on a Windows host (`pkg/build/windows-build.go`).
- Automated Hyper-V Windows 11 VM build path on a Windows host, including Default Switch configuration.
- Build test CI job targeting a Windows host runner (`.github/workflows/test-build.yml`).
- Ansible role and playbook additions: VirtualBox provisioning for the Windows 11 GitHub runner image.
- Windows runner setup: Bash, Make, and Packer installed and added to `PATH` on the runner.

#### macOS Tart Runner Infrastructure
- Shell script (`scripts/macos/`) to create ephemeral macOS GitHub runners in a loop using Tart VMs.
- Golden image creation pipeline for macOS Tart runners (pre-installs GitHub runner, Homebrew packages, Packer, Go).
- Graceful runner de-registration on script termination.

#### CI: ISO Caching
- Runner ISO cache directory (`./cache/windows11/iso/`) for all Windows 11 build jobs.
- Cache population step on first run per host so subsequent runs skip the download.
- Azure Blob Storage ISO upload and download composite actions (`build/gh_actions/`) for cross-host cache warming.
- `iso-path` output propagated from the download action to downstream build steps.
- Windows 11 ISO download test (`test: added windows11 download test`).

#### CI: Azure VM Build Runners
- Azure VM temp disk (`/mnt`) used as the Packer build workspace to avoid OS disk I/O bottlenecks.
- Spot-instance support for Azure Windows runners with automatic fallback to on-demand pricing.
- Build job opt-in variables for manual workflow dispatches (`VM_USE_SPOT`, `CUSTOM_IMAGE_ID`, Hyper-V / VirtualBox flavor selection).
- Fallback mechanism: if the self-hosted runner label is unavailable, the workflow automatically falls back to a GitHub-hosted runner.

#### CI: Build-Runner Tests
- Comprehensive unit-test suite for `runParallelBuilds` covering 6 scenarios: all succeed (parallelism=2), partial failure with others still running, SIGINT via context cancel, OS SIGINT signal wiring, sequential-all-succeed, and sequential-failure-does-not-skip-remainder (`cmd/cmd/build_parallel_test.go`).
- New GitHub Actions workflow (`.github/workflows/test-build-runner.yml`) that runs the build-runner tests on push/PR changes to `cmd/cmd/build.go`, `cmd/cmd/build_parallel_test.go`, and the workflow file itself.
- `make test-build-runner` Makefile target for running the build-runner unit tests locally.

---

### Changed

#### Deploy Command Runner
- macOS UTM deploy now uses a shared command runner with streamed stdout/stderr and timeouts.
- Hyper-V Vagrant instructions now reference the cache path for the Windows 11 box.
- Hyper-V Windows provisioning now discovers the VM host IP on demand and runs Ansible with an inline host target instead of mutating a tracked inventory file.
- WinRM settings for Hyper-V Windows provisioning are sourced from process environment or project-root `.env`.
- Command logging/error surfaces now redact `ansible_password` values in CLI arguments.

#### CI Workflow Topology
- `test-build.yml` restructured: separate matrix jobs for Hyper-V and VirtualBox flavors; `fail-fast: false` set on the matrix.
- Azure runner region moved to `eastus2`; VM type upgraded to `Standard_D4s_v5` for faster builds.
- Dynamic memory allocation for Packer Win11 builds replaces hard-coded values.
- ISO paths unified to `./cache/windows11/iso/` across all workflow steps (was `vendor/windows/`).
- Blob upload/download steps refactored into reusable composite actions.
- Job dependency graph optimised for faster Azure runner cleanup after builds complete.
- macOS jobs now run on self-hosted Tart-based runners instead of GitHub-hosted macOS.
- Tart runner prepare script simplified; GitHub CLI auth check added before VM build.

#### Build Package (`pkg/build`) / `cmd/cmd`
- Windows build code extracted into dedicated file (`windows-build.go`); generic helpers moved to `generic_build.go`.
- `checkIfBuildArtifactsExist` function extracted for reuse.
- Build script handling refactored into smaller, focused functions.
- VNC recording config now passed by reference.
- Windows ISO path constant updated to `./cache` directory.
- `runParallelBuilds` extracted to a standalone, context-aware function in `cmd/cmd/build.go`; errors from individual VM builds are now collected and reported (with VM identity) instead of being silently discarded.

#### Hyper-V Setup
- Setup logic refactored; PowerShell provisioning script extracted from inline Python code.
- Restart sequence corrected; existing-resource handling hardened.

#### Terraform / Infrastructure
- Azure storage account creation moved into the Terraform module (removed from GitHub Actions workflow steps).
- Custom VM names and resource group names passed explicitly from CI to the Function App.
- Key Vault name passed to the Function App via environment variable.

#### Documentation
- `.github/runners/README.md` ISO path references updated from `vendor/windows/` to `cache/windows11/iso/`.
- VirtioFS mount path corrected from `/Volumes/iso-cache/` to `/Volumes/My Shared Files/iso-cache/` to match the actual Tart `--dir` mount name.

---

### Fixed

- Deadlock in `stopVncScreenCaptureOnMacosDarwin`: non-blocking channel send now used when the
  VNC goroutine has already exited on a successful vncsnapshot run.
- Build hanging after all VMs complete: `RunExternalProcessWithRetries` previously returned
  `context.Background()` (never done) on retry exhaustion; now returns a cancelled context so
  dependents unblock correctly.
- SIGINT/SIGTERM during vncsnapshot retry-interval sleep no longer causes a hang; the sleep
  is now interruptible via signal.
- Removed hardcoded `-k de` (German keyboard layout) from QEMU args that caused incorrect
  inputs in the `boot_command` sequence on non-German systems.

#### Security & Auth
- **Azure Function auth guard hardened**: JWT-based validation now enforced; dead key-based auth code removed (`fix(auth): harden function app auth guard`).
- **Input validation added** to Function App endpoints; error responses no longer leak internal stack traces or infrastructure detail (`fix(security): validate inputs and prevent error detail leakage`).
- OAuth2 scope added and Azure CLI pre-authorized on the app registration (`fix(auth): add OAuth2 scope and pre-authorize Azure CLI on app registration`).
- Function App endpoint secured so only Azure CLI / OIDC-authenticated callers can invoke it.
- OIDC subject claim fixed for pull request workflows (`fix: oidc subject for gh prs`).
- Azure login step added to the CI cleanup job so resource group deletion succeeds when credentials rotate.

#### Build & Runtime
- Packer `init` errors in the Windows build path are now propagated and fail fast instead of being silently swallowed (`fix(build): propagate packer init errors in Windows build path`).
- Dependency bootstrap failures (venv creation, pip install, Playwright setup) now propagate immediately with context instead of continuing with a broken environment (`fix(build): propagate dependency bootstrap failures immediately`).
- Duplicate `cmd.Wait()` call removed — eliminated spurious `"wait: no child processes"` errors in long-running build processes.
- Azure VM name truncated to 15 characters to satisfy Windows NetBIOS naming constraint.
- Hyper-V Default Switch IP race condition mitigated with retry logic on early build failures.
- Hyper-V Windows 11 boot command timing corrected.
- Max CPU core count capped at 2 to respect hypervisor limits; CPU count variables added to VirtualBox and Hyper-V configs.
- Playwright `stealth` plugin installed during dependency reconciliation.
- Incomplete Playwright downloads cleaned up before retry.
- Playwright Win 11 download approach reverted to stable method.
- Windows ISO download skips re-download if the file already exists.
- PowerShell log file encoding fixed to UTF-8.
- `HostOs` field set correctly for macOS VM definitions.
- Blocked channel removed from build pipeline.

#### CI
- `go mod vendor` run before `go test` to ensure vendored dependencies are present on the runner.
- Go test invocation changed from file path to package path (e.g. `./pkg/build/...`).
- `go test` on macOS build jobs now runs with `sudo` when required.
- Workflow trigger path filter updated to reference the current workflow file (was pointing to a deleted file).
- Azure Blob upload step now checks for file existence before attempting upload.
- ISO download step no longer masks errors from `az storage blob download`.
- Permissions added to the blob upload job (`id-token: write`, `contents: read`).
- macOS Tart ephemeral runners safely removed after use.
- Runner label shortened to satisfy GitHub runner name length limit.
- `az rest` CLI command updated with `--resource` flag.
- Python setup fixed for custom GitHub Actions Windows runner.

#### Terraform
- Terraform `random` integer range restored to a meaningful spread with a `lifecycle { ignore_changes }` guard to prevent resource replacement on re-apply (`fix(terraform): restore random suffix range with stable lifecycle guard`).

---

### Security

> The following items were addressed in this branch; open items are tracked as follow-up work.

| # | Finding | Status |
|---|---------|--------|
| 1 | Function App reachable anonymously (`unauthenticated_action = "AllowAnonymous"`) | **Partially mitigated** — auth guard hardened; `unauthenticated_action` enforcement tracked as follow-up |
| 2 | Auth gate validated only on header *presence*, not JWT claims | **Fixed** — JWT-based validation enforced; dead code removed |
| 3 | Function identity holds subscription-wide `Contributor` | **Open** — least-privilege scoping tracked as follow-up |
| 4 | RDP port 3389 open to `*` in NSG | **Open** — tracked as follow-up |
| 5 | Input validation absent; stack traces leaked in error responses | **Fixed** — input sanitised; error responses scrubbed |
| 6 | `random_integer` range pinned to a single value (no uniqueness) | **Fixed** — range restored with lifecycle guard |

---

### Breaking Changes

> Operators upgrading from `origin/main` (`baf420f5`) must action all items below before deploying.

#### New Required Secrets / Environment Variables
| Name | Where | Purpose |
|------|-------|---------|
| `AZURE_CLIENT_ID` | GitHub Actions / OIDC | Replaces `AZURE_CREDENTIALS` JSON secret |
| `AZURE_TENANT_ID` | GitHub Actions / OIDC | Required for OIDC federation |
| `AZURE_SUBSCRIPTION_ID` | GitHub Actions / OIDC | Required for OIDC federation |
| `FUNCTION_KEY` | GitHub Actions | HTTP key to call the runner broker Function App |
| `CUSTOM_IMAGE_ID` | GitHub Actions (optional) | Pre-baked Azure VM image for runner VMs |
| `VM_USE_SPOT` | GitHub Actions (optional) | Set `true` to request spot pricing |
| `KEYVAULT_NAME` | Terraform / Function App env | Key Vault holding the runner PAT |

#### Workflow Changes
- `test-build.yml` is the canonical build-test workflow; the old `test-packer-build-win11-on-macos.yml` has been removed — update any branch protection rules that referenced it.
- Self-hosted runner labels have changed; update any workflow `runs-on:` references accordingly.

#### Terraform Prerequisites
- **Terragrunt** must be installed (`terragrunt >= 0.55`).
- Backend configuration files (`backend_azure.hcl` / `backend_local.hcl`) must be populated before `terraform init`.
- The new `azure_gh_runner` module creates an Azure App Registration — ensure the deploying principal has `Application.ReadWrite.OwnedBy` or equivalent AAD permission.

#### ISO Path Change
- Windows 11 ISO is now cached under `./cache/windows11/iso/` (was `vendor/windows/`).
- Update any local scripts or documentation that reference the old path.

#### Hyper-V Windows Provisioning Inventory Removal (Migration Required)
- `inventory/hyperv_windows_winrm.yml` is no longer created or updated by the project.
- Old workflow: scripts/runbooks that invoked `ansible-playbook` with `-i inventory/hyperv_windows_winrm.yml` must be updated.
- New workflow: run the provisioning wrapper from repo root:
  - `go run cmd/main.go provision windows11 --arch amd64 --check`
  - `go run cmd/main.go provision windows11 --arch amd64`
- Required credentials are now read from environment (or project-root `.env`):
  - `HYPERV_WINDOWS_ANSIBLE_USER`
  - `HYPERV_WINDOWS_ANSIBLE_PASSWORD`
- Optional connection overrides:
  - `HYPERV_WINDOWS_ANSIBLE_CONNECTION`
  - `HYPERV_WINDOWS_ANSIBLE_WINRM_TRANSPORT`
  - `HYPERV_WINDOWS_ANSIBLE_PORT`

---

### Dependencies

- Python: `playwright`, `playwright-stealth` (updated version) added to Function App and dependency reconciliation scripts.
- Go: no new direct dependencies; vendor directory kept in sync.
- Terraform providers: `hashicorp/azurerm`, `hashicorp/random` managed by Terragrunt root.
- Packer plugins: managed by `packer init`; ensure network access to the Packer plugin registry on first run.

> **Recommendation:** Add Dependabot configuration for GitHub Actions, Go modules, and Python `requirements.txt` to automate future dependency updates.
