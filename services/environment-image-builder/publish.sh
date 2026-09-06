#!/bin/sh
set -eu

if [ "$#" -ne 6 ]; then
    printf '%s\n' 'usage: publish.sh <source-root> <bake-file> <identity-file> <buildx-builder> <oci-image:version> <absolute-handoff-path>' >&2
    exit 2
fi

source_input="$1"
bake_file="$2"
identity_file="$3"
image_builder="$4"
image_tag="$5"
handoff_path="$6"

if [ ! -d "${source_input}" ]; then
    printf 'environment source is not a directory: %s\n' "${source_input}" >&2
    exit 2
fi
source_root="$(CDPATH= cd -- "${source_input}" && pwd -P)"

resolve_source_file() {
    relative_path="$1"
    label="$2"
    case "${relative_path}" in
        /*|'')
            printf '%s must be a non-empty source-relative path\n' "${label}" >&2
            return 1
            ;;
    esac
    relative_directory="$(dirname -- "${relative_path}")"
    if [ ! -d "${source_root}/${relative_directory}" ]; then
        printf '%s directory does not exist: %s\n' "${label}" "${relative_directory}" >&2
        return 1
    fi
    resolved_directory="$(CDPATH= cd -- "${source_root}/${relative_directory}" && pwd -P)"
    resolved_path="${resolved_directory}/$(basename -- "${relative_path}")"
    case "${resolved_path}" in
        "${source_root}"/*) ;;
        *)
            printf '%s must stay inside the environment source\n' "${label}" >&2
            return 1
            ;;
    esac
    if [ ! -f "${resolved_path}" ] || [ -L "${resolved_path}" ]; then
        printf '%s must be a regular non-symlink file: %s\n' "${label}" "${relative_path}" >&2
        return 1
    fi
    printf '%s\n' "${resolved_path}"
}

bake_path="$(resolve_source_file "${bake_file}" 'Bake file')"
identity_path="$(resolve_source_file "${identity_file}" 'identity file')"

environment_ids="$(sed -n 's/^[[:space:]]*id[[:space:]]*=[[:space:]]*"\([A-Za-z0-9][A-Za-z0-9_.-]*\)"[[:space:]]*$/\1/p' "${identity_path}")"
old_ifs="${IFS}"
IFS='
'
set -- ${environment_ids}
IFS="${old_ifs}"
if [ "$#" -ne 1 ]; then
    printf '%s\n' 'environment identity must contain exactly one valid id' >&2
    exit 2
fi
environment_id="$1"

case "${image_builder}" in
    ''|-*|*[!A-Za-z0-9_.-]*)
        printf 'invalid image-production Buildx builder: %s\n' "${image_builder}" >&2
        exit 2
        ;;
esac

case "${image_tag}" in
    ''|*[!A-Za-z0-9_./:-]*|*@*)
        printf 'OCI publication reference must be a tag without spaces or digest: %s\n' "${image_tag}" >&2
        exit 2
        ;;
esac
image_component="${image_tag##*/}"
case "${image_component}" in
    *:*) ;;
    *)
        printf 'OCI publication reference must include an explicit version tag: %s\n' "${image_tag}" >&2
        exit 2
        ;;
esac
image_name="${image_tag%:*}"
if [ -z "${image_name}" ]; then
    printf 'OCI publication reference has no image name: %s\n' "${image_tag}" >&2
    exit 2
fi

case "${handoff_path}" in
    /*) ;;
    *)
        printf '%s\n' 'environment publication handoff path must be absolute' >&2
        exit 2
        ;;
esac
handoff_parent="$(dirname -- "${handoff_path}")"
if [ ! -d "${handoff_parent}" ]; then
    printf 'environment publication handoff parent does not exist: %s\n' "${handoff_parent}" >&2
    exit 2
fi
handoff_name="$(basename -- "${handoff_path}")"
case "${handoff_name}" in
    ''|.|..)
        printf 'environment publication handoff needs a file name: %s\n' "${handoff_path}" >&2
        exit 2
        ;;
esac
handoff_parent="$(CDPATH= cd -- "${handoff_parent}" && pwd -P)"
handoff_path="${handoff_parent}/${handoff_name}"
case "${handoff_path}" in
    "${source_root}"|"${source_root}"/*)
        printf '%s\n' 'environment publication handoff must be outside the environment source' >&2
        exit 2
        ;;
esac
if [ -e "${handoff_path}" ] || [ -L "${handoff_path}" ]; then
    printf 'environment publication handoff already exists: %s\n' "${handoff_path}" >&2
    exit 2
fi

source_git_root="$(git -C "${source_root}" rev-parse --show-toplevel)"
source_git_root="$(CDPATH= cd -- "${source_git_root}" && pwd -P)"
if [ "${source_git_root}" != "${source_root}" ]; then
    printf 'environment source must be a Git worktree root: %s\n' "${source_root}" >&2
    exit 2
fi
if [ -n "$(git -C "${source_root}" status --porcelain)" ]; then
    printf '%s\n' 'environment publication requires a clean source revision' >&2
    exit 2
fi
source_revision="$(git -C "${source_root}" rev-parse HEAD)"
case "${source_revision}" in
    *[!0-9a-f]*|'')
        printf 'invalid Git source revision: %s\n' "${source_revision}" >&2
        exit 2
        ;;
esac

temporary_root="$(mktemp -d "${TMPDIR:-/tmp}/rm-relay-environment-publish.XXXXXX")"
handoff_temporary="$(mktemp "${handoff_path}.tmp.XXXXXX")"
cleanup() {
    rm -rf -- "${temporary_root}"
    rm -f -- "${handoff_temporary}"
}
trap cleanup EXIT HUP INT TERM

build_metadata="${temporary_root}/build-metadata.json"

cd "${source_root}"
docker buildx bake \
    --builder "${image_builder}" \
    --file "${bake_file}" \
    --check publish

docker buildx bake \
    --builder "${image_builder}" \
    --file "${bake_file}" \
    publish \
    --set "*.tags=${image_tag}" \
    --push \
    --provenance mode=max \
    --sbom true \
    --metadata-file "${build_metadata}" \
    --progress plain

digest="$(sed -n 's/.*"containerimage.digest"[[:space:]]*:[[:space:]]*"\(sha256:[0-9a-f]\{64\}\)".*/\1/p' "${build_metadata}" | sed -n '1p')"
if [ -z "${digest}" ]; then
    printf '%s\n' 'Buildx metadata does not contain a valid top-level SHA-256 digest' >&2
    exit 1
fi

manifest="$(docker buildx imagetools inspect "${image_tag}" --format '{{json .Manifest}}')"
for architecture in amd64 arm64; do
    if ! printf '%s\n' "${manifest}" | grep -q '"architecture"[[:space:]]*:[[:space:]]*"'"${architecture}"'"'; then
        printf 'published OCI manifest is missing linux/%s\n' "${architecture}" >&2
        exit 1
    fi
done

immutable_reference="${image_name}@${digest}"
{
    printf '%s\n' 'schema_version = 1'
    printf 'environment_id = "%s"\n' "${environment_id}"
    printf 'tag = "%s"\n' "${image_tag}"
    printf 'digest = "%s"\n' "${digest}"
    printf 'immutable_reference = "%s"\n' "${immutable_reference}"
    printf 'source_revision = "%s"\n' "${source_revision}"
    printf '%s\n' 'platforms = ["linux/amd64", "linux/arm64"]'
} > "${handoff_temporary}"

mv -- "${handoff_temporary}" "${handoff_path}"
printf '%s\n' "${immutable_reference}"
