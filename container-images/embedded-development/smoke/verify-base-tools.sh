#!/bin/sh
set -eu

versions_file=/opt/embedded-development/versions.env
contract_source=/usr/local/lib/embedded-development/smoke/cxx20-compiler-contract.cpp

test -r "${versions_file}"
test -r "${contract_source}"
. "${versions_file}"

assert_command() {
    command -v "$1" >/dev/null
}

for command_name in ccache cmake ninja clang clang++ clang-format clang-tidy \
    gcc-14 g++-14 gdb git mise python3 uv uvx; do
    assert_command "${command_name}"
done

test -r /opt/rm-relay/execution/mise/base.mise.toml
test -r /opt/rm-relay/build/cmake/build.mise.toml

. /etc/os-release
test "${VERSION_ID}" = "${UBUNTU_LTS}"

gcc-14 -dumpfullversion -dumpversion | grep -Eq "^${NATIVE_GCC_MAJOR}([.]|$)"
cmake --version
ninja --version
clang --version
gcc-14 --version
uv --version | grep -F "${UV_VERSION}"
mise --version | grep -F "${MISE_VERSION}"

temporary_directory="$(mktemp -d)"
case "${temporary_directory}" in
    /tmp/*) ;;
    *)
        printf 'unexpected temporary directory: %s\n' "${temporary_directory}" >&2
        exit 1
        ;;
esac
trap 'rm -rf -- "${temporary_directory}"' EXIT HUP INT TERM

g++-14 -std=c++20 -Wall -Wextra -Werror "${contract_source}" \
    -o "${temporary_directory}/gcc-cxx20-contract"
"${temporary_directory}/gcc-cxx20-contract"

clang++ -std=c++20 -Wall -Wextra -Werror "${contract_source}" \
    -o "${temporary_directory}/clang-cxx20-contract"
"${temporary_directory}/clang-cxx20-contract"

clang++ -std=c++20 -Wall -Wextra -Werror -fsanitize=address,undefined \
    -fno-omit-frame-pointer "${contract_source}" \
    -o "${temporary_directory}/clang-sanitizer-contract"
"${temporary_directory}/clang-sanitizer-contract"
