#pragma once

#include <cstdint>

namespace portable_code {

struct CycleInput {
  std::uint32_t cycle_id;
  bool valid;
};

struct CycleOutput {
  std::uint32_t cycle_id;
  bool fault;
};

class CycleProcessor {
 public:
  [[nodiscard]] constexpr CycleOutput step(const CycleInput input) const noexcept {
    return {.cycle_id = input.cycle_id, .fault = !input.valid};
  }
};

}  // namespace portable_code
