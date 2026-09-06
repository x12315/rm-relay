#include "pi_control_example/pi_controller.hpp"

#include <catch2/catch_approx.hpp>
#include <catch2/catch_test_macros.hpp>

namespace {

constexpr float kTolerance = 0.00001F;

}  // namespace

TEST_CASE("nominal input produces the expected command") {
  pi_control_example::PiController controller;
  const auto output = controller.step({.target = 1.0F, .measured = 0.25F, .valid = true},
                                      {.microseconds = 10'000U});

  REQUIRE_FALSE(output.fault);
  REQUIRE(output.command == Catch::Approx(0.4515F).margin(kTolerance));
}

TEST_CASE("invalid input clears the integral state") {
  pi_control_example::PiController controller;
  static_cast<void>(controller.step({.target = 1.0F, .measured = 0.25F, .valid = true},
                                    {.microseconds = 10'000U}));

  const auto invalid = controller.step({.target = 1.0F, .measured = 0.25F, .valid = false},
                                       {.microseconds = 10'000U});
  REQUIRE(invalid.fault);
  REQUIRE(invalid.command == Catch::Approx(0.0F).margin(kTolerance));

  const auto recovered = controller.step({.target = 1.0F, .measured = 0.25F, .valid = true},
                                         {.microseconds = 10'000U});
  REQUIRE_FALSE(recovered.fault);
  REQUIRE(recovered.command == Catch::Approx(0.4515F).margin(kTolerance));
}

TEST_CASE("invalid duration produces a deterministic fault") {
  pi_control_example::PiController controller;

  const auto timeout =
      controller.step({.target = 1.0F, .measured = 0.0F, .valid = true}, {.microseconds = 20'001U});
  REQUIRE(timeout.fault);
  REQUIRE(timeout.command == Catch::Approx(0.0F).margin(kTolerance));

  const auto zero_dt =
      controller.step({.target = 1.0F, .measured = 0.0F, .valid = true}, {.microseconds = 0U});
  REQUIRE(zero_dt.fault);
  REQUIRE(zero_dt.command == Catch::Approx(0.0F).margin(kTolerance));
}

TEST_CASE("command saturates at the positive limit") {
  pi_control_example::PiController controller;

  const auto saturated = controller.step({.target = 10.0F, .measured = 0.0F, .valid = true},
                                         {.microseconds = 10'000U});
  REQUIRE_FALSE(saturated.fault);
  REQUIRE(saturated.command == Catch::Approx(1.0F).margin(kTolerance));
}
