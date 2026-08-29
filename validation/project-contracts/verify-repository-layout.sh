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
    toolkit/README.md \
    toolkit/go.mod \
    toolkit/cmd/rm-relay/main.go \
    toolkit/cmd/rm-relay-node/.gitkeep \
    toolkit/internal/backend/localcontainer/backend.go \
    toolkit/internal/buildoutput/manifest.go \
    toolkit/internal/cli/application.go \
    toolkit/internal/commandexec/runner.go \
    toolkit/internal/executionplan/plan.go \
    toolkit/internal/miseexec/invocation.go \
    toolkit/internal/project/config.go \
    toolkit/internal/profile/catalog.go \
    toolkit/internal/target/adapter.go \
    toolkit/internal/target/openocd/adapter.go \
    toolkit/internal/workspacebuild/service.go \
    toolkit/container-images/embedded-development/README.md \
    toolkit/container-images/embedded-development/Dockerfile \
    toolkit/container-images/embedded-development/docker-bake.hcl \
    toolkit/container-images/embedded-development/locks/README.md \
    toolkit/container-images/embedded-development/locks/versions.env \
    toolkit/mise/core.toml \
    toolkit/project-templates/cross-platform-cpp/README.md \
    toolkit/project-templates/cross-platform-cpp/CMakeLists.txt \
    toolkit/project-templates/cross-platform-cpp/CMakePresets.json \
    toolkit/project-templates/cross-platform-cpp/rm-relay.toml \
    toolkit/project-templates/cross-platform-cpp/mise.toml \
    toolkit/project-templates/cross-platform-cpp/portable-code/README.md \
    toolkit/profiles/embedded-stm32f407-robomaster-c/profile.toml \
    toolkit/profiles/embedded-stm32f407-robomaster-c/mise.toml \
    toolkit/openocd/boards/robomaster-c.cfg \
    examples/deterministic-pi-control/README.md \
    examples/deterministic-pi-control/CMakeLists.txt \
    examples/deterministic-pi-control/CMakePresets.json \
    examples/deterministic-pi-control/portable-controller/README.md \
    docs/community/README.md \
    docs/community/why-rm-relay.md \
    docs/architecture/README.md \
    docs/architecture/environments-and-profiles.md \
    docs/architecture/builds-and-outputs.md \
    docs/architecture/targets-and-access.md \
    docs/architecture/service-topology.md \
    docs/reference/development-contracts.md \
    docs/user-guide/README.md \
    docs/user-guide/build-native.md \
    docs/operator-guide/build-and-verify-images.md \
    docs/operator-guide/repository-assets.md \
    validation/README.md \
    validation/project-contracts/verify-mise-development-cycle.sh \
    validation/project-contracts/verify-toolchain-source-policy.sh; do
    assert_file_exists "${required_file}"
done

for absent_path in \
    CMakeLists.txt \
    CMakePresets.json \
    .clang-format \
    .clang-tidy \
    components \
    container-images \
    containers \
    cmake \
    embedded-dev-docker \
    examples/shared_core \
    examples/native \
    examples/stm32f407 \
    openocd \
    templates \
    docs/operator-guide/repository-boundaries.md \
    docs/architecture/targets-and-sessions.md; do
    assert_path_absent "${absent_path}"
done

printf '%s\n' 'repository layout contract passed'
