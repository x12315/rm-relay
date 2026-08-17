.syntax unified
.cpu cortex-m4
.fpu fpv4-sp-d16
.thumb

.section .isr_vector,"a",%progbits
.global g_pfnVectors
.type g_pfnVectors,%object
g_pfnVectors:
  .word _estack
  .word Reset_Handler
  .rept 14
  .word Default_Handler
  .endr
  .rept 82
  .word Default_Handler
  .endr
.size g_pfnVectors, . - g_pfnVectors
