export async function installKernels(memory) {
  try {
    const response = await fetch(new URL("./kernels.wasm", import.meta.url));
    if (!response.ok) return false;
    const { instance } = await WebAssembly.instantiate(await response.arrayBuffer(), { env: { memory } });
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
    return true;
  } catch {
    // Browsers without relaxed SIMD still use the complete Go implementation.
    return false;
  }
}
