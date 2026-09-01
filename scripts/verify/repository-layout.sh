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
    mise.toml \
    scripts/release/cli.sh \
    scripts/release/goreleaser.yaml \
    scripts/release/README.md \
    scripts/release/tasks.toml \
    environments/embedded-development/tasks.toml \
    services/buildkit/compose.yaml \
    services/buildkit/buildkitd.toml \
    services/buildkit/tasks.toml \
    tests/tasks.toml \
    tests/support/candidate/tasks.toml \
    cmd/rm-relay/main.go \
    cmd/rm-relay/main_test.go \
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
    internal/build/backend/remotebuildkit/backend.go \
    internal/builder/model.go \
    internal/builder/store.go \
    internal/builder/service.go \
    internal/build/cmake/workflow.go \
    internal/build/cmake/build.mise.toml \
    internal/execution/command/runner.go \
    internal/execution/docker/client.go \
    internal/execution/buildx/client.go \
    internal/execution/mise/invocation.go \
    internal/execution/mise/base.mise.toml \
    internal/execution/resourcecache/store.go \
    tests/support/candidate/cmd/main.go \
    tests/support/candidate/internal/candidate/service.go \
    internal/target/adapter.go \
    internal/target/openocd/adapter.go \
    internal/target/openocd/board/robomaster-c.cfg \
    tests/architecture/dependency_direction_test.go \
    tests/integration/development_cycle_test.go \
    tests/integration/fixture_test.go \
    tests/distribution/archives_test.go \
    tests/e2e/local_mcu_cycle_test.go \
    tests/manual/README.md \
    tests/manual/user-experience/local-mcu-development.md \
    scripts/verify/repository-layout.sh \
    scripts/verify/toolchain-source-policy.sh \
    environments/embedded-development/README.md \
    environments/embedded-development/Dockerfile \
    environments/embedded-development/docker-bake.hcl \
    project-templates/cross-platform-cpp/README.md \
    project-templates/cross-platform-cpp/.gitignore \
    project-templates/cross-platform-cpp/.dockerignore \
    project-templates/cross-platform-cpp/CMakeLists.txt \
    project-templates/cross-platform-cpp/CMakePresets.json \
    project-templates/cross-platform-cpp/rm-relay.toml \
    examples/deterministic-pi-control/README.md \
    docs/operator-guide/candidate-experience-environment.md \
    docs/operator-guide/cli-distribution.md; do
    assert_file_exists "${required_file}"
done

for source_path in \
    internal/build/plan.go \
    internal/build/backend/localcontainer/backend.go \
    internal/build/cmake/build.mise.toml \
    tests/architecture/dependency_direction_test.go \
    tests/integration/development_cycle_test.go \
    tests/e2e/local_mcu_cycle_test.go; do
    assert_source_not_ignored "${source_path}"
done

assert_source_not_ignored "dist"

for absent_path in \
    toolkit \
    assets \
    config \
    profiles \
    openocd \
    project-templates/cross-platform-cpp/mise.toml \
    tests/manual/local-mcu-development-cycle-darwin-arm64.md \
    validation \
    cmd/rm-relay-maintainer \
    internal/maintainer \
    distribution \
    container-images \
    .goreleaser.yaml; do
    assert_path_absent "${absent_path}"
done

assert_path_absent "dist"

printf '%s\n' 'repository layout contract passed'
