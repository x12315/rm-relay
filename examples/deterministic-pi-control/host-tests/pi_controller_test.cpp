#include "pi_control_example/pi_controller.hpp"

#include <cmath>
#include <cstdio>

namespace {

bool near(float lhs, float rhs, float tolerance = 0.00001F) {
  return std::fabs(lhs - rhs) <= tolerance;
}

int fail(const char* message) {
  std::fprintf(stderr, "FAIL: %s\n", message);
  return 1;
}

}  // namespace

int main() {
  pi_control_example::PiController controller;

  const auto nominal = controller.step({.target = 1.0F, .measured = 0.25F, .valid = true},
                                       {.microseconds = 10'000U});
  if (nominal.fault || !near(nominal.command, 0.4515F)) {
    return fail("nominal vector");
  }

  const auto invalid = controller.step({.target = 1.0F, .measured = 0.25F, .valid = false},
                                       {.microseconds = 10'000U});
  if (!invalid.fault || !near(invalid.command, 0.0F)) {
    return fail("invalid input");
  }

  const auto recovered = controller.step({.target = 1.0F, .measured = 0.25F, .valid = true},
                                         {.microseconds = 10'000U});
  if (recovered.fault || !near(recovered.command, 0.4515F)) {
    return fail("fault clears integral state");
  }

  const auto timeout =
      controller.step({.target = 1.0F, .measured = 0.0F, .valid = true}, {.microseconds = 20'001U});
  if (!timeout.fault || !near(timeout.command, 0.0F)) {
    return fail("timeout input");
  }

  const auto zero_dt =
      controller.step({.target = 1.0F, .measured = 0.0F, .valid = true}, {.microseconds = 0U});
  if (!zero_dt.fault || !near(zero_dt.command, 0.0F)) {
    return fail("zero dt");
  }

  const auto saturated = controller.step({.target = 10.0F, .measured = 0.0F, .valid = true},
                                         {.microseconds = 10'000U});
  if (saturated.fault || !near(saturated.command, 1.0F)) {
    return fail("positive saturation");
  }

  controller.reset();
  return 0;
}
