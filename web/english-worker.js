import "./wasm_exec.js";

self.onmessage = async ({ data: { id, method, args } }) => {
  try {
    if (method === "init") {
      const started = new Promise((resolve) => { globalThis.__aoikaneStarted = resolve; });
      const go = new Go();
      const instance = await WebAssembly.instantiate(args[0], go.importObject);
      go.run(instance).catch((error) => self.postMessage({ fatal: String(error) }));
      await started;
      self.postMessage({ id });
      return;
    }

    const result = await aoikane[method](...args);
    self.postMessage({ id, result }, result?.audio ? [result.audio.buffer] : []);
  } catch (error) {
    self.postMessage({ id, error: error instanceof Error ? error.message : String(error) });
  }
};
