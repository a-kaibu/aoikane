let initialization;

export function initializeRendererPool(module, assets) {
  if (initialization) return initialization;
  initialization = startPool(module, assets).catch((error) => {
    console.warn("Renderer workers unavailable; using main-thread acoustic inference", error);
    return false;
  });
  return initialization;
}

async function startPool(module, assets) {
  if ((navigator.hardwareConcurrency || 2) < 6 || !(navigator.deviceMemory >= 4) || !globalThis.__aoikaneVocode) return false;
  const count = navigator.deviceMemory >= 8 ? 2 : 1;
  const slots = [];
  const queue = [];
  let nextID = 1, failed = false;
  const fail = (error) => {
    if (failed) return;
    failed = true;
    globalThis.__aoikaneRender = undefined;
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
      slot.job = queue.shift();
      slot.timer = setTimeout(() => fail(new Error("renderer worker timed out")), 120_000);
      try { slot.worker.postMessage({ id: slot.job.id, request: slot.job.request }); }
      catch (error) { fail(error); return; }
    }
  };
  try {
    const ready = [];
    for (let i = 0; i < count; i++) {
      const slot = { worker: new Worker(new URL("./renderer-worker.js", import.meta.url), { type: "module" }), job: null };
      slots.push(slot);
      ready.push(new Promise((resolve, reject) => {
        slot.resolveReady = resolve; slot.rejectReady = reject;
        slot.timer = setTimeout(() => fail(new Error("renderer worker initialization timed out")), 60_000);
      }));
      slot.worker.onerror = (event) => fail(new Error(event.message || "renderer worker failed"));
      slot.worker.onmessageerror = () => fail(new Error("renderer worker message could not be decoded"));
      slot.worker.onmessage = ({ data }) => {
        if (data.kind === "vocode") {
          const run = globalThis.__aoikaneVocode;
          const response = run ? run(data.request) : Promise.reject(new Error("vocoder pool stopped"));
          response.then((audio) => {
            if (!failed) slot.worker.postMessage({ kind: "vocode-result", id: data.id, audio }, [audio.buffer]);
          }, (error) => {
            if (!failed) slot.worker.postMessage({ kind: "vocode-result", id: data.id, error: String(error) });
          }).catch(fail);
          return;
        }
        if (data.error) { fail(new Error(data.error)); return; }
        if (data.id === 0) { clearTimeout(slot.timer); slot.resolveReady(); return; }
        const job = slot.job;
        if (!job || data.id !== job.id || !(data.pcm instanceof Uint8Array) || data.pcm.byteLength % 2) {
          fail(new Error("invalid renderer worker response")); return;
        }
        clearTimeout(slot.timer); slot.job = null;
        job.resolve(data.pcm); pump();
      };
      const copies = Object.fromEntries(Object.entries(assets).map(([name, bytes]) => [name, bytes.slice()]));
      slot.worker.postMessage({ id: 0, module, kernels: globalThis.__aoikaneKernelModule, assets: copies, vocoderWorkers: globalThis.__aoikaneVocoderWorkers }, Object.values(copies).map((b) => b.buffer));
    }
    await Promise.all(ready);
    globalThis.__aoikaneRender = (request) => new Promise((resolve, reject) => {
      if (failed) { reject(new Error("renderer worker pool stopped")); return; }
      queue.push({ id: nextID++, request, resolve, reject }); pump();
    });
    return true;
  } catch (error) { fail(error); throw error; }
}
