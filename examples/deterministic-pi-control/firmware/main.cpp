#include <cstdint>

#include "pi_control_example/pi_controller.hpp"

extern "C" {

volatile float g_pi_control_observed_command = 0.0F;
volatile std::uint32_t g_pi_control_observed_fault = 1U;

__attribute__((noinline)) void pi_control_example_observation_ready() { asm volatile("nop"); }

}  // extern "C"

int main() {
  pi_control_example::PiController controller;
  const auto output = controller.step({.target = 1.0F, .measured = 0.25F, .valid = true},
                                      {.microseconds = 10'000U});
  g_pi_control_observed_command = output.command;
  g_pi_control_observed_fault = output.fault ? 1U : 0U;
  pi_control_example_observation_ready();

  while (true) {
  }
}
