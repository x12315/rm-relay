#!/bin/sh
set -eu

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repository_root="$(CDPATH= cd -- "${script_directory}/../.." && pwd)"
development_image="${DEVELOPMENT_IMAGE:-mcu-dev/toolchain:local}"

for project_path in toolkit/project-templates/cross-platform-cpp examples/deterministic-pi-control; do
    printf 'verifying project: %s\n' "${project_path}"
    docker run --rm \
        -v "${repository_root}:/workspace" \
        -w "/workspace/${project_path}" \
        "${development_image}" \
        sh -lc '
            for preset in native-clang native-gcc native-asan; do
                cmake --preset "${preset}" &&
                cmake --build --preset "${preset}" --clean-first &&
                ctest --preset "${preset}"
            done &&
            cmake --preset stm32f407-robomaster-c &&
            cmake --build --preset stm32f407-robomaster-c --clean-first
        '
done
