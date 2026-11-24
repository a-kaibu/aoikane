const { readFileSync } = require("node:fs");
const { join } = require("node:path");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");

const address = process.argv[2] || "http://127.0.0.1:8765/";
const model = process.argv[3] || "english_7m_refl_2026-10";
const text = readFileSync(join(__dirname, "sample.txt"), "utf8").trim();

(async () => {
  const browser = await chromium.launch({
    executablePath: process.env.BROWSER_CHROMIUM_PATH || undefined,
  });
  try {
    const page = await browser.newPage();
    await page.goto(address);
    await page.waitForFunction(() => typeof window.runPocketProbe === "function");
    const result = await page.evaluate(
      ({ model, text }) => window.runPocketProbe(model, text),
      { model, text },
    );
    const runs = result.Runs.map(({ Seconds, AudioSeconds, RTF }) => ({
      seconds: Seconds,
      audioSeconds: AudioSeconds,
      rtf: RTF,
    }));
    if (runs.length !== 3 || runs.some(({ rtf }) => !(rtf > 0 && Number.isFinite(rtf)))) {
      throw new Error("expected three valid synthesis timings");
    }
    const medianRTF = runs.map(({ rtf }) => rtf).sort((a, b) => a - b)[1];
    console.log(JSON.stringify({
      browser: browser.version(),
      model,
      loadSeconds: result.LoadSeconds,
      medianRTF,
      runs,
    }, null, 2));
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
