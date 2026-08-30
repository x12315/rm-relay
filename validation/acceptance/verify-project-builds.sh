#!/bin/sh
set -eu

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repository_root="$(CDPATH= cd -- "${script_directory}/../.." && pwd)"
development_image="${DEVELOPMENT_IMAGE:-mcu-dev/toolchain:local}"

temporary_root="$(mktemp -d)"
case "${temporary_root}" in
    /tmp/* | /private/tmp/* | /var/folders/* | /private/var/folders/*) ;;
    *)
        printf 'unexpected temporary directory: %s\n' "${temporary_root}" >&2
        exit 1
        ;;
esac
trap 'rm -rf -- "${temporary_root}"' EXIT HUP INT TERM

for project_path in project-templates/cross-platform-cpp examples/deterministic-pi-control; do
    printf 'verifying project: %s\n' "${project_path}"
    validation_project="${temporary_root}/$(basename "${project_path}")"
    mkdir -p "${validation_project}"
    tar \
        --exclude='./build' \
        --exclude='./install' \
        -C "${repository_root}/${project_path}" \
        -cf - . \
        | tar -C "${validation_project}" -xf -
    docker run --rm \
        -v "${validation_project}:/workspace/project" \
        -w /workspace/project \
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
