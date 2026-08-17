add_executable(robomaster-c-starter
  main.cpp
  startup.cpp
  vector_table.S)

set_target_properties(robomaster-c-starter PROPERTIES SUFFIX ".elf")
target_link_libraries(robomaster-c-starter PRIVATE cross_platform_starter::portable_code gcc)
target_link_options(robomaster-c-starter PRIVATE
  -nostartfiles
  -nostdlib
  "-T${BOARD_LINKER_SCRIPT}"
  -Wl,--gc-sections
  -Wl,--cref
  "-Wl,-Map,${CMAKE_CURRENT_BINARY_DIR}/robomaster-c-starter.map"
  -Wl,--print-memory-usage)

add_custom_command(TARGET robomaster-c-starter POST_BUILD
  COMMAND "${CMAKE_OBJCOPY}" -O binary
          "$<TARGET_FILE:robomaster-c-starter>"
          "${CMAKE_CURRENT_BINARY_DIR}/robomaster-c-starter.bin"
  COMMAND "${CMAKE_SIZE}" "$<TARGET_FILE:robomaster-c-starter>"
  VERBATIM)
