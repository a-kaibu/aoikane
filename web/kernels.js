export async function installKernels(memory, compiled) {
  try {
    if (!compiled) {
      const response = await fetch(new URL("./kernels.wasm", import.meta.url));
      if (!response.ok) return false;
      compiled = await WebAssembly.compile(await response.arrayBuffer());
    }
    const instance = await WebAssembly.instantiate(compiled, { env: { memory } });
    if (!["conv", "chains", "leaky", "binary"].every((name) => typeof instance.exports[name] === "function")) return false;
    globalThis.__aoikaneConvKernel = (...args) => {
      instance.exports.conv(...args);
      return true;
    };
    globalThis.__aoikaneChainKernel = (...args) => {
      instance.exports.chains(...args);
    };
    globalThis.__aoikaneLeakyKernel = instance.exports.leaky;
    globalThis.__aoikaneBinaryKernel = instance.exports.binary;
    globalThis.__aoikaneGatherKernel = instance.exports.gather;
    globalThis.__aoikaneIndexedConvKernel = instance.exports.indexed_conv;
    globalThis.__aoikaneOrderedConvKernel = instance.exports.ordered_conv;
    globalThis.__aoikaneGEMVNormalKernel = instance.exports.gemv_normal;
    globalThis.__aoikaneGEMMNormalKernel = instance.exports.gemm_normal;
    globalThis.__aoikaneProjectionKernel = instance.exports.projection;
    globalThis.__aoikanePowerKernel = instance.exports.window_power;
    // Boundary classification needs fused rounding, unlike opt-in acoustic
    // fast math. Do not use relaxed SIMD here on an unfused implementation.
    if (instance.exports.madd_probe?.(1 + 2 ** -23, 1 - 2 ** -23, -1) === -(2 ** -46)) {
      globalThis.__aoikaneLSTMGatesKernel = instance.exports.lstm_gates;
    }
    globalThis.__aoikaneKernelModule = compiled;
    return true;
  } catch {
    // Browsers without relaxed SIMD still use the complete Go implementation.
    return false;
  }
}
