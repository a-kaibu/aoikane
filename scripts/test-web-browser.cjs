// Functional browser regression tests, not a performance threshold. Model
// loading and worker startup are deliberately excluded from timing claims.
const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const path = require("node:path");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");

(async () => {
  const root = path.resolve(process.argv[2] || "dist");
  const server = http.createServer((req, res) => {
    const pathname = new URL(req.url, "http://localhost").pathname;
    const file = path.resolve(root, `.${decodeURIComponent(pathname === "/" ? "/index.html" : pathname)}`);
    if (!file.startsWith(root + path.sep)) { res.writeHead(403).end(); return; }
    fs.stat(file, (error, stat) => {
      if (error || !stat.isFile()) { res.writeHead(404).end(); return; }
      const types = { ".wasm": "application/wasm", ".js": "text/javascript", ".html": "text/html", ".json": "application/json", ".css": "text/css", ".svg": "image/svg+xml" };
      res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream", "Content-Length": stat.size });
      fs.createReadStream(file).pipe(res);
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const browser = await chromium.launch({ executablePath: process.env.BROWSER_CHROMIUM_PATH || undefined, args: ["--no-sandbox", "--disable-dev-shm-usage"] });
  const url = `http://127.0.0.1:${server.address().port}/`;
  try {
    const page = await browser.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.addInitScript(() => {
      Object.defineProperty(navigator, "hardwareConcurrency", { get: () => 12 });
      Object.defineProperty(navigator, "deviceMemory", { get: () => 8 });
      const OriginalWorker = Worker;
      globalThis.__testWorkers = [];
      globalThis.Worker = class extends OriginalWorker {
        constructor(url, options) {
          super(url, options);
          __testWorkers.push({ worker: this, url: String(url) });
        }
      };
    });
    await boot(page, url);
    await loadVoice(page, "aoi");
    assert.equal(await page.evaluate(() => globalThis.__aoikaneVocoderWorkers), 6);
    assert.equal(await page.evaluate(() => typeof globalThis.__aoikaneRender), "function");
    const requests = [
      { voice: "aoi", text: "こんにちは。今日はいい天気ですね。" },
      { voice: "aoi", text: "今日はいい天気ですね。こんにちは。", rate: 1.15, pitch: 1.1, volume: 0.8, emphasis: 1.2 },
      { voice: "akane", text: "こんにちは。今日はええ天気やね。" },
      { voice: "akane", text: "急いで帰ろか。大阪では雨やね。", rate: 0.9, pitch: 0.95 },
      { voice: "aoi", text: "。。" },
    ];
    for (const request of requests) {
      await loadVoice(page, request.voice);
      const parallel = await synth(page, request);
      const inline = await synth(page, request, true);
      assert.deepEqual(parallel, inline, `worker/inline mismatch: ${request.voice} ${request.text}`);
      console.log("PASS worker/inline", request.voice, request.text, parallel.bytes);
    }
    // Fail a live worker send, not merely startup. Both queued and active jobs
    // must settle so renderers can use their local vocoder and finish normally.
    const request = requests[0];
    const expected = await synth(page, request, true);
    await page.evaluate(() => {
      const slot = __testWorkers.find((s) => s.url.includes("vocoder-worker"));
      slot.worker.postMessage = () => { throw new Error("injected vocoder send failure"); };
    });
    assert.deepEqual(await synth(page, request), expected, "live vocoder failure did not recover");
    await page.evaluate(() => {
      const slot = __testWorkers.find((s) => s.url.includes("renderer-worker"));
      slot.worker.postMessage = () => { throw new Error("injected renderer send failure"); };
    });
    assert.deepEqual(await synth(page, request), expected, "live renderer failure did not recover");
    assert.deepEqual(errors, [], "unexpected browser errors");
    await page.close();

    const fallback = await browser.newPage();
    await fallback.addInitScript(() => {
      Object.defineProperty(navigator, "hardwareConcurrency", { get: () => 12 });
      Object.defineProperty(navigator, "deviceMemory", { get: () => 8 });
      globalThis.Worker = class { constructor() { throw new Error("injected unsupported worker"); } };
    });
    await boot(fallback, url);
    await loadVoice(fallback, "aoi");
    assert.notEqual(await fallback.evaluate(() => typeof globalThis.__aoikaneRender), "function");
    assert.deepEqual(await synth(fallback, request), expected, "worker initialization fallback mismatch");
    await fallback.close();
    console.log("Browser workers, controls, sentence order, voice switching, and fallbacks: PASS");
  } finally {
    await browser.close();
    await new Promise((resolve) => server.close(resolve));
  }
})().catch((error) => { console.error(error); process.exitCode = 1; });

async function boot(page, url) {
  await page.goto(url);
  await page.waitForFunction(() => {
    const s = document.querySelector("#status").textContent;
    return s.includes("準備ができました") || s.startsWith("エラー");
  }, null, { timeout: 120_000 });
  assert.ok(!(await page.locator("#status").textContent()).startsWith("エラー"));
}

async function loadVoice(page, voice) {
  // The UI owns model loading; synth() below deliberately exercises the Go API.
  await page.selectOption("#voice", voice);
  await page.fill("#text", "こんにちは。");
  await page.click("#synthesize");
  await page.waitForFunction(() => {
    const s = document.querySelector("#status").textContent;
    return s.startsWith("合成しました") || s.startsWith("エラー");
  }, null, { timeout: 180_000 });
  assert.ok((await page.locator("#status").textContent()).startsWith("合成しました"));
}

async function synth(page, request, inline = false) {
  return page.evaluate(async ({ request, inline }) => {
    const saved = [globalThis.__aoikaneRender, globalThis.__aoikaneVocode, globalThis.__aoikaneVocoderWorkers];
    if (inline) {
      globalThis.__aoikaneRender = undefined;
      globalThis.__aoikaneVocode = undefined;
      globalThis.__aoikaneVocoderWorkers = 0;
    }
    try {
      const result = await aoikane.synthesize(JSON.stringify({ rate: 1, pitch: 1, volume: 1, emphasis: 1, ...request }));
      const header = new TextDecoder().decode(result.audio.subarray(0, 4));
      if (header !== "RIFF") throw new Error("invalid WAV header");
      const hash = Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256", result.audio))).map((v) => v.toString(16).padStart(2, "0")).join("");
      return { hash, bytes: result.audio.length, cues: result.cues };
    } finally {
      if (inline) [globalThis.__aoikaneRender, globalThis.__aoikaneVocode, globalThis.__aoikaneVocoderWorkers] = saved;
    }
  }, { request, inline });
}
