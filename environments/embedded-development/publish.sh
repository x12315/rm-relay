#!/bin/sh
set -eu

if [ "$#" -ne 3 ]; then
    printf '%s\n' 'usage: publish.sh <buildx-builder> <oci-image:version> <absolute-handoff-path>' >&2
    exit 2
fi

image_builder="$1"
image_tag="$2"
handoff_path="$3"

script_directory="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repository_root="$(CDPATH= cd -- "${script_directory}/../.." && pwd)"

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
case "${handoff_path}" in
    "${repository_root}"|"${repository_root}"/*)
        printf '%s\n' 'environment publication handoff must be outside the repository' >&2
        exit 2
        ;;
esac
if [ -e "${handoff_path}" ] || [ -L "${handoff_path}" ]; then
    printf 'environment publication handoff already exists: %s\n' "${handoff_path}" >&2
    exit 2
fi
handoff_parent="$(dirname -- "${handoff_path}")"
if [ ! -d "${handoff_parent}" ]; then
    printf 'environment publication handoff parent does not exist: %s\n' "${handoff_parent}" >&2
    exit 2
fi

if [ -n "$(git -C "${repository_root}" status --porcelain)" ]; then
    printf '%s\n' 'environment publication requires a clean repository revision' >&2
    exit 2
fi
source_revision="$(git -C "${repository_root}" rev-parse HEAD)"
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

bake_file="environments/embedded-development/docker-bake.hcl"
build_metadata="${temporary_root}/build-metadata.json"

cd "${repository_root}"
docker buildx bake \
    --builder "${image_builder}" \
    --file "${bake_file}" \
    --check publish

docker buildx bake \
    --builder "${image_builder}" \
    --file "${bake_file}" \
    publish \
    --set "mcu-dev-multiarch.tags=${image_tag}" \
    --push \
    --provenance mode=max \
    --sbom true \
    --metadata-file "${build_metadata}" \
    --progress plain

manifest="$(docker buildx imagetools inspect "${image_tag}" --format '{{json .Manifest}}')"
digest="$(printf '%s\n' "${manifest}" \
    | sed -n 's/^[[:space:]]*"digest":[[:space:]]*"\(sha256:[0-9a-f]\{64\}\)"[,]\{0,1\}[[:space:]]*$/\1/p' \
    | sed -n '1p')"
if [ -z "${digest}" ]; then
    printf '%s\n' 'published OCI manifest does not contain a valid top-level SHA-256 digest' >&2
    exit 1
fi
for architecture in amd64 arm64; do
    if ! printf '%s\n' "${manifest}" | grep -q '"architecture"[[:space:]]*:[[:space:]]*"'"${architecture}"'"'; then
        printf 'published OCI manifest is missing linux/%s\n' "${architecture}" >&2
        exit 1
    fi
done

immutable_reference="${image_name}@${digest}"
{
    printf '%s\n' 'schema_version = 1'
    printf '%s\n' 'environment_id = "embedded-development"'
    printf 'tag = "%s"\n' "${image_tag}"
    printf 'digest = "%s"\n' "${digest}"
    printf 'immutable_reference = "%s"\n' "${immutable_reference}"
    printf 'source_revision = "%s"\n' "${source_revision}"
    printf '%s\n' 'platforms = ["linux/amd64", "linux/arm64"]'
} > "${handoff_temporary}"

mv -- "${handoff_temporary}" "${handoff_path}"
printf '%s\n' "${immutable_reference}"
