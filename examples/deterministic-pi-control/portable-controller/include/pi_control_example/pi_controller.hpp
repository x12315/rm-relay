#pragma once

#include <cstdint>

namespace pi_control_example {

struct Duration {
  std::uint32_t microseconds;
};

struct ControlInput {
  float target;
  float measured;
  bool valid;
};

struct ControlOutput {
  float command;
  bool fault;
};

class PiController {
 public:
  [[nodiscard]] ControlOutput step(const ControlInput& input, Duration dt) noexcept;
  void reset() noexcept;

 private:
  float integral_{0.0F};
};

}  // namespace pi_control_example
