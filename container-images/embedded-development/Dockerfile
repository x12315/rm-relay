# syntax=docker/dockerfile:1.7

FROM alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce AS certificates

FROM ghcr.io/astral-sh/uv:0.11.29@sha256:eb2843a1e56fd9e30c7276ce1a52cba86e64c7b385f5e3279a0e08e02dd058fc AS uv

FROM ubuntu:24.04 AS base

ARG DEBIAN_FRONTEND=noninteractive
ARG TARGETARCH
ARG UBUNTU_MIRROR=https://mirrors.ustc.edu.cn/ubuntu
ARG UBUNTU_PORTS_MIRROR=https://mirrors.ustc.edu.cn/ubuntu-ports

ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    TZ=Asia/Shanghai

COPY container-images/embedded-development/locks/versions.env \
     /opt/embedded-development/versions.env
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt \
     /etc/ssl/certs/ca-certificates.crt

RUN case "${TARGETARCH}" in \
        amd64) selected_ubuntu_mirror="${UBUNTU_MIRROR}" ;; \
        arm64) selected_ubuntu_mirror="${UBUNTU_PORTS_MIRROR}" ;; \
        *) printf 'unsupported TARGETARCH: %s\n' "${TARGETARCH}" >&2; exit 1 ;; \
    esac \
    && sed -i \
       "s|^URIs: .*|URIs: ${selected_ubuntu_mirror}|" \
       /etc/apt/sources.list.d/ubuntu.sources \
    && grep -F 'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg' \
       /etc/apt/sources.list.d/ubuntu.sources \
    && apt-get -o Acquire::Retries=5 -o Acquire::https::Timeout=30 update \
    && apt-get -o Acquire::Retries=5 -o Acquire::https::Timeout=30 \
       install --no-install-recommends -y \
        binutils \
        build-essential \
        ca-certificates \
        clang \
        clang-format \
        clang-tidy \
        cmake \
        curl \
        file \
        g++-14 \
        gcc-14 \
        gdb \
        git \
        jq \
        libclang-rt-18-dev \
        ninja-build \
        python3 \
        python3-venv \
        sudo \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*

COPY --from=uv /uv /uvx /usr/local/bin/
COPY container-images/embedded-development/smoke/ \
     /usr/local/lib/embedded-development/smoke/

RUN chmod +x \
      /usr/local/lib/embedded-development/smoke/verify-base-tools.sh \
      /usr/local/lib/embedded-development/smoke/verify-embedded-tools.sh \
    && /usr/local/lib/embedded-development/smoke/verify-base-tools.sh \
    && dpkg-query --show --showformat='${Package}\t${Version}\n' \
       | sort > /opt/embedded-development/base-packages.txt

WORKDIR /workspace

LABEL org.opencontainers.image.base.name="ubuntu:24.04" \
      dev.embedded.ubuntu.lts="24.04" \
      dev.embedded.native-gcc.major="14" \
      dev.embedded.uv.version="0.11.29" \
      dev.embedded.capabilities="native-build,native-test,analysis"

FROM base AS mcu-dev

RUN apt-get -o Acquire::Retries=5 -o Acquire::https::Timeout=30 update \
    && apt-get -o Acquire::Retries=5 -o Acquire::https::Timeout=30 \
       install --no-install-recommends -y \
        dfu-util \
        gcc-arm-none-eabi \
        gdb-multiarch \
        libstdc++-arm-none-eabi-dev \
        openocd \
    && rm -rf /var/lib/apt/lists/* \
    && /usr/local/lib/embedded-development/smoke/verify-embedded-tools.sh \
    && dpkg-query --show --showformat='${Package}\t${Version}\n' \
       | sort > /opt/embedded-development/embedded-packages.txt

LABEL dev.embedded.arm-gnu.version="13.2.rel1" \
      dev.embedded.openocd.version="0.12.0" \
      dev.embedded.dfu-util.version="0.11" \
      dev.embedded.capabilities="native-build,native-test,analysis,arm-none-eabi,openocd,dfu-util"
