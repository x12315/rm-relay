#include <cstdint>

#include "portable_code/cycle_processor.hpp"

extern "C" {

volatile std::uint32_t g_starter_observed_cycle_id = 0U;
volatile std::uint32_t g_starter_observed_fault = 1U;

__attribute__((noinline)) void starter_firmware_observation_ready() { asm volatile("nop"); }

}  // extern "C"

int main() {
  const portable_code::CycleProcessor processor;
  const auto output = processor.step({.cycle_id = 42U, .valid = true});
  g_starter_observed_cycle_id = output.cycle_id;
  g_starter_observed_fault = output.fault ? 1U : 0U;
  starter_firmware_observation_ready();

  while (true) {
  }
}
