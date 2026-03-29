# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.1](https://github.com/csautter/dev-alchemy/compare/v0.9.0...v0.9.1) (2026-03-29)


### Fixed

* **ci:** specify repository for gh release upload ([f693ae1](https://github.com/csautter/dev-alchemy/commit/f693ae1277228050c7df9f9f64d9dca98f7addc7))

## [0.9.0](https://github.com/csautter/dev-alchemy/compare/v0.8.0...v0.9.0) (2026-03-29)


### Added

* embed runtime assets for standalone CLI builds and release binaries ([b8d7506](https://github.com/csautter/dev-alchemy/commit/b8d7506db767d1f1ce00cc01cf0981329869b43c))
* embed runtime assets for standalone CLI builds and release binaries ([af5d159](https://github.com/csautter/dev-alchemy/commit/af5d159650060283e88d6d6ae908d359b2e1421c))


### Fixed

* **build:** harden embedded project extraction against gosec findings ([b850868](https://github.com/csautter/dev-alchemy/commit/b8508683e7233e3e2d5e246a6aa3048c43cf179f))
* **build:** stop external process retries immediately on interrupt ([24bf8da](https://github.com/csautter/dev-alchemy/commit/24bf8da8c02cfaa26d1c9b0cbbe54792d9c92296))
* **build:** write Windows release zip to dist for artifact upload ([ca05356](https://github.com/csautter/dev-alchemy/commit/ca0535682c2dc7dc0c8feebe881a0773c8f58b96))
* **deps:** upgrade google.golang.org/grpc to v1.79.3 for GO-2026-4762 ([9942bd7](https://github.com/csautter/dev-alchemy/commit/9942bd7327db700b97c6da12b078f08bd8a3e02f))


### CI

* run release binary build dry-runs in pull requests ([4a0a132](https://github.com/csautter/dev-alchemy/commit/4a0a1320dd3291a3ab5e9da8af0abdb4f6be9884))
* store Hyper-V diagnostics in private Azure cach ([9e4d904](https://github.com/csautter/dev-alchemy/commit/9e4d9045c1fab0d301f12b8e1598e01d47d42206))

## [0.8.0](https://github.com/csautter/dev-alchemy/compare/v0.7.0...v0.8.0) (2026-03-29)


### Added

* enhaned debugging for hyperv runner ([90ce346](https://github.com/csautter/dev-alchemy/commit/90ce3464f342f01a66b988f0f54c856aac991ef2))


### Fixed

* **build:** add engine selection and mark virtualbox unstable ([8434021](https://github.com/csautter/dev-alchemy/commit/8434021c904b737eb38826c98b84dfbdea48a12a))
* **build:** exclude unstable virtualbox target from build all by default ([0876478](https://github.com/csautter/dev-alchemy/commit/087647888cd14ce7d18115c6577bce492ce7185f))
* **build:** restrict managed app directory permissions to 0700 ([2bc4f7d](https://github.com/csautter/dev-alchemy/commit/2bc4f7de660dd7112ce6bf85a5ed759c916a8c65))
* **build:** shorten macos qemu output paths and stop noisy vnc retries ([20354ec](https://github.com/csautter/dev-alchemy/commit/20354ec011cd5d5d394e091e622061f1814971d8))
* **ci:** align macOS workflow cache paths with managed app data ([0e08270](https://github.com/csautter/dev-alchemy/commit/0e082708048154c32f3b4e0e91de756805f04aba))
* **ci:** align Windows workflow cache paths with managed app data ([c0e419a](https://github.com/csautter/dev-alchemy/commit/c0e419a43d7d0de214d07fc36733ac415a89d8a0))
* **ci:** capture sanitized Hyper-V diagnostics for Windows build failures ([bfb7246](https://github.com/csautter/dev-alchemy/commit/bfb72463ccb442f26980fb00b1c3996d45395571))
* **ci:** harden Playwright Windows ISO fetch and upload diagnostics ([99b0cf3](https://github.com/csautter/dev-alchemy/commit/99b0cf390348d4d4c09ee48165273a0e0c1ba099))
* **packer:** use valid validation messages for cache_dir ([aa924f4](https://github.com/csautter/dev-alchemy/commit/aa924f426b3f23c29ec46ef31797c53e9a062dde))


### Changed

* move VM state into managed app data ([9697909](https://github.com/csautter/dev-alchemy/commit/9697909783a3b774a782a3706e17dfb52261b437))
* move VM state into managed app data ([1077820](https://github.com/csautter/dev-alchemy/commit/1077820edb0fcdb1e7e72f5a57134c13420bf4b9))


### CI

* refresh Azure auth before HyperV queue cleanup ([9189f40](https://github.com/csautter/dev-alchemy/commit/9189f400c91c24a464e8c0c8ef6cd76823f8c00a))
* run deploy and provision unit tests on main pushes ([23dc24f](https://github.com/csautter/dev-alchemy/commit/23dc24ffff8f76e7da0ccde7d6dad99d063181c9))
* run gitleaks and cmd unit tests on main pushes ([4db3625](https://github.com/csautter/dev-alchemy/commit/4db3625756827014e0a124f8b4594b6c8c1d1f6c))

## [0.7.0](https://github.com/csautter/dev-alchemy/compare/v0.6.0...v0.7.0) (2026-03-28)


### Added

* **cli:** add destroy command for managed VMs ([490901b](https://github.com/csautter/dev-alchemy/commit/490901bea7816f307a477889b29ff8017ac2ae53))
* **create:** detect existing VM targets before deploy ([5ab8087](https://github.com/csautter/dev-alchemy/commit/5ab808707132864dd736b0f6722c25371d7ec334))
* **destroy:** add VM destroy readiness listing and tart state fixes ([05e3175](https://github.com/csautter/dev-alchemy/commit/05e31751954c7905c98045fd6971eda461eb9686))
* **vm:** add lifecycle commands for starting, stopping, and destroying managed VMs ([529adfa](https://github.com/csautter/dev-alchemy/commit/529adfa3ca28c7a45377cefc18217bfa5f1fa6fd))
* **vm:** add start command and fail-fast provision preflight ([1cb7a7a](https://github.com/csautter/dev-alchemy/commit/1cb7a7a593827bfe9b1deba4bab43b2a12bc2223))
* **vm:** add stop command with graceful UTM shutdown ([79b5689](https://github.com/csautter/dev-alchemy/commit/79b568916c4a5a0924d1519425fe2dfda2313bed))


### Fixed

* **deploy:** harden Hyper-V Vagrant stop with forced halt fallback ([fb75cdb](https://github.com/csautter/dev-alchemy/commit/fb75cdbffacf72353ec50cf56d5c94764ba2bfdb))
* **deploy:** infer canonical UTM targets for macOS deploy configs ([893d616](https://github.com/csautter/dev-alchemy/commit/893d616cace68f17fcdff80145f5c30dad94f11a))
* **deploy:** inspect Hyper-V VM state via PowerShell ([21bd7da](https://github.com/csautter/dev-alchemy/commit/21bd7da2d964006ce4b6c2af87befcc969393f7e))
* **deploy:** make Hyper-V stop timeout boundary deterministic ([36f9499](https://github.com/csautter/dev-alchemy/commit/36f9499547f7e20eefc0dad6cbea616a33f21ffb))
* **deploy:** normalize Hyper-V Vagrant dotfile env paths on Windows ([0aec0d9](https://github.com/csautter/dev-alchemy/commit/0aec0d9f69dc696c0475e110f83223f511b6d5ea))
* **hyperv:** isolate vagrant state per vm and prevent false create detection ([eabb917](https://github.com/csautter/dev-alchemy/commit/eabb9172a7f78034b64b8b886c7e8890c079c6ca))
* **provision:** use VM-specific Hyper-V Vagrant settings ([5c78d8d](https://github.com/csautter/dev-alchemy/commit/5c78d8dd7a425459dd70c863ae82151d4a6ecb40))


### Changed

* **provision:** move VM provisioning logic out of pkg/deploy ([f6cc593](https://github.com/csautter/dev-alchemy/commit/f6cc5937910ba70932123f90a4b88a30989bb37e))


### CI

* add PR coverage for deploy and provision tests ([33dd2a3](https://github.com/csautter/dev-alchemy/commit/33dd2a316600ad176c3501ff605e382fc638549d))

## [0.6.0](https://github.com/csautter/dev-alchemy/compare/v0.5.0...v0.6.0) (2026-03-24)


### Added

* **tart:** add macOS Tart target for create and provision ([e721799](https://github.com/csautter/dev-alchemy/commit/e7217997e0b7672d81347202d7462f454034b905))
* **tart:** add macOS Tart target for create and provision ([e0203c9](https://github.com/csautter/dev-alchemy/commit/e0203c9bb1624ebee751fc0502eb535ec9feef12))


### Fixed

* **ansible:** resolve macOS Java homes on target before jenv add ([a408ef2](https://github.com/csautter/dev-alchemy/commit/a408ef24ee458ceb8f85807d9d4c19d50aa45b18))
* **deploy:** address gosec findings in tart deploy helpers ([b564d3f](https://github.com/csautter/dev-alchemy/commit/b564d3f28cb6c90bc658b6c199dfd7e6f3380d6c))
* **deploy:** handle existing running Tart macOS VMs explicitly ([6ee6e8c](https://github.com/csautter/dev-alchemy/commit/6ee6e8ce06dc4edd1157a61fb4567277ff1491dc))
* **deploy:** parse tart list output by columns for local VM lookup ([715b704](https://github.com/csautter/dev-alchemy/commit/715b70498ed060c91442d635111f23deda6929a9))


### Changed

* **deploy:** extract shared SSH wait helper ([35d32cd](https://github.com/csautter/dev-alchemy/commit/35d32cd503f4733f8692bc88c09b2fccde43f54b))
* rename shared SSH provision arg builder ([f1b0629](https://github.com/csautter/dev-alchemy/commit/f1b0629a6949361a3034d6e255e33c30e2aef729))

## [0.5.0](https://github.com/csautter/dev-alchemy/compare/v0.4.0...v0.5.0) (2026-03-23)


### Added

* **provision:** add Ubuntu UTM provisioning support on macOS ([33dace4](https://github.com/csautter/dev-alchemy/commit/33dace45b6b2e9b32251c32a29f5cf97f8308884))
* **provision:** add Ubuntu UTM provisioning support on macOS ([cc83f38](https://github.com/csautter/dev-alchemy/commit/cc83f381030f2e95fd42a10894faa98375bbee0f))


### Fixed

* **actions:** simplify Hyper-V queue watcher job parsing ([4bbaff8](https://github.com/csautter/dev-alchemy/commit/4bbaff841f83280acac627448eb126f4132bd536))
* **ci:** retry HyperV runner cleanup before canceling queued builds ([a4a710c](https://github.com/csautter/dev-alchemy/commit/a4a710cfb89f85d58fd5dab275851d1d41187e35))


### CI

* cancel Windows Hyper-V test runs when the runner stays queued ([30c9852](https://github.com/csautter/dev-alchemy/commit/30c98526f07c44d4a87a20a58b2c140591c6191c))
* scope Windows workflow actions write permission to queue watcher job ([4d3c23e](https://github.com/csautter/dev-alchemy/commit/4d3c23e20c1aa849e19fb82360cb6ea1df1c60a0))
* trigger macOS VM workflow for provision code changes ([907cc21](https://github.com/csautter/dev-alchemy/commit/907cc21a9047316af0dbbeb987871dbb5e61a924))

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
