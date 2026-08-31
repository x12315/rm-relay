#include <array>
#include <concepts>
#include <optional>
#include <span>

template <std::integral Value>
constexpr Value sum(std::span<const Value> values) noexcept {
  Value result{};
  for (const Value value : values) {
    result += value;
  }
  return result;
}

int main() {
  constexpr std::array values{1, 2, 3};
  static_assert(sum<int>(values) == 6);
  const std::optional<int> result{sum<int>(values)};
  return result.value_or(0) == 6 ? 0 : 1;
}
