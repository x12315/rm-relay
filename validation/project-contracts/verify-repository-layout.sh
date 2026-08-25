#!/bin/sh
set -eu

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repository_root="$(CDPATH= cd -- "${script_directory}/../.." && pwd)"

assert_file_exists() {
    relative_path="$1"
    if [ ! -f "${repository_root}/${relative_path}" ]; then
        printf 'required file is missing: %s\n' "${relative_path}" >&2
        return 1
    fi
}

assert_path_absent() {
    relative_path="$1"
    if [ -e "${repository_root}/${relative_path}" ]; then
        printf 'legacy or misplaced path still exists: %s\n' "${relative_path}" >&2
        return 1
    fi
}

for required_file in \
    CONTRIBUTING.md \
    LICENSE \
    README.md \
    ROADMAP.md \
    container-images/embedded-development/README.md \
    container-images/embedded-development/Dockerfile \
    container-images/embedded-development/docker-bake.hcl \
    container-images/embedded-development/locks/README.md \
    container-images/embedded-development/locks/versions.env \
    templates/cross-platform-cpp/README.md \
    templates/cross-platform-cpp/CMakeLists.txt \
    templates/cross-platform-cpp/CMakePresets.json \
    templates/cross-platform-cpp/portable-code/README.md \
    examples/deterministic-pi-control/README.md \
    examples/deterministic-pi-control/CMakeLists.txt \
    examples/deterministic-pi-control/CMakePresets.json \
    examples/deterministic-pi-control/portable-controller/README.md \
    docs/community/README.md \
    docs/community/why-rm-relay.md \
    docs/architecture/README.md \
    docs/architecture/environments-and-profiles.md \
    docs/architecture/builds-and-outputs.md \
    docs/architecture/targets-and-sessions.md \
    docs/architecture/service-topology.md \
    docs/reference/development-contracts.md \
    docs/user-guide/build-native.md \
    docs/operator-guide/build-and-verify-images.md \
    docs/operator-guide/repository-boundaries.md \
    validation/README.md \
    validation/project-contracts/verify-toolchain-source-policy.sh; do
    assert_file_exists "${required_file}"
done

for absent_path in \
    CMakeLists.txt \
    CMakePresets.json \
    .clang-format \
    .clang-tidy \
    containers \
    cmake \
    embedded-dev-docker \
    examples/shared_core \
    examples/native \
    examples/stm32f407; do
    assert_path_absent "${absent_path}"
done

printf '%s\n' 'repository layout contract passed'
