FROM mcr.microsoft.com/devcontainers/go:2-1.25-trixie

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ansible \
        ca-certificates \
        curl \
        gpg \
        nodejs \
        npm \
        ruby-full \
        yq \
    && rm -rf /var/lib/apt/lists/*

RUN . /etc/os-release \
    && curl -fsSL https://apt.releases.hashicorp.com/gpg | \
        gpg --dearmor --yes -o /usr/share/keyrings/hashicorp-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com ${VERSION_CODENAME} main" \
        > /etc/apt/sources.list.d/hashicorp.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends packer \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/securego/gosec/v2/cmd/gosec@v2.25.0
