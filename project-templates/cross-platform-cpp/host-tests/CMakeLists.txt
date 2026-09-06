find_package(Catch2 3.4 REQUIRED)
include(Catch)

add_executable(cycle_processor_test cycle_processor_test.cpp)
target_link_libraries(cycle_processor_test PRIVATE
  cross_platform_starter::portable_code
  Catch2::Catch2WithMain)
catch_discover_tests(cycle_processor_test TEST_PREFIX "portable_code.")
