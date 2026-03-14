# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0](https://github.com/csautter/dev-alchemy/compare/v0.1.1...v0.2.0) (2026-03-14)


### Added

* added windows build on windows to go wrapper ([6d87a37](https://github.com/csautter/dev-alchemy/commit/6d87a3762e25215a326a0a6a0c8d8563143f4d48))
* automatically handle max cpu count for vm configs ([d0c76b7](https://github.com/csautter/dev-alchemy/commit/d0c76b7cbdddf5f25aa5bb7cc2f154856fd55aa7))
* **build:** propagate dynamic CPU and memory to macOS QEMU builds ([d704748](https://github.com/csautter/dev-alchemy/commit/d7047484e955c4dd0e72bb2bfbb77272c994b77c))
* cache windows iso file to azure blob storage ([b065849](https://github.com/csautter/dev-alchemy/commit/b065849b17418d4bd4f5ac6207f6933a87a2af63))
* **cache:** add local runner cache save after blob download in PowerShell script ([48a03ec](https://github.com/csautter/dev-alchemy/commit/48a03ecb68514a6fdb8a5926ec8cba04dfb6b48f))
* check if windows iso can be downloaded from azure blob storage ([dea90ad](https://github.com/csautter/dev-alchemy/commit/dea90ad63fdeeeb0e4c45503966acf06cd955ac2))
* **ci:** added runner iso cache dir ([4a1e79c](https://github.com/csautter/dev-alchemy/commit/4a1e79c1effbe5d481312f01de5f72ca679901e6))
* **ci:** replace iso-cache with generic build-cache for ISOs and other dependencies ([e59656a](https://github.com/csautter/dev-alchemy/commit/e59656a84468c3d7fd081facb2621a55db30a3b4))
* **ci:** use azure vm temp disk for builds ([67ae62a](https://github.com/csautter/dev-alchemy/commit/67ae62aa1548eb8c159186540f3e0fb3fb6f9aac))
* **deploy:** add Windows Hyper-V create flow and unify deploy command execution ([8e2836c](https://github.com/csautter/dev-alchemy/commit/8e2836c4e25630a0233a08da8770487ae5972f59))
* **deploy:** add Windows Hyper-V vagrant deploy and unify deploy command runner ([0a3f7be](https://github.com/csautter/dev-alchemy/commit/0a3f7bed39a68b1e849ee78bcf4a667fb5441adf))
* gh runner vm set default switch ([cfccd43](https://github.com/csautter/dev-alchemy/commit/cfccd4305350011b6c4758137c9ba53a24eb023e))
* macos tart runner script - create ephemeral runners in a loop ([5c26a00](https://github.com/csautter/dev-alchemy/commit/5c26a001a2845d321c2d657699a2f257e85a5828))
* **provision:** add unified VM provision command for Hyper-V Windows ([f1063ba](https://github.com/csautter/dev-alchemy/commit/f1063bade3e5c5b85da8d250ed75192465dd3a0b))
* **provision:** add unified VM provision command with Hyper-V Windows flow via Cygwin ([177c979](https://github.com/csautter/dev-alchemy/commit/177c979171d01e1fc4086ef7b69da9e88141067f))
* **runners:** add parallel runner pool and VM CPU/memory configuration ([a444551](https://github.com/csautter/dev-alchemy/commit/a44455184eb619a05c77e1338569189f17d1aac2))
* script for creating macos gh runners with tart ([422f884](https://github.com/csautter/dev-alchemy/commit/422f884a90c4876cf6d84a0c3961364c3550b410))


### Fixed

* add sudo check as prerequisite for test ([7f6c74b](https://github.com/csautter/dev-alchemy/commit/7f6c74b94e40987e2d709724946681bb6690682b))
* added missing windows iso dependency ([4b12461](https://github.com/csautter/dev-alchemy/commit/4b12461f3fb0c7d833d6487e5a8f75a2801439a0))
* **auth:** add OAuth2 scope and pre-authorize Azure CLI on app registration ([cd054b9](https://github.com/csautter/dev-alchemy/commit/cd054b950565335b892e2017a72ecb1757b64ae3))
* **auth:** harden function app auth guard and remove dead key auth code ([603dacf](https://github.com/csautter/dev-alchemy/commit/603dacfbdf50fbd389d942ce137dfdfbbe08c527))
* await context.cookies() in playwright_win11_iso.py ([66e96dc](https://github.com/csautter/dev-alchemy/commit/66e96dc788f9611ea0b3a79317c1766a0d606dd0))
* az vm gh runner image id ([28551a8](https://github.com/csautter/dev-alchemy/commit/28551a87ccd26cb4ad7b4455cb73d5019e293b8c))
* azure hyper-v github integration ([2e8ee76](https://github.com/csautter/dev-alchemy/commit/2e8ee763cda7396f154c7900f4d7f7a18eeb1387))
* **build:** dependency download bar speed calculation ([1cd7da5](https://github.com/csautter/dev-alchemy/commit/1cd7da5ef213fc16904b697340457eaa5ddc56b7))
* **build:** propagate dependency bootstrap failures immediately ([2f801f9](https://github.com/csautter/dev-alchemy/commit/2f801f9ce2d8b739cf69184ca04b88c8dd41339d))
* **build:** propagate packer init errors in Windows build path ([f1cf041](https://github.com/csautter/dev-alchemy/commit/f1cf0415c9de7d01827f0bbc2763cfd10d171928))
* **build:** remove duplicate cmd.Wait() causing spurious "wait: no child processes" failure ([0c3e93c](https://github.com/csautter/dev-alchemy/commit/0c3e93c09c30aa93be601fac08a525ec719b6223))
* **build:** removed de keymap from qemu args to avoid wrong inputs in boot_command ([00b924f](https://github.com/csautter/dev-alchemy/commit/00b924f262bc3797b52d71e5c566b204ce1b86ba))
* **cache:** removed special chars to ensure powershell compatibility ([58a1908](https://github.com/csautter/dev-alchemy/commit/58a19080376e4dcfabceebc7a2194f3d2470410c))
* **ci:** added go mod vendor before go test ([a6586fe](https://github.com/csautter/dev-alchemy/commit/a6586fe478b95b041a512df8ca582b4ea8cdf936))
* **ci:** added iso-path output to download action ([fcee7bc](https://github.com/csautter/dev-alchemy/commit/fcee7bcfd077ce98604c42bd42e97987d655c007))
* **ci:** added permissions for win iso blob uploads ([6d36c0f](https://github.com/csautter/dev-alchemy/commit/6d36c0f68bcb2639536a9ef27c12d803ff2e0b42))
* **ci:** added safe remove of tart ephemeral runners ([326b00b](https://github.com/csautter/dev-alchemy/commit/326b00bfe62fada7d9e6cbd8926dd81e8ea2ade9))
* **ci:** blob up and download - use iso-path filename ([74a1cf8](https://github.com/csautter/dev-alchemy/commit/74a1cf82e1fcbbdf1e2034b787d8ecc0d9814ab7))
* **ci:** check if file exists before upload to az blob storage ([864787e](https://github.com/csautter/dev-alchemy/commit/864787e0cc60624fcf8aef6402b4efaeb21b710d))
* **ci:** continue on windows iso from az download error ([5ee61eb](https://github.com/csautter/dev-alchemy/commit/5ee61eb6fc76ce7102d921f6c9eaacbd0a7732b9))
* **ci:** download iso from az blob - don't mask issues ([dbf8f49](https://github.com/csautter/dev-alchemy/commit/dbf8f498ef44f13a8842a44f665fcfa939df478f))
* **ci:** go test run - use package path not file ([3d41b7e](https://github.com/csautter/dev-alchemy/commit/3d41b7e3514e4040852874ecdcde5cf6a534c009))
* **ci:** macos runner - include homebrew installed binaries in PATH ([14e2b14](https://github.com/csautter/dev-alchemy/commit/14e2b144a7f9f10225e069cd5ffdb34966e862ee))
* **ci:** macos tart iso caching - fixed cache volume path ([125f89a](https://github.com/csautter/dev-alchemy/commit/125f89a97896322e7411fe51b1dfeaf430b50df4))
* **ci:** removed az blob storage creation in gh actions ([1a8a62a](https://github.com/csautter/dev-alchemy/commit/1a8a62a729f0b7747154793727acb3b206fbe3b6))
* **ci:** run go build tests with sudo ([0dfde0a](https://github.com/csautter/dev-alchemy/commit/0dfde0a695a2e879c4d05f26dc1b7dd05241094e))
* **ci:** shortened runner name ([17cefc5](https://github.com/csautter/dev-alchemy/commit/17cefc502886d753b415acb88530dbb6345561cc))
* **ci:** updated win11 packer temp path ([fd6b6ff](https://github.com/csautter/dev-alchemy/commit/fd6b6ff61d9f3b32d375d8c7f4687a717256ddad))
* **ci:** updated workflow trigger path ([36befeb](https://github.com/csautter/dev-alchemy/commit/36befebafcf11c03ddef4b8ac0fd257451b997c7))
* **ci:** win11 packer build - increased os memory headroom ([87c8af6](https://github.com/csautter/dev-alchemy/commit/87c8af65f99cf7e8e40fb5bddad1d735648a2c69))
* collapse multiline python -c scripts to fix YAML block scalar parsing ([1f5ec78](https://github.com/csautter/dev-alchemy/commit/1f5ec78cb7246978b0d6ce3c6ad1b395cf94a79a))
* **dependencies:** install playwright stealth for reconciliation ([22d406c](https://github.com/csautter/dev-alchemy/commit/22d406ca5ba6b2a863e3e9af06f48eed39826a7f))
* **dependencies:** reverted playwright win11 download approach ([23345d3](https://github.com/csautter/dev-alchemy/commit/23345d38440a1fb154d69f6fd48639cbba7a443d))
* **deploy:** fail fast when Cygwin bash is missing on Windows provisioning ([c34f0cc](https://github.com/csautter/dev-alchemy/commit/c34f0cc74b15c35b1d1574dba2189f663e10fdd1))
* **deploy:** prevent truncated streamed logs and surface scanner errors ([7724f4a](https://github.com/csautter/dev-alchemy/commit/7724f4a91bad9cda12bd89883fec822077a52f3e))
* **deploy:** remove windows build constraints from hyperv deploy file ([5c86ebe](https://github.com/csautter/dev-alchemy/commit/5c86ebe52e85f7bbf5dce9cca867a8c4ca07e04f))
* **deploy:** return command execution errors instead of panicking ([9d881bf](https://github.com/csautter/dev-alchemy/commit/9d881bfc6344015ce512068e780d07ca9fbaae9e))
* handle empty cookies.json by catching JSONDecodeError ([477fe1d](https://github.com/csautter/dev-alchemy/commit/477fe1dce64b5bbf1ab248a1838b0e304b6e744e))
* hyper-v setup handle existing resources ([6a3370f](https://github.com/csautter/dev-alchemy/commit/6a3370f08d8421517745a0d0869d64f57922e72e))
* hyper-v setup restart order ([965d6f0](https://github.com/csautter/dev-alchemy/commit/965d6f0cc54a52e889797dba000da8feabca5598))
* hyper-v windows11 boot_command timing ([1b6966e](https://github.com/csautter/dev-alchemy/commit/1b6966eb34da36b9271cbef1ae64f5707812a25e))
* **hyperv:** retry early build failures to handle transient Default Switch IP race ([e115bde](https://github.com/csautter/dev-alchemy/commit/e115bde82fbd4a07e3c9413964da4151a23dd465))
* limit az vm name to 15 chars ([3efcb43](https://github.com/csautter/dev-alchemy/commit/3efcb439031f4b5360676598137389024f07472f))
* listener bar null check ([25a688a](https://github.com/csautter/dev-alchemy/commit/25a688ab32b5949e05bc0b83445d802f68de3a95))
* make vncsnapshot retry loop interruptible by SIGINT/SIGTERM ([a18eb8d](https://github.com/csautter/dev-alchemy/commit/a18eb8df7aaf750b9d175243c5d428f315b6c1cf))
* pass cache files JSON via env vars to avoid PowerShell string parsing errors ([24a6e73](https://github.com/csautter/dev-alchemy/commit/24a6e73eacb02c8893e4d5dd70866109bd12a64a))
* pass vnc_recording_config by reference ([3c0129f](https://github.com/csautter/dev-alchemy/commit/3c0129f93adb545f4a4d24ff0e7bbeddb43550b9))
* power shell log file encoding ([3b53a6c](https://github.com/csautter/dev-alchemy/commit/3b53a6cde7f2c11fee670bd2f18b45142d52c5d1))
* prevent deadlock in stopVncScreenCaptureOnMacosDarwin when vncsnapshot exits cleanly ([784183b](https://github.com/csautter/dev-alchemy/commit/784183bd0968f11f8d67ac428968f1183a5382eb))
* prevent deadlock in stopVncScreenCaptureOnMacosDarwin when vncsnapshot exits cleanly ([9c2ca4f](https://github.com/csautter/dev-alchemy/commit/9c2ca4faecf2f783bd28e56ca86c3f9d54e557ff))
* reboot after hyper-v install on azure instance ([1a9112f](https://github.com/csautter/dev-alchemy/commit/1a9112fd8e4fded6a94375232fec901caf4e97a3))
* replace powershell backtick continuations with argument splatting in cache actions ([cabe95b](https://github.com/csautter/dev-alchemy/commit/cabe95b232b7ac0f6c01c39bdb7a74dbc81f2ebc))
* secured az function app endpoint ([e40ff9b](https://github.com/csautter/dev-alchemy/commit/e40ff9b72d9369d5f951e977820aed7791e2527a))
* secured az function app endpoint - allowed access by az cli ([18fa192](https://github.com/csautter/dev-alchemy/commit/18fa192858a156d57bf10c71ac2ac662a4a32246))
* **security:** validate inputs and prevent error detail leakage in function app ([240c019](https://github.com/csautter/dev-alchemy/commit/240c0197f51bf458c07d8f6bac3fe994f4e7e56a))
* set iso path depending on architecture ([2eb049a](https://github.com/csautter/dev-alchemy/commit/2eb049aa134ab95e4d2e0e1ee8d1b10386f54560))
* set max cpu cores to 2 ([d85f99c](https://github.com/csautter/dev-alchemy/commit/d85f99c44154997d3d8e86f10346d0a449a9f18b))
* stop re-rendering completed progress bars ([d5d4ee8](https://github.com/csautter/dev-alchemy/commit/d5d4ee8355d00c5e63471be3b77ae377782de7bf))
* **tart-runner:** recover worker on SSH/VM failure instead of exiting ([f629934](https://github.com/csautter/dev-alchemy/commit/f629934d57d6563449ab9bfc8432586c0b595cfb))
* **terraform:** restore random suffix range with stable lifecycle guard ([fefcd44](https://github.com/csautter/dev-alchemy/commit/fefcd4479d80e584e139adb179ab7a3519a67de8))
* test utm build on darwin - added engine config ([6edefaf](https://github.com/csautter/dev-alchemy/commit/6edefafc242bf01f13d1e7d35a8d2fac1d61d4a6))
* update downloadWebFileDependency call in test to pass progress argument ([c6a8cfc](https://github.com/csautter/dev-alchemy/commit/c6a8cfccc8d8b94c9e8277a738db05d17f901974))
* use bash 3.2-compatible array append for WORKER_PIDS ([b4db84f](https://github.com/csautter/dev-alchemy/commit/b4db84f53943c23a44930daa791ec700d2c0c510))
* **utm:** set arm64 cpu type to default to fix issues with vm boot ([0773b0f](https://github.com/csautter/dev-alchemy/commit/0773b0f61a828e7ddb191abba60ad9028ed5c380))
* vbox build winrm_timeout increased ([ea38edc](https://github.com/csautter/dev-alchemy/commit/ea38edc6fd17af10a27ac448eae99283db85a73e))
* win11 iso download func - check if file exists and skip ([02a1f67](https://github.com/csautter/dev-alchemy/commit/02a1f679a6cc89091c567cc3bb74352e417ed832))
* **windows11:** fix arm64 QEMU boot hang and headless mode handling ([a9bb0ad](https://github.com/csautter/dev-alchemy/commit/a9bb0ad3f250a451515f4e31d864d8cbced622c9))


### Changed

* **actions:** extract inline scripts into separate files for download/upload-build-cache ([0d40c17](https://github.com/csautter/dev-alchemy/commit/0d40c173fd6d9d165de51c3b60a8107ae53d6ab2))
* **build:** removed windows isos az storage container ([5b7843b](https://github.com/csautter/dev-alchemy/commit/5b7843bd1b2473606a18e9e01dd57aa3106cee93))
* changed windows iso path to ./cache dir ([e06c7f1](https://github.com/csautter/dev-alchemy/commit/e06c7f1a4a7863d8e363700a5437a15960e34195))
* **ci:** bundled blob up and down actions ([fd9707b](https://github.com/csautter/dev-alchemy/commit/fd9707b7be43d8b572dcbb0167e76de767b313b8))
* **ci:** optimized job dependencies for faster az runner cleanup ([87dbf02](https://github.com/csautter/dev-alchemy/commit/87dbf02e49687f532dea9148e22ed7542f4a908e))
* **deploy:** replace static winrm inventory with env-var-driven ansible config ([c87bf2b](https://github.com/csautter/dev-alchemy/commit/c87bf2bf388ddb2eb814a7b063f3c57a93bcf372))
* extracted powershell script from python code ([b33545d](https://github.com/csautter/dev-alchemy/commit/b33545d3d6fb8a5801db0c6ff2343f24590f26bc))
* gh actions - reusable iso up and download configs ([dcbedc9](https://github.com/csautter/dev-alchemy/commit/dcbedc93157dc967b23490b4f372e2e8e4da1898))
* hyper-v setup ([16f2025](https://github.com/csautter/dev-alchemy/commit/16f20255b510d78dc654b8f8c0d573cb5120142a))
* install gh runner in golden image for macos ([72b11d3](https://github.com/csautter/dev-alchemy/commit/72b11d34b64ecc6224ba088153c66b4b4a379d08))
* **provision:** reject "all" target with explicit error ([d94c306](https://github.com/csautter/dev-alchemy/commit/d94c306270fd166f3c8f9eb512a0c78b51c7b537))
* relocate build artifact paths from /vendor to /cache ([c7de8fa](https://github.com/csautter/dev-alchemy/commit/c7de8fa0b7700927ec50bd037bdee4f550fb4256))
* resolve VirtualMachineConfig from available list instead of constructing manually ([f9730b4](https://github.com/csautter/dev-alchemy/commit/f9730b432b2cbb0dd14c46d0caed64ffc8c1b599))
* tart runner prepare script simplified ([b626a17](https://github.com/csautter/dev-alchemy/commit/b626a17e309ea679727e62159bccd1c91c0d9dd0))


### CI

* **actions:** add workflow concurrency to prevent push/pr parallel runs ([f5498b1](https://github.com/csautter/dev-alchemy/commit/f5498b13a9b2c4a7be3c0972efcd3fae2383589b))
* **actions:** add workflow concurrency to prevent push/pr parallel runs ([b875e69](https://github.com/csautter/dev-alchemy/commit/b875e690efa3d7e1591c5bb10ddff2efcd2c75a1))
* add Go security scanning workflow (govulncheck + gosec) ([c786bd5](https://github.com/csautter/dev-alchemy/commit/c786bd55110db26b5599102bb65e45af6d172939))
* add Makefile target and GitHub Actions workflow for build-runner tests ([4d7d305](https://github.com/csautter/dev-alchemy/commit/4d7d305b510f72a81ef09504318dadcdca24b622))
* configure release-please for automated releases ([95e3fd7](https://github.com/csautter/dev-alchemy/commit/95e3fd762e6f811c5ad7b918ffeed31daa0b6d1d))
* fix unit-test workflow path filters to self-reference current file ([0849419](https://github.com/csautter/dev-alchemy/commit/084941958fb36d505ab278cdea9d41e021c576b8))
* **security-scan:** don't checkout the code a second time ([b65153b](https://github.com/csautter/dev-alchemy/commit/b65153b4c7a793e846277491c5327ce6a85a9849))
* **security:** prevent gosec from failing workflow and gitignore sarif results ([e8e24e1](https://github.com/csautter/dev-alchemy/commit/e8e24e10491307f3f62c6d50fe8b2f94468a8738))


### Miscellaneous

* release 0.2.0 ([032ec53](https://github.com/csautter/dev-alchemy/commit/032ec53c14b2b687ef1cd4e6e70e871be14d7e88))

## [Unreleased]

## [v0.2.0] - 2026-03-15

> Commits `bdbed922..c9927fb5` · 2025-11-23 → 2026-02-22
>
> 145 commits · 82 files changed · +5033 / -704

---

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

---

[Unreleased]: https://github.com/csautter/dev-alchemy/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/csautter/dev-alchemy/compare/v0.1.1...v0.2.0
