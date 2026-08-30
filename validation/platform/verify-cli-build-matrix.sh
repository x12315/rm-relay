#!/bin/sh
set -eu

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repository_root="$(CDPATH= cd -- "${script_directory}/../.." && pwd)"
temporary_root="$(mktemp -d)"

case "${temporary_root}" in
    /tmp/* | /private/tmp/* | /var/folders/* | /private/var/folders/*) ;;
    *)
        printf 'unexpected temporary directory: %s\n' "${temporary_root}" >&2
        exit 1
        ;;
esac
trap 'rm -rf -- "${temporary_root}"' EXIT HUP INT TERM

for target in \
    linux/amd64 \
    linux/arm64 \
    darwin/amd64 \
    darwin/arm64 \
    windows/amd64 \
    windows/arm64; do
    target_os="${target%/*}"
    target_arch="${target#*/}"
    output_name="rm-relay-${target_os}-${target_arch}"
    if [ "${target_os}" = windows ]; then
        output_name="${output_name}.exe"
    fi
    printf 'building CLI target: %s\n' "${target}"
    (
        cd "${repository_root}"
        CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
            go build -trimpath -o "${temporary_root}/${output_name}" ./cmd/rm-relay
    )
    test -s "${temporary_root}/${output_name}"
done

printf '%s\n' 'CLI platform build matrix passed'
