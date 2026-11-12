import { installKernels } from "./kernels.js";
import { initializeVocoderPool } from "./vocoder-pool.js";
import { initializeRendererPool } from "./renderer-pool.js";
import { createModelCache } from "./model-cache.js";

const WASM_URL = "./aoikane.wasm";
const MANIFEST_URL = "./manifest.json";

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
  return state.manifest.files.filter((file) => file.group === "shared" || (voice === "akane" && file.group === "akane"));
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

async function loadModels(voice) {
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
    const bytes = await state.modelCache.read(file, (inFile, cached) => {
      updateProgress(Math.min(total, done + inFile), total, true, cached);
    });
    if (!state.aoikane.registerAsset(file.path, bytes)) {
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
  await initializeVocoderPool(state.module, state.vocoderAsset);
  await initializeRendererPool(state.module, state.synthesisAssets);
  state.vocoderAsset = null;
  state.synthesisAssets = {};
  await state.aoikane.warmup(voice);
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
    await loadModels(voice);

    const startedAt = performance.now();
    const timer = window.setInterval(() => {
      setStatus(`合成中… ${((performance.now() - startedAt) / 1000).toFixed(1)} 秒`);
    }, 100);

    let result;
    try {
      const request = {
        text,
        voice,
        rate: Number(sliders.rate.value),
        pitch: Number(sliders.pitch.value),
        volume: Number(sliders.volume.value),
        emphasis: Number(sliders.emphasis.value),
      };
      result = await state.aoikane.synthesize(JSON.stringify(request));
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
