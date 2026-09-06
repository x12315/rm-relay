cmake_minimum_required(VERSION 3.28)
project(cross_platform_cpp_starter LANGUAGES C CXX ASM)

set(CMAKE_C_STANDARD 17)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

option(ENABLE_SANITIZERS "Enable AddressSanitizer and UndefinedBehaviorSanitizer" OFF)

add_library(project_options INTERFACE)
target_compile_options(project_options INTERFACE
  $<$<COMPILE_LANGUAGE:C,CXX>:-Wall;-Wextra;-Wpedantic;-Werror>)

if(CMAKE_CROSSCOMPILING)
  if(NOT DEFINED BOARD_PROFILE OR NOT EXISTS "${BOARD_PROFILE}")
    message(FATAL_ERROR "BOARD_PROFILE must name an existing board profile")
  endif()
  include("${BOARD_PROFILE}")
  target_compile_options(project_options INTERFACE
    ${BOARD_CPU_OPTIONS}
    -ffreestanding
    -ffunction-sections
    -fdata-sections
    $<$<COMPILE_LANGUAGE:CXX>:-fno-exceptions;-fno-rtti;-fno-threadsafe-statics;-fno-use-cxa-atexit>)
  target_link_options(project_options INTERFACE ${BOARD_CPU_OPTIONS})
elseif(ENABLE_SANITIZERS)
  target_compile_options(project_options INTERFACE
    -fsanitize=address,undefined
    -fno-omit-frame-pointer)
  target_link_options(project_options INTERFACE -fsanitize=address,undefined)
endif()

add_subdirectory(portable-code)

if(CMAKE_CROSSCOMPILING)
  add_subdirectory(firmware)
else()
  enable_testing()
  add_subdirectory(host-tests)
endif()
