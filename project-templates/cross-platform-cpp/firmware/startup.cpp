#include <cstdint>

extern "C" {

extern std::uint32_t _sidata;
extern std::uint32_t _sdata;
extern std::uint32_t _edata;
extern std::uint32_t _sbss;
extern std::uint32_t _ebss;

int main();

[[noreturn]] void Default_Handler() {
  while (true) {
  }
}

[[noreturn]] void Reset_Handler() {
  constexpr std::uintptr_t scb_cpacr_address = 0xE000ED88U;
  constexpr std::uint32_t enable_cp10_cp11 = 0x00F00000U;
  auto* const scb_cpacr = reinterpret_cast<volatile std::uint32_t*>(scb_cpacr_address);
  *scb_cpacr |= enable_cp10_cp11;
  asm volatile("dsb");
  asm volatile("isb");

  auto* source = &_sidata;
  for (auto* destination = &_sdata; destination < &_edata; ++destination) {
    *destination = *source++;
  }
  for (auto* destination = &_sbss; destination < &_ebss; ++destination) {
    *destination = 0U;
  }
  static_cast<void>(main());
  while (true) {
  }
}

}  // extern "C"
