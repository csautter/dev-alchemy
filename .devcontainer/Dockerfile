FROM mcr.microsoft.com/devcontainers/go:2-1.26-trixie

ENV DEBIAN_FRONTEND=noninteractive
ENV NODE_VERSION=24.15.0
# renovate: datasource=npm depName=renovate versioning=npm
ENV RENOVATE_VERSION="43.220.0"
# renovate: datasource=custom.powershell-linux-x64 depName=PowerShell/PowerShell versioning=semver
ENV POWERSHELL_VERSION="7.6.2"
ENV POWERSHELL_SHA256_X64="6cbcfbf20e376aa62ffd91c973493c41a7a52ddfd5a5db3ff9bc12f0d0fe9292"
# renovate: datasource=custom.powershell-linux-arm64 depName=PowerShell/PowerShell version=7.6.2 versioning=semver
ENV POWERSHELL_SHA256_ARM64="a8d4e386dfafda385d0604045eed03ce6f3a843d45fc8f0b9588b836ca17cdb8"

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ansible \
        ca-certificates \
        curl \
        gpg \
        ruby-full \
        shellcheck \
        xz-utils \
        yq \
    && rm -rf /var/lib/apt/lists/*

RUN arch="$(dpkg --print-architecture)" \
    && case "${arch}" in \
        amd64) powershell_arch="x64"; powershell_sha256="${POWERSHELL_SHA256_X64}" ;; \
        arm64) powershell_arch="arm64"; powershell_sha256="${POWERSHELL_SHA256_ARM64}" ;; \
        *) echo "Unsupported PowerShell architecture: ${arch}" >&2; exit 1 ;; \
    esac \
    && mkdir -p /opt/microsoft/powershell/7 \
    && curl -fsSL "https://github.com/PowerShell/PowerShell/releases/download/v${POWERSHELL_VERSION}/powershell-${POWERSHELL_VERSION}-linux-${powershell_arch}.tar.gz" -o /tmp/powershell.tar.gz \
    && printf '%s  %s\n' "${powershell_sha256}" /tmp/powershell.tar.gz | sha256sum -c - \
    && tar -xzf /tmp/powershell.tar.gz -C /opt/microsoft/powershell/7 \
    && chmod +x /opt/microsoft/powershell/7/pwsh \
    && ln -sf /opt/microsoft/powershell/7/pwsh /usr/local/bin/pwsh \
    && rm -f /tmp/powershell.tar.gz \
    && pwsh --version

RUN arch="$(dpkg --print-architecture)" \
    && case "${arch}" in \
        amd64) node_arch="x64" ;; \
        arm64) node_arch="arm64" ;; \
        *) echo "Unsupported Node.js architecture: ${arch}" >&2; exit 1 ;; \
    esac \
    && curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node_arch}.tar.xz" -o /tmp/node.tar.xz \
    && rm -rf /usr/local/lib/node_modules /usr/local/include/node /usr/local/bin/node /usr/local/bin/npm /usr/local/bin/npx \
    && tar -xJf /tmp/node.tar.xz -C /usr/local --strip-components=1 \
    && rm -f /tmp/node.tar.xz \
    && node --version \
    && npm --version \
    && npx --version

RUN npm install --global "renovate@${RENOVATE_VERSION}" \
    && renovate-config-validator --version \
    && npm cache clean --force

RUN . /etc/os-release \
    && curl -fsSL https://apt.releases.hashicorp.com/gpg | \
        gpg --dearmor --yes -o /usr/share/keyrings/hashicorp-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com ${VERSION_CODENAME} main" \
        > /etc/apt/sources.list.d/hashicorp.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends packer \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/securego/gosec/v2/cmd/gosec@v2.26.1
