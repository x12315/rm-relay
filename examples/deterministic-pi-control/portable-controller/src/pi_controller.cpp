#include "pi_control_example/pi_controller.hpp"

namespace pi_control_example {
namespace {

constexpr float clamp(float value, float minimum, float maximum) noexcept {
  return value < minimum ? minimum : (value > maximum ? maximum : value);
}

}  // namespace

ControlOutput PiController::step(const ControlInput& input, Duration dt) noexcept {
  if (!input.valid || dt.microseconds == 0U || dt.microseconds > 20'000U) {
    reset();
    return {.command = 0.0F, .fault = true};
  }

  constexpr float proportional_gain = 0.6F;
  constexpr float integral_gain = 0.2F;
  constexpr float microseconds_to_seconds = 0.000001F;

  const float error = input.target - input.measured;
  const float seconds = static_cast<float>(dt.microseconds) * microseconds_to_seconds;
  integral_ = clamp(integral_ + error * seconds, -0.5F, 0.5F);
  const float command = clamp(proportional_gain * error + integral_gain * integral_, -1.0F, 1.0F);
  return {.command = command, .fault = false};
}

void PiController::reset() noexcept { integral_ = 0.0F; }

}  // namespace pi_control_example
