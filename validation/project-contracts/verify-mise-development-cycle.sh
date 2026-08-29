#!/bin/sh
set -eu

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repository_root="$(CDPATH= cd -- "${script_directory}/../.." && pwd)"
go_image='golang:1.26.7-bookworm@sha256:e8c859f5632dcfde7b32d2012b4351728f6437930887c2f6a91ea242459e5514'
development_image="${DEVELOPMENT_IMAGE:-mcu-dev/toolchain:local}"
profile_id=embedded-stm32f407-robomaster-c

case "$(uname -s)" in
    Darwin) host_goos=darwin ;;
    Linux) host_goos=linux ;;
    *)
        printf 'unsupported validation host OS: %s\n' "$(uname -s)" >&2
        exit 1
        ;;
esac

case "$(uname -m)" in
    arm64 | aarch64) host_goarch=arm64 ;;
    x86_64 | amd64) host_goarch=amd64 ;;
    *)
        printf 'unsupported validation host architecture: %s\n' "$(uname -m)" >&2
        exit 1
        ;;
esac

temporary_root="$(mktemp -d)"
case "${temporary_root}" in
    /tmp/* | /private/tmp/* | /var/folders/* | /private/var/folders/*) ;;
    *)
        printf 'unexpected temporary directory: %s\n' "${temporary_root}" >&2
        exit 1
        ;;
esac
trap 'rm -rf -- "${temporary_root}"' EXIT HUP INT TERM

distribution_root="${temporary_root}/distribution"
project_root="${temporary_root}/project"
mkdir -p \
    "${distribution_root}/bin" \
    "${distribution_root}/share/rm-relay" \
    "${project_root}"

cp -R \
    "${repository_root}/toolkit/mise" \
    "${repository_root}/toolkit/profiles" \
    "${repository_root}/toolkit/openocd" \
    "${distribution_root}/share/rm-relay/"

tar \
    --exclude='./build' \
    --exclude='./install' \
    -C "${repository_root}/toolkit/project-templates/cross-platform-cpp" \
    -cf - . \
    | tar -C "${project_root}" -xf -

docker run --rm \
    -e CGO_ENABLED=0 \
    -e "GOOS=${host_goos}" \
    -e "GOARCH=${host_goarch}" \
    -v "${repository_root}:/workspace:ro" \
    -v "${distribution_root}/bin:/output" \
    -w /workspace/toolkit \
    "${go_image}" \
    go build \
        -trimpath \
        -ldflags '-s -w -X main.version=validation' \
        -o /output/rm-relay \
        ./cmd/rm-relay
chmod +x "${distribution_root}/bin/rm-relay"

run_relay() {
    RM_RELAY_HOME="${distribution_root}" \
        "${distribution_root}/bin/rm-relay" \
        --project "${project_root}" \
        "$@"
}

run_relay init
run_relay build

manifest_path="${project_root}/install/${profile_id}/rm-relay-output.json"
test -f "${manifest_path}"
test "$(jq -r '.schema_version' "${manifest_path}")" = 1
test "$(jq -r '.profile_id' "${manifest_path}")" = "${profile_id}"
test "$(jq -r '.producer_version' "${manifest_path}")" = validation
test "$(jq -r '.artifacts | length' "${manifest_path}")" = 3
test "$(jq -r '[.artifacts[].role] | sort | join(",")' "${manifest_path}")" \
    = 'firmware.bin,firmware.elf,linker.map'

jq -r '.artifacts[] | [.path, .size, .sha256] | @tsv' "${manifest_path}" \
    | while IFS="$(printf '\t')" read -r relative_path expected_size expected_hash; do
        artifact_path="${project_root}/install/${profile_id}/${relative_path}"
        test -f "${artifact_path}"
        actual_size="$(wc -c < "${artifact_path}" | tr -d ' ')"
        actual_hash="$(shasum -a 256 "${artifact_path}" | awk '{print $1}')"
        test "${actual_size}" = "${expected_size}"
        test "${actual_hash}" = "${expected_hash}"
    done

file "${project_root}/install/${profile_id}/robomaster-c-starter.elf" \
    | grep -F 'ELF 32-bit LSB executable, ARM'

flash_result="$(run_relay --format json flash --target openocd-stlink --dry-run)"
test "$(printf '%s' "${flash_result}" | jq -r '.ok')" = true
test "$(printf '%s' "${flash_result}" | jq -r '.operation')" = flash
test "$(printf '%s' "${flash_result}" | jq -r '.executed')" = false
test "$(printf '%s' "${flash_result}" | jq -r '.command | index("openocd") != null')" = true

printf '%s\n' 'mise-driven development cycle passed'
