add_executable(robomaster-c-pi-control-example
  main.cpp
  startup.cpp
  vector_table.S)

set_target_properties(robomaster-c-pi-control-example PROPERTIES SUFFIX ".elf")
target_link_libraries(robomaster-c-pi-control-example
  PRIVATE pi_control_example::portable_controller gcc)
target_link_options(robomaster-c-pi-control-example PRIVATE
  -nostartfiles
  -nostdlib
  "-T${BOARD_LINKER_SCRIPT}"
  -Wl,--gc-sections
  -Wl,--cref
  "-Wl,-Map,${CMAKE_CURRENT_BINARY_DIR}/robomaster-c-pi-control-example.map"
  -Wl,--print-memory-usage)

add_custom_command(TARGET robomaster-c-pi-control-example POST_BUILD
  COMMAND "${CMAKE_OBJCOPY}" -O binary
          "$<TARGET_FILE:robomaster-c-pi-control-example>"
          "${CMAKE_CURRENT_BINARY_DIR}/robomaster-c-pi-control-example.bin"
  COMMAND "${CMAKE_SIZE}" "$<TARGET_FILE:robomaster-c-pi-control-example>"
  VERBATIM)
