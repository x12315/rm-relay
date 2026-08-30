find_package(Catch2 3.4 REQUIRED)
include(Catch)

add_executable(pi_controller_test pi_controller_test.cpp)
target_link_libraries(pi_controller_test PRIVATE
  pi_control_example::portable_controller
  Catch2::Catch2WithMain)
catch_discover_tests(pi_controller_test TEST_PREFIX "pi_control_example.")
