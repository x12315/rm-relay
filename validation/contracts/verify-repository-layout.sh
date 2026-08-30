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

assert_source_not_ignored() {
    relative_path="$1"
    if git -C "${repository_root}" check-ignore --no-index --quiet -- "${relative_path}"; then
        printf 'source path is hidden by .gitignore: %s\n' "${relative_path}" >&2
        return 1
    fi
}

for required_file in \
    CONTRIBUTING.md \
    LICENSE \
    README.md \
    ROADMAP.md \
    go.mod \
    go.sum \
    cmd/rm-relay/main.go \
    cmd/rm-relay-node/.gitkeep \
    internal/cli/application.go \
    internal/project/config.go \
    internal/profile/catalog.go \
    internal/profile/builtin/embedded-stm32f407-robomaster-c/profile.toml \
    internal/build/plan.go \
    internal/build/service.go \
    internal/build/workflow.go \
    internal/build/output/manifest.go \
    internal/build/backend/localcontainer/backend.go \
    internal/build/cmake/workflow.go \
    internal/build/cmake/build.mise.toml \
    internal/execution/command/runner.go \
    internal/execution/mise/invocation.go \
    internal/execution/mise/base.mise.toml \
    internal/execution/resourcecache/store.go \
    internal/target/adapter.go \
    internal/target/openocd/adapter.go \
    internal/target/openocd/board/robomaster-c.cfg \
    container-images/embedded-development/README.md \
    container-images/embedded-development/Dockerfile \
    container-images/embedded-development/docker-bake.hcl \
    project-templates/cross-platform-cpp/README.md \
    project-templates/cross-platform-cpp/.gitignore \
    project-templates/cross-platform-cpp/CMakeLists.txt \
    project-templates/cross-platform-cpp/CMakePresets.json \
    project-templates/cross-platform-cpp/rm-relay.toml \
    examples/deterministic-pi-control/README.md \
    validation/README.md \
    validation/contracts/verify-repository-layout.sh \
    validation/contracts/verify-toolchain-source-policy.sh \
    validation/module-boundaries/dependency_direction_test.go \
    validation/acceptance/verify-project-builds.sh \
    validation/acceptance/verify-local-mcu-cycle.sh \
    validation/platform/verify-cli-build-matrix.sh; do
    assert_file_exists "${required_file}"
done

for source_path in \
    internal/build/plan.go \
    internal/build/backend/localcontainer/backend.go \
    internal/build/cmake/build.mise.toml; do
    assert_source_not_ignored "${source_path}"
done

for absent_path in \
    toolkit \
    assets \
    config \
    profiles \
    openocd \
    project-templates/cross-platform-cpp/mise.toml; do
    assert_path_absent "${absent_path}"
done

printf '%s\n' 'repository layout contract passed'
