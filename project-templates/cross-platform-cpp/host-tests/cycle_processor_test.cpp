#include "portable_code/cycle_processor.hpp"

#include <catch2/catch_test_macros.hpp>

TEST_CASE("valid cycles preserve identity") {
  const portable_code::CycleProcessor processor;
  const auto output = processor.step({.cycle_id = 42U, .valid = true});

  REQUIRE(output.cycle_id == 42U);
  REQUIRE_FALSE(output.fault);
}

TEST_CASE("invalid cycles produce a deterministic fault") {
  const portable_code::CycleProcessor processor;
  const auto output = processor.step({.cycle_id = 43U, .valid = false});

  REQUIRE(output.cycle_id == 43U);
  REQUIRE(output.fault);
}
