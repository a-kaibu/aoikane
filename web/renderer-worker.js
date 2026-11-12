import "./wasm_exec.js";
import { installKernels } from "./kernels.js";

let ready = false;
let nextID = 1;
const pending = new Map();

self.onmessage = async ({ data }) => {
  if (data.kind === "vocode-result") {
    const job = pending.get(data.id);
    if (!job) return;
    pending.delete(data.id);
    if (data.error) job.reject(new Error(data.error));
    else job.resolve(data.audio);
    return;
  }
  const { id } = data;
  try {
    if (id === 0) {
      const started = new Promise((resolve) => { globalThis.__aoikaneStarted = resolve; });
      const go = new Go();
      const instance = await WebAssembly.instantiate(data.module, go.importObject);
      await installKernels(instance.exports.mem, data.kernels);
      globalThis.__aoikaneVocoderWorkers = data.vocoderWorkers;
      globalThis.__aoikaneVocode = (request) => new Promise((resolve, reject) => {
        const rpc = nextID++;
        pending.set(rpc, { resolve, reject });
        self.postMessage({ kind: "vocode", id: rpc, request }, [request.f0.buffer, request.bap.buffer, request.spc.buffer]);
      });
      go.run(instance).catch((error) => self.postMessage({ id: -1, error: String(error) }));
      await started;
      for (const [name, bytes] of Object.entries(data.assets)) {
        if (!aoikane.registerAsset(name, bytes)) throw new Error("renderer asset registration failed");
      }
      await aoikane.warmupRenderer();
      ready = true;
      self.postMessage({ id: 0 });
      return;
    }
    if (!ready) throw new Error("renderer worker is not ready");
    const pcm = await aoikane.render(data.request);
    self.postMessage({ id, pcm }, [pcm.buffer]);
  } catch (error) {
    self.postMessage({ id, error: error instanceof Error ? error.message : String(error) });
  }
};
