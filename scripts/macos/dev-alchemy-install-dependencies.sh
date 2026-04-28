#!/usr/bin/env bash

set -ex

# Renovate-managed version pins
# renovate: datasource=custom.hashicorp depName=packer packageName=packer versioning=semver
PACKER_VERSION="1.15.3"
# renovate: datasource=custom.homebrew-formula depName=go packageName=go versioning=loose
GO_VERSION="1.26.2"
# renovate: datasource=custom.homebrew-cask depName=utm packageName=utm versioning=loose
UTM_VERSION="4.7.5"
# renovate: datasource=custom.homebrew-formula depName=qemu packageName=qemu versioning=loose
QEMU_VERSION="11.0.0"
# renovate: datasource=custom.homebrew-formula depName=xz packageName=xz versioning=loose
XZ_VERSION="5.8.3"
# renovate: datasource=custom.homebrew-formula depName=ffmpeg packageName=ffmpeg versioning=loose
FFMPEG_VERSION="8.1"
# renovate: datasource=custom.homebrew-formula depName=vncsnapshot packageName=vncsnapshot versioning=loose
VNCSNAPSHOT_VERSION="1.2a"
# renovate: datasource=custom.homebrew-formula depName=xorriso packageName=xorriso versioning=loose
XORRISO_VERSION="1.5.8.pl01"
# renovate: datasource=custom.homebrew-formula depName=ansible packageName=ansible versioning=loose
ANSIBLE_VERSION="13.5.0"

install_go="false"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--with-go)
		install_go="true"
		shift
		;;
	*)
		echo "Unknown option: $1" >&2
		exit 1
		;;
	esac
done

brew_install() {
	local label="$1" cmd="$2" pkg="$3" version="$4" tap="${5:-}" flags="${6:-}"
	local list_pkg="${pkg##*/}"
	local installed_version
	local -a install_args=("${pkg}")
	local -a list_flags=()

	if [[ -n "${tap}" ]]; then
		brew tap "${tap}" || echo "WARNING: tap ${tap} failed, continuing..."
	fi

	if [[ " ${flags} " == *" --cask "* ]]; then
		install_args=(--cask "${pkg}")
		list_flags=(--cask)
	fi

	installed_version="$(brew list "${list_flags[@]}" --versions "${list_pkg}" 2>/dev/null | awk 'NR == 1 { print $2 }')"

	if ! brew_version_matches "${installed_version}" "${version}" "${flags}"; then
		if [[ -n "${installed_version}" ]]; then
			echo "${label} version ${installed_version} does not match pinned ${version}; updating via Homebrew..."
			if [[ " ${flags} " != *" --cask "* ]]; then
				brew unpin "${list_pkg}" || true
			fi
			brew upgrade "${install_args[@]}" || brew install "${install_args[@]}"
		else
			echo "Installing ${label} ${version}..."
			brew install "${install_args[@]}"
		fi

		installed_version="$(brew list "${list_flags[@]}" --versions "${list_pkg}" 2>/dev/null | awk 'NR == 1 { print $2 }')"
	fi

	if ! brew_version_matches "${installed_version}" "${version}" "${flags}"; then
		echo "ERROR: ${label} installed version ${installed_version:-<missing>} does not match pinned ${version}." >&2
		echo "       Update the Renovate-managed pin or ensure Homebrew can provide that version." >&2
		exit 1
	fi

	if [[ "${cmd}" == /* ]]; then
		if [[ ! -e "${cmd}" ]]; then
			echo "ERROR: ${label} installed at pinned version ${version}, but expected path ${cmd} was not found." >&2
			exit 1
		fi
	elif ! command -v "${cmd}" &>/dev/null; then
		echo "ERROR: ${label} installed at pinned version ${version}, but expected command ${cmd} was not found." >&2
		exit 1
	fi

	if [[ " ${flags} " != *" --cask "* ]]; then
		brew pin "${list_pkg}" || true
		echo "${label} ${installed_version} installed and pinned (matches ${version})."
	else
		echo "${label} ${installed_version} installed and verified against pin ${version}."
	fi
}

brew_version_matches() {
	local installed="$1" pinned="$2" flags="${3:-}" revision

	if [[ "${installed}" == "${pinned}" ]]; then
		return 0
	fi

	if [[ " ${flags} " != *" --cask "* && "${installed}" == "${pinned}_"* ]]; then
		revision="${installed#"${pinned}_"}"
		[[ "${revision}" =~ ^[0-9]+$ ]] && return 0
	fi

	return 1
}

brew_install "Packer"      packer                    hashicorp/tap/packer  "${PACKER_VERSION}"       hashicorp/tap
brew_install "UTM"         /Applications/UTM.app     utm                   "${UTM_VERSION}"          ""  --cask
brew_install "QEMU"        qemu-img                  qemu                  "${QEMU_VERSION}"

brew_install "xz"          xz                        xz                    "${XZ_VERSION}"
brew_install "FFmpeg"      ffmpeg                    ffmpeg                "${FFMPEG_VERSION}"
brew_install "vncsnapshot" vncsnapshot               vncsnapshot           "${VNCSNAPSHOT_VERSION}"
brew_install "xorriso"     xorriso                   xorriso               "${XORRISO_VERSION}"
brew_install "Ansible"     ansible                   ansible               "${ANSIBLE_VERSION}"

if [[ "${install_go}" == "true" ]]; then
	brew_install "Go" go go "${GO_VERSION}"
fi
