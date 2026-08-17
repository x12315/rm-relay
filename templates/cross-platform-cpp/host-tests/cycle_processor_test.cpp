#include "portable_code/cycle_processor.hpp"

#include <cstdio>

namespace {

int fail(const char* message) {
  std::fprintf(stderr, "FAIL: %s\n", message);
  return 1;
}

}  // namespace

int main() {
  const portable_code::CycleProcessor processor;

  const auto valid_cycle = processor.step({.cycle_id = 42U, .valid = true});
  if (valid_cycle.cycle_id != 42U || valid_cycle.fault) {
    return fail("valid cycle");
  }

  const auto invalid_cycle = processor.step({.cycle_id = 43U, .valid = false});
  if (invalid_cycle.cycle_id != 43U || !invalid_cycle.fault) {
    return fail("invalid cycle");
  }

  return 0;
}
