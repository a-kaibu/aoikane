#include <stdint.h>
#include <stdlib.h>
#include <openzl/zl_decompress.h>

// Keep the exports wasm32-friendly even though OpenZL is compiled with
// 64-bit pointers. Binaryen lowers the module to 32-bit linear memory.
uint32_t aoikane_alloc(uint32_t size) {
    return (uint32_t)(uintptr_t)malloc(size);
}

void aoikane_free(uint32_t address) {
    free((void *)(uintptr_t)address);
}

int aoikane_decode(uint32_t dst, uint32_t dst_size, uint32_t src, uint32_t src_size) {
    const void *input = (const void *)(uintptr_t)src;
    ZL_Report size = ZL_getDecompressedSize(input, src_size);
    if (ZL_isError(size) || ZL_validResult(size) != dst_size) return 0;
    ZL_Report result = ZL_decompress((void *)(uintptr_t)dst, dst_size, input, src_size);
    return !ZL_isError(result) && ZL_validResult(result) == dst_size;
}
