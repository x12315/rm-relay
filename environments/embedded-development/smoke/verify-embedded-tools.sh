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

for command_name in arm-none-eabi-gcc arm-none-eabi-g++ gdb-multiarch \
    arm-none-eabi-objcopy arm-none-eabi-readelf arm-none-eabi-size dfu-util openocd; do
    assert_command "${command_name}"
done

arm-none-eabi-gcc -dumpfullversion -dumpversion \
    | grep -Eq "^${ARM_GNU_MAJOR}([.]|$)"
arm-none-eabi-gcc --version | grep '13.2'
gdb-multiarch --version
dfu-util --version
openocd --version

temporary_directory="$(mktemp -d)"
case "${temporary_directory}" in
    /tmp/*) ;;
    *)
        printf 'unexpected temporary directory: %s\n' "${temporary_directory}" >&2
        exit 1
        ;;
esac
trap 'rm -rf -- "${temporary_directory}"' EXIT HUP INT TERM

arm-none-eabi-g++ -std=c++20 -ffreestanding -fno-exceptions -fno-rtti \
    -Wall -Wextra -Werror -c "${contract_source}" \
    -o "${temporary_directory}/arm-cxx20-contract.o"
