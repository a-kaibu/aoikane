let initialization;
export function initializeVocoderPool(module, model) {
  if (initialization) return initialization;
  initialization = startPool(module, model).catch((error) => {
    console.warn("Vocoder workers unavailable; using the main-thread engine", error);
    return false;
  });
  return initialization;
}

async function startPool(module, model) {
  const memoryLimit = navigator.deviceMemory >= 8 ? 6 : navigator.deviceMemory >= 4 ? 3 : 2;
  const cpuLimit = Math.max(0, (navigator.hardwareConcurrency || 2) - 2);
  const count = Math.min(memoryLimit, cpuLimit);
  if (count < 2 || !globalThis.Worker || !module || !model) return false;
  const slots = [];
  const queue = [];
  let nextID = 1;
  let failed = false;
  const fail = (error) => {
    if (failed) return;
    failed = true;
    globalThis.__aoikaneVocoderWorkers = 0;
    globalThis.__aoikaneVocode = undefined;
    for (const slot of slots) {
      clearTimeout(slot.timer);
      slot.rejectReady?.(error);
      slot.job?.reject(error);
      slot.worker.terminate();
    }
    for (const job of queue.splice(0)) job.reject(error);
  };
  const pump = () => {
    if (failed) return;
    for (const slot of slots) {
      if (slot.job || !queue.length) continue;
      const job = queue.shift();
      slot.job = job;
      slot.timer = setTimeout(() => fail(new Error("vocoder worker timed out")), 120_000);
      const buffers = [job.request.f0.buffer, job.request.bap.buffer, job.request.spc.buffer];
      try { slot.worker.postMessage({ id: job.id, request: job.request }, buffers); }
      catch (error) { fail(error); return; }
    }
  };
  try {
    const ready = [];
    for (let i = 0; i < count; i++) {
      const slot = { worker: new Worker(new URL("./vocoder-worker.js", import.meta.url), { type: "module" }), job: null };
      slots.push(slot);
      ready.push(new Promise((resolve, reject) => {
        slot.resolveReady = resolve;
        slot.rejectReady = reject;
        slot.timer = setTimeout(() => fail(new Error("vocoder worker initialization timed out")), 60_000);
      }));
      slot.worker.onerror = (event) => fail(new Error(event.message || "vocoder worker failed"));
      slot.worker.onmessageerror = () => fail(new Error("vocoder worker message could not be decoded"));
      slot.worker.onmessage = ({ data }) => {
        if (data.error) { fail(new Error(data.error)); return; }
        if (data.id === 0) {
          clearTimeout(slot.timer);
          slot.resolveReady();
          return;
        }
        const job = slot.job;
        const valid = data.audio instanceof Uint8Array && data.audio.byteLength === job?.request.frames * 240 * 4;
        if (!job || data.id !== job.id || !valid) {
          fail(new Error("invalid vocoder worker response"));
          return;
        }
        clearTimeout(slot.timer);
        slot.job = null;
        job.resolve(data.audio);
        pump();
      };
      // Each compute-only worker receives one model, never dictionaries.
      const bytes = model.slice();
      slot.worker.postMessage({ id: 0, module, kernels: globalThis.__aoikaneKernelModule, model: bytes }, [bytes.buffer]);
    }
    await Promise.all(ready);
    globalThis.__aoikaneVocoderWorkers = count;
    globalThis.__aoikaneVocode = (request) => new Promise((resolve, reject) => {
      if (failed) { reject(new Error("vocoder worker pool stopped")); return; }
      queue.push({ id: nextID++, request, resolve, reject });
      pump();
    });
    return true;
  } catch (error) {
    fail(error);
    throw error;
  }
}
