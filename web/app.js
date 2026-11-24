import { installKernels } from "./kernels.js";
import { initializeVocoderPool } from "./vocoder-pool.js";
import { initializeRendererPool } from "./renderer-pool.js";
import { createModelCache } from "./model-cache.js";

const WASM_URL = "./aoikane.wasm";
const MANIFEST_URL = "./manifest.json";
const englishVoices = new Map([["en-alba", "Alba（英語）"], ["en-aoi", "葵（英語）"], ["en-akane", "茜（英語）"]]);

const els = {
  text: document.querySelector("#text"),
  voice: document.querySelector("#voice"),
  synthesize: document.querySelector("#synthesize"),
  modelInfo: document.querySelector("#model-info"),
  status: document.querySelector("#status"),
  progress: document.querySelector("#progress"),
  result: document.querySelector("#result"),
  audio: document.querySelector("#audio"),
  download: document.querySelector("#download"),
  cues: document.querySelector("#cues"),
  version: document.querySelector("#version"),
};

const sliders = {
  rate: ["#rate", "#rate-value"],
  pitch: ["#pitch", "#pitch-value"],
  volume: ["#volume", "#volume-value"],
  emphasis: ["#emphasis", "#emphasis-value"],
};

const state = {
  aoikane: null,
  englishRuntime: null,
  manifest: null,
  registered: new Set(),
  audioURL: null,
  busy: false,
  lastProgressPaint: 0,
  module: null,
  vocoderAsset: null,
  synthesisAssets: {},
  modelCache: createModelCache(document.baseURI),
};

let startedResolve;
const started = new Promise((resolve) => {
  startedResolve = resolve;
});
globalThis.__aoikaneStarted = startedResolve;

const bootPromise = boot();

for (const [name, [inputSelector, outputSelector]] of Object.entries(sliders)) {
  const input = document.querySelector(inputSelector);
  const output = document.querySelector(outputSelector);
  const update = () => {
    output.value = Number(input.value).toFixed(2);
  };
  input.addEventListener("input", update);
  update();
  sliders[name] = input;
}

els.voice.addEventListener("change", () => {
  updateVoiceControls();
  updateModelInfo();
});

els.synthesize.addEventListener("click", () => {
  synthesize().catch(showError);
});

window.addEventListener("beforeunload", () => {
  if (state.audioURL) URL.revokeObjectURL(state.audioURL);
});

async function boot() {
  els.synthesize.disabled = true;
  setStatus("WASM を読み込み中です…");

  try {
    const go = new Go();
    let instance, module;
    try {
      ({ instance, module } = await WebAssembly.instantiateStreaming(fetch(WASM_URL), go.importObject));
    } catch (streamingError) {
      const response = await fetch(WASM_URL);
      if (!response.ok) {
        throw new Error(`WASM の取得に失敗しました (${response.status})`);
      }
      ({ instance, module } = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject));
    }

    await installKernels(instance.exports.mem);
    go.run(instance).catch((error) => {
      showError(error);
    });
    await started;
    state.aoikane = globalThis.aoikane;
    state.module = module;
    if (!state.aoikane) {
      throw new Error("WASM の初期化に失敗しました");
    }

    const response = await fetch(MANIFEST_URL, { cache: "no-cache" });
    if (!response.ok) {
      throw new Error(`モデル一覧の取得に失敗しました (${response.status})`);
    }
    state.manifest = await response.json();
    if (!Array.isArray(state.manifest.files)) {
      throw new Error("モデル一覧の形式が不正です");
    }

    await state.modelCache.inventory(state.manifest.files);
    if (state.manifest.files.filter((file) => file.group === "english").length === 2) {
      for (const [id, label] of englishVoices) {
        if (state.manifest.files.some((file) => file.group === id)) {
          els.voice.add(new Option(label, id));
        }
      }
    }
    updateVoiceControls();
    updateModelInfo();
    els.version.textContent = `aoikane-wasm ${state.aoikane.version}`;
    els.synthesize.disabled = false;
    setStatus("準備ができました。テキストを入力して「音声を合成」を押してください。");
  } catch (error) {
    showError(error);
  }
}

function requiredFiles(voice) {
  if (!state.manifest) return [];
  if (englishVoices.has(voice)) {
    return state.manifest.files.filter((file) => file.group === "english" || file.group === voice);
  }
  return state.manifest.files.filter((file) => file.group === "shared" || (voice === "akane" && file.group === "akane"));
}

function updateVoiceControls() {
  const isEnglish = englishVoices.has(els.voice.value);
  for (const name of ["rate", "pitch", "emphasis"]) {
    sliders[name].disabled = isEnglish;
  }
}

function updateModelInfo() {
  if (!state.manifest) return;
  const files = requiredFiles(els.voice.value);
  if (files.length === 0) return;
  const remaining = files.reduce((sum, file) => sum + (state.registered.has(file.path) ? 0 : file.size), 0);
  if (remaining === 0) {
    els.modelInfo.textContent = `この声のモデルは読み込み済みです。${state.modelCache.unavailable || state.modelCache.writeFailed ? "端末への保存は利用できないため、次回は再取得する場合があります。" : ""}`;
    return;
  }
  const network = files.reduce((sum, file) => sum + (state.registered.has(file.path) || state.modelCache.stored.has(file.path) ? 0 : file.size), 0);
  if (network === 0) {
    els.modelInfo.textContent = "この声のモデルは端末に保存済みです。モデルを再ダウンロードせずに読み込みます。";
  } else if (state.modelCache.unavailable || state.modelCache.writeFailed) {
    els.modelInfo.textContent = `約 ${formatBytes(network)} を取得します。このブラウザーではモデルを保存できないため、次回も再取得する場合があります。`;
  } else {
    els.modelInfo.textContent = `未保存のモデル（約 ${formatBytes(network)}）を取得して端末に保存します。次回以降は保存済みデータを利用します。`;
  }
}

function startEnglishWorker(onFatal) {
  const worker = new Worker(new URL("./english-worker.js", import.meta.url), { type: "module" });
  let nextID = 0;
  const pending = new Map();
  const close = (error) => {
    worker.terminate();
    for (const { reject } of pending.values()) reject(error);
    pending.clear();
  };
  const fail = (message) => onFatal(new Error(message));
  worker.onerror = (event) => fail(event.message || "英語合成ワーカーが停止しました");
  worker.onmessageerror = () => fail("英語合成ワーカーの応答を読み取れませんでした");
  worker.onmessage = ({ data }) => {
    if (data.fatal) {
      fail(data.fatal);
      return;
    }
    const job = pending.get(data.id);
    if (!job) return;
    pending.delete(data.id);
    if (data.error) job.reject(new Error(data.error));
    else job.resolve(data.result);
  };
  const call = (method, args, transfer = []) => new Promise((resolve, reject) => {
    const id = nextID++;
    pending.set(id, { resolve, reject });
    try { worker.postMessage({ id, method, args }, transfer); }
    catch (error) { fail(error.message); }
  });
  return { call, close };
}

async function startEnglishRuntime() {
  const cores = navigator.hardwareConcurrency || 1;
  const memory = navigator.deviceMemory || 4;
  const limit = memory < 4 ? 1 : Math.max(1, Math.min(memory >= 8 ? 4 : 2, Math.floor(cores / 2)));
  const workers = [];
  const assets = new Map();
  let failed;
  const fail = (error) => {
    if (failed) return;
    failed = error;
    for (const worker of workers) worker.close(error);
    assets.clear();
    state.englishRuntime = null;
    for (const path of state.registered) {
      if (path.startsWith("assets/english/")) state.registered.delete(path);
    }
  };
  const register = (worker, name, original) => {
    const bytes = original.slice();
    return worker.call("registerAsset", [name, bytes], [bytes.buffer]);
  };
  const addWorker = async (primary) => {
    if (failed) throw failed;
    const remove = (error) => {
      worker.close(error);
      const index = workers.indexOf(worker);
      if (index >= 0) workers.splice(index, 1);
    };
    const worker = startEnglishWorker(primary ? fail : remove);
    workers.push(worker);
    try {
      await worker.call("init", [state.module]);
      for (const [name, bytes] of assets) {
        if (!(await register(worker, name, bytes))) throw new Error("英語のモデルを登録できませんでした");
      }
      return worker;
    } catch (error) {
      remove(error);
      throw error;
    }
  };

  let first;
  try { first = await addWorker(true); }
  catch (error) { fail(error); throw error; }
  return {
    registerAsset: async (name, bytes) => {
      const accepted = await Promise.all(workers.map((worker) => register(worker, name, bytes)));
      if (!accepted.every(Boolean)) return false;
      if (limit > 1) assets.set(name, bytes);
      return true;
    },
    warmup: (voice) => Promise.all(workers.map((worker) => worker.call("warmup", [voice]))),
    synthesize: async (requestJSON) => {
      if (limit === 1) return first.call("synthesize", [requestJSON]);
      const chunks = JSON.parse(await first.call("prepareEnglish", [requestJSON]));
      if (chunks.length === 1) return first.call("synthesize", [requestJSON]);

      const request = JSON.parse(requestJSON);
      const count = Math.min(chunks.length, limit);
      await Promise.allSettled(Array.from({ length: Math.max(0, count - workers.length) }, async () => {
        const worker = await addWorker(false);
        try {
          await worker.call("warmup", [request.voice]);
        } catch (error) {
          worker.close(error);
          const index = workers.indexOf(worker);
          if (index >= 0) workers.splice(index, 1);
        }
      }));
      if (failed) throw failed;
      if (workers.length === 1) return first.call("synthesize", [requestJSON]);

      const results = new Array(chunks.length);
      let next = 0, total = 0, jobError;
      await Promise.all(workers.slice(0, count).map(async (worker) => {
        try {
          while (!jobError && next < chunks.length) {
            const index = next++;
            const chunk = chunks[index];
            const result = await worker.call("synthesizeEnglishChunk", [JSON.stringify({ request, chunk, index })]);
            total += result.audio.length - 44;
            if (total > 256 * 1024 * 1024) throw new Error("合成音声が256 MiBを超えました");
            results[index] = result;
          }
        } catch (error) {
          jobError ??= error;
        }
      }));
      if (jobError) {
        if (failed) throw failed;
        return first.call("synthesize", [requestJSON]);
      }
      return joinEnglishChunks(results, total);
    },
  };
}

function joinEnglishChunks(chunks, pcmBytes) {
  const header = 44;
  const audio = new Uint8Array(header + pcmBytes);
  audio.set(chunks[0].audio.subarray(0, header));
  const wav = new DataView(audio.buffer);
  wav.setUint32(4, audio.length - 8, true);
  wav.setUint32(40, pcmBytes, true);
  const bytesPerSecond = wav.getUint32(28, true);
  let offset = header;
  const cues = chunks.map((chunk) => {
    const start = (offset - header) / bytesPerSecond;
    audio.set(chunk.audio.subarray(header), offset);
    offset += chunk.audio.length - header;
    return { text: chunk.cues[0].text, start, end: (offset - header) / bytesPerSecond };
  });
  return { audio, cues };
}

async function loadModels(voice, runtime) {
  const files = requiredFiles(voice);
  if (files.length === 0) {
    throw new Error("モデル一覧が読み込まれていません");
  }

  const total = files.reduce((sum, file) => sum + file.size, 0);
  let done = files.reduce((sum, file) => sum + (state.registered.has(file.path) ? file.size : 0), 0);
  updateProgress(done, total, true);

  for (const file of files) {
    if (state.registered.has(file.path)) continue;
    setStatus(`モデルを読み込み中… ${formatBytes(done)} / ${formatBytes(total)}`);
    let bytes = await state.modelCache.read(file, (inFile, cached) => {
      updateProgress(Math.min(total, done + inFile), total, true, cached);
    });
    if (file.openzl) {
      const { decodeOpenZL } = await import("./openzl.js");
      bytes = await decodeOpenZL(bytes, file.openzl.size);
    }
    if (!(await runtime.registerAsset(file.path, bytes))) {
      throw new Error(`モデルを登録できませんでした: ${file.path}`);
    }
    state.registered.add(file.path);
    if (file.path === "assets/synth/vocoder.onnx") state.vocoderAsset = bytes;
    if (["assets/synth/vocoder.onnx", "assets/synth/featnet.onnx", "assets/synth/compressor.json"].includes(file.path)) state.synthesisAssets[file.path] = bytes;
    done += file.size;
    updateProgress(done, total, true);
  }

  updateProgress(total, total, true);
  setStatus("モデルを準備中…");
  if (!englishVoices.has(voice)) {
    await initializeVocoderPool(state.module, state.vocoderAsset);
    await initializeRendererPool(state.module, state.synthesisAssets);
    state.vocoderAsset = null;
    state.synthesisAssets = {};
  }
  await runtime.warmup(voice);
  els.progress.hidden = true;
  updateModelInfo();
}

async function synthesize() {
  if (state.busy) return;
  await bootPromise;
  if (!state.aoikane) {
    throw new Error("WASM がまだ準備できていません");
  }

  const text = els.text.value.trim();
  if (!text) {
    throw new Error("テキストを入力してください");
  }

  state.busy = true;
  els.synthesize.disabled = true;
  els.status.classList.remove("error");

  try {
    const voice = els.voice.value;
    const isEnglish = englishVoices.has(voice);
    if (isEnglish && !state.englishRuntime) {
      state.englishRuntime = await startEnglishRuntime().catch(() => state.aoikane);
    }
    const runtime = isEnglish ? state.englishRuntime : state.aoikane;
    await loadModels(voice, runtime);

    const startedAt = performance.now();
    const timer = window.setInterval(() => {
      setStatus(`合成中… ${((performance.now() - startedAt) / 1000).toFixed(1)} 秒`);
    }, 100);

    let result;
    try {
      const request = {
        text,
        voice,
        rate: isEnglish ? 1 : Number(sliders.rate.value),
        pitch: isEnglish ? 1 : Number(sliders.pitch.value),
        volume: Number(sliders.volume.value),
        emphasis: isEnglish ? 1 : Number(sliders.emphasis.value),
      };
      result = await runtime.synthesize(JSON.stringify(request));
    } finally {
      window.clearInterval(timer);
    }

    showResult(result);
    const elapsed = (performance.now() - startedAt) / 1000;
    setStatus(`合成しました（${elapsed.toFixed(elapsed < 1 ? 3 : 1)} 秒）。`);
  } finally {
    state.busy = false;
    els.synthesize.disabled = false;
  }
}

function showResult(result) {
  const blob = new Blob([result.audio], { type: "audio/wav" });
  if (state.audioURL) URL.revokeObjectURL(state.audioURL);
  state.audioURL = URL.createObjectURL(blob);
  els.audio.src = state.audioURL;
  els.download.href = state.audioURL;
  els.download.download = `aoikane-${timestamp()}.wav`;

  els.cues.replaceChildren();
  for (const cue of result.cues || []) {
    const item = document.createElement("li");
    const time = document.createElement("time");
    time.textContent = `${formatTime(cue.start)} – ${formatTime(cue.end)}`;
    const label = document.createElement("span");
    label.textContent = cue.text;
    item.append(time, label);
    els.cues.append(item);
  }

  els.result.hidden = false;
  els.audio.play().catch(() => {});
  els.result.scrollIntoView({ behavior: "smooth", block: "nearest" });
}

function setStatus(text) {
  els.status.classList.remove("error");
  els.status.textContent = text;
}

function showError(error) {
  console.error(error);
  els.status.classList.add("error");
  els.status.textContent = `エラー: ${error instanceof Error ? error.message : String(error)}`;
  els.progress.hidden = true;
}

function updateProgress(done, total, visible, cached = false) {
  const now = performance.now();
  if (done < total && now - state.lastProgressPaint < 80) return;
  state.lastProgressPaint = now;
  els.progress.hidden = !visible;
  els.progress.value = total > 0 ? Math.min(1, done / total) : 0;
  if (visible) {
    setStatus(`${cached ? "保存済みモデルを読み込み中" : "モデルを取得中"}… ${formatBytes(done)} / ${formatBytes(total)}`);
  }
}

function formatBytes(bytes) {
  if (bytes >= 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatTime(seconds) {
  const minutes = Math.floor(seconds / 60);
  const rest = (seconds % 60).toFixed(2).padStart(5, "0");
  return `${minutes}:${rest}`;
}

function timestamp() {
  return new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
}
