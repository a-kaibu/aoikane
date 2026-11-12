import "./wasm_exec.js";
import { installKernels } from "./kernels.js";

let ready = false;
self.onmessage = async ({ data }) => {
  const { id } = data;
  try {
    if (id === 0) {
      const started = new Promise((resolve) => { globalThis.__aoikaneStarted = resolve; });
      const go = new Go();
      const instance = await WebAssembly.instantiate(data.module, go.importObject);
      await installKernels(instance.exports.mem, data.kernels);
      go.run(instance).catch((error) => self.postMessage({ id: -1, error: String(error) }));
      await started;
      if (!aoikane.registerAsset("assets/synth/vocoder.onnx", data.model)) throw new Error("model asset registration failed");
      await aoikane.warmupVocoder();
      ready = true;
      self.postMessage({ id: 0 });
      return;
    }
    if (!ready) throw new Error("vocoder worker is not ready");
    const audio = await aoikane.vocode(data.request);
    self.postMessage({ id, audio }, [audio.buffer]);
  } catch (error) {
    self.postMessage({ id, error: error instanceof Error ? error.message : String(error) });
  }
};
