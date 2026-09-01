#!/bin/sh
set -eu

usage() {
    printf '%s\n' 'usage: cli.sh <build|snapshot> <absolute-output-directory>' >&2
    exit 2
}

[ "$#" -eq 2 ] || usage
release_mode="$1"
requested_output="$2"

case "${release_mode}" in
    build|snapshot) ;;
    *) usage ;;
esac

case "${requested_output}" in
    /*) ;;
    *)
        printf '%s\n' 'CLI output directory must be absolute' >&2
        exit 2
        ;;
esac

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)"
repository_root="$(git -C "${script_directory}" rev-parse --show-toplevel)"
repository_root="$(CDPATH= cd -- "${repository_root}" && pwd -P)"

if [ -n "$(git -C "${repository_root}" status --porcelain)" ]; then
    printf '%s\n' 'repository contains uncommitted changes; commit or remove them before packaging the CLI' >&2
    exit 1
fi

output_parent="$(dirname -- "${requested_output}")"
output_name="$(basename -- "${requested_output}")"
mkdir -p -- "${output_parent}"
output_parent="$(CDPATH= cd -- "${output_parent}" && pwd -P)"
output_directory="${output_parent}/${output_name}"

case "${output_directory}/" in
    "${repository_root}/"*)
        printf 'CLI output directory must be outside repository %s\n' "${repository_root}" >&2
        exit 2
        ;;
esac

if [ -e "${output_directory}" ] || [ -L "${output_directory}" ]; then
    printf 'CLI output directory already exists: %s\n' "${output_directory}" >&2
    exit 2
fi

temporary_root="$(mktemp -d "${TMPDIR:-/tmp}/rm-relay-release.XXXXXX")"
staging_directory="$(mktemp -d "${output_parent}/.${output_name}.partial.XXXXXX")"
cleanup() {
    rm -rf -- "${temporary_root}"
    if [ -n "${staging_directory}" ]; then
        rm -rf -- "${staging_directory}"
    fi
}
trap cleanup EXIT HUP INT TERM

checkout="${temporary_root}/repository"
git clone --quiet --no-hardlinks "${repository_root}" "${checkout}"

(
    cd "${checkout}"
    if [ "${release_mode}" = build ]; then
        goreleaser build --snapshot --clean --config scripts/release/goreleaser.yaml
    else
        goreleaser release --snapshot --clean --config scripts/release/goreleaser.yaml
    fi
)

cp -R "${checkout}/dist/." "${staging_directory}/"
mv -- "${staging_directory}" "${output_directory}"
staging_directory=""
