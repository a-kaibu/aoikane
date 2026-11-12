const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");
const fs = require("node:fs");
const path = require("node:path");
const { createHash } = require("node:crypto");

const corpusRoot = path.resolve(__dirname, "../benchmarks");
const selected = (process.env.BENCH_CASES || "").split(",").filter(Boolean);
const allCases = JSON.parse(
  fs.readFileSync(path.join(corpusRoot, "cases.json"), "utf8"),
);
if (selected.some((id) => !allCases.some((item) => item.id === id)))
  throw new Error("unknown benchmark case");
const cases = allCases.filter(
  (item) => !selected.length || selected.includes(item.id),
);
const voices = (process.env.BENCH_VOICES || "aoi,akane").split(",");
const count = Number(process.env.BENCH_RUNS || 3);
if (
  !cases.length ||
  !Number.isInteger(count) ||
  count < 1 ||
  count > 100 ||
  voices.some((v) => !["aoi", "akane"].includes(v))
)
  throw new Error("invalid benchmark selection");
for (const item of cases)
  if (item.file)
    item.text = fs
      .readFileSync(path.join(corpusRoot, item.file), "utf8")
      .trim();
const median = (values) => {
  const v = values.toSorted((a, b) => a - b);
  return v.length % 2
    ? v[v.length >> 1]
    : (v[v.length / 2 - 1] + v[v.length / 2]) / 2;
};

(async () => {
  const browser = await chromium.launch({
    executablePath: process.env.BROWSER_CHROMIUM_PATH || undefined,
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  try {
    let failed = false;
    const page = await browser.newPage();
    await page.goto(process.argv[2] || "http://127.0.0.1:8000/");
    await page.waitForFunction(
      () =>
        document
          .querySelector("#status")
          .textContent.includes("準備ができました"),
      null,
      { timeout: 120_000 },
    );
    console.log(
      JSON.stringify(
        await page.evaluate(() => ({
          hardwareConcurrency: navigator.hardwareConcurrency,
          deviceMemoryGB: navigator.deviceMemory,
        })),
      ),
    );
    for (const voice of voices) {
      const loadingStarted = Date.now();
      await page.selectOption("#voice", voice);
      await page.fill("#text", "こんにちは。今日はいい天気ですね。");
      await page.click("#synthesize");
      await page.waitForFunction(
        () =>
          /^(合成しました|エラー)/.test(
            document.querySelector("#status").textContent,
          ),
        null,
        { timeout: 600_000 },
      );
      const status = await page.locator("#status").textContent();
      if (status.startsWith("エラー")) throw new Error(status);
      const loadingAndFirstSynthesisMs = Date.now() - loadingStarted;
      console.log(
        JSON.stringify({ voice, loadingAndFirstSynthesisMs, phase: "warmup" }),
      );
      for (const item of cases) {
        try {
          const text = item.text || item.texts[voice];
          const runs = [];
          for (let i = 0; i < count; i++) {
            runs.push(
              await page.evaluate(
                async ({ voice, text }) => {
                  const started = performance.now();
                  const result = await aoikane.synthesize(
                    JSON.stringify({
                      voice,
                      text,
                      rate: 1,
                      pitch: 1,
                      volume: 1,
                      emphasis: 1,
                    }),
                  );
                  const ms = performance.now() - started;
                  const hash = Array.from(
                    new Uint8Array(
                      await crypto.subtle.digest("SHA-256", result.audio),
                    ),
                  )
                    .map((v) => v.toString(16).padStart(2, "0"))
                    .join("");
                  const header = new DataView(
                    result.audio.buffer,
                    result.audio.byteOffset,
                    result.audio.byteLength,
                  );
                  const audioSeconds =
                    header.getUint32(40, true) / header.getUint32(28, true);
                  return {
                    ms,
                    hash,
                    bytes: result.audio.length,
                    audioSeconds,
                    sentences: result.cues.length,
                  };
                },
                { voice, text },
              ),
            );
          }
          if (new Set(runs.map((r) => r.hash)).size !== 1)
            throw new Error(`non-deterministic output: ${item.id}`);
          const medianMs = median(runs.map((r) => r.ms));
          console.log(
            JSON.stringify({
              id: item.id,
              title: item.title,
              voice,
              characters: [...text].length,
              textSHA256: createHash("sha256").update(text).digest("hex"),
              source: item.source,
              medianMs,
              audioSeconds: runs[0].audioSeconds,
              medianRTF: medianMs / 1000 / runs[0].audioSeconds,
              runs,
            }),
          );
        } catch (error) {
          failed = true;
          console.log(
            JSON.stringify({
              id: item.id,
              title: item.title,
              voice,
              error: String(error),
            }),
          );
        }
      }
    }
    if (failed) process.exitCode = 1;
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
