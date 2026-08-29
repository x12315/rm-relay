#!/bin/sh
set -eu

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repository_root="$(CDPATH= cd -- "${script_directory}/../.." && pwd)"
dockerfile="${repository_root}/toolkit/container-images/embedded-development/Dockerfile"
versions_file="${repository_root}/toolkit/container-images/embedded-development/locks/versions.env"
base_smoke="${repository_root}/toolkit/container-images/embedded-development/smoke/verify-base-tools.sh"
embedded_smoke="${repository_root}/toolkit/container-images/embedded-development/smoke/verify-embedded-tools.sh"
template_presets="${repository_root}/toolkit/project-templates/cross-platform-cpp/CMakePresets.json"
example_presets="${repository_root}/examples/deterministic-pi-control/CMakePresets.json"

assert_contains() {
    file_path="$1"
    expected_text="$2"
    if ! grep -Fq -- "${expected_text}" "${file_path}"; then
        printf 'required text is missing: file=%s text=%s\n' \
            "${file_path}" "${expected_text}" >&2
        return 1
    fi
}

assert_absent() {
    file_path="$1"
    forbidden_text="$2"
    if grep -Fq -- "${forbidden_text}" "${file_path}"; then
        printf 'forbidden text is present: file=%s text=%s\n' \
            "${file_path}" "${forbidden_text}" >&2
        return 1
    fi
}

assert_contains "${dockerfile}" 'FROM ubuntu:24.04 AS base'
assert_contains "${dockerfile}" 'ARG UBUNTU_MIRROR=https://mirrors.ustc.edu.cn/ubuntu'
assert_contains "${dockerfile}" 'ARG UBUNTU_PORTS_MIRROR=https://mirrors.ustc.edu.cn/ubuntu-ports'
assert_contains "${dockerfile}" 'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg'
assert_absent "${dockerfile}" 'snapshot.ubuntu.com'
assert_absent "${dockerfile}" '_DEB_VERSION'
assert_absent "${dockerfile}" 'Trusted: yes'
assert_absent "${dockerfile}" 'AllowUnauthenticated'
assert_absent "${dockerfile}" 'AllowInsecureRepositories'

assert_contains "${versions_file}" 'UBUNTU_LTS=24.04'
assert_contains "${versions_file}" 'UBUNTU_CODENAME=noble'
assert_contains "${versions_file}" 'NATIVE_GCC_MAJOR=14'
assert_contains "${versions_file}" 'ARM_GNU_RELEASE=13.2.Rel1'
assert_contains "${versions_file}" 'MISE_VERSION=2026.8.14'
assert_contains "${versions_file}" 'MISE_ARCHIVE_SHA256_AMD64=64d5f34aeb7a4e0e327dc1c9be66cd8162e14899a47b11901154a100285a3d61'
assert_contains "${versions_file}" 'MISE_ARCHIVE_SHA256_ARM64=940639580227bd838e3b3ea5b2084ea397399b0db162c2e4dd90b5730850e48e'
assert_absent "${versions_file}" 'UBUNTU_SNAPSHOT='
assert_absent "${versions_file}" '_DEB_VERSION='

assert_contains "${base_smoke}" 'g++-14 -std=c++20'
assert_contains "${base_smoke}" 'gcc-14 -dumpfullversion'
assert_contains "${base_smoke}" 'mise --version | grep -F "${MISE_VERSION}"'
assert_absent "${base_smoke}" 'assert_dpkg_version'
assert_absent "${embedded_smoke}" 'assert_dpkg_version'

assert_contains "${template_presets}" '"CMAKE_C_COMPILER": "gcc-14"'
assert_contains "${template_presets}" '"CMAKE_CXX_COMPILER": "g++-14"'
assert_contains "${example_presets}" '"CMAKE_C_COMPILER": "gcc-14"'
assert_contains "${example_presets}" '"CMAKE_CXX_COMPILER": "g++-14"'

printf '%s\n' 'toolchain and source policy contract passed'
