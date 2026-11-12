const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");

(async () => {
  const browser = await chromium.launch({ executablePath: process.env.BROWSER_CHROMIUM_PATH || undefined, args: ["--no-sandbox", "--disable-dev-shm-usage"] });
  try {
    const page = await browser.newPage();
    await page.goto(process.argv[2] || "http://127.0.0.1:8000/");
    await page.waitForFunction(() => document.querySelector("#status").textContent.includes("準備ができました"), null, { timeout: 120_000 });
    console.log(JSON.stringify(await page.evaluate(() => ({ hardwareConcurrency: navigator.hardwareConcurrency, deviceMemoryGB: navigator.deviceMemory }))));
    for (const [voice, text] of [["aoi", "こんにちは。今日はいい天気ですね。"], ["akane", "こんにちは。今日はええ天気やね。"]]) {
      const loadingStarted = Date.now();
      await page.selectOption("#voice", voice);
      await page.fill("#text", text);
      await page.click("#synthesize");
      await page.waitForFunction(() => /^(合成しました|エラー)/.test(document.querySelector("#status").textContent), null, { timeout: 180_000 });
      const status = await page.locator("#status").textContent();
      if (status.startsWith("エラー")) throw new Error(status);
      const loadingAndFirstSynthesisMs = Date.now() - loadingStarted;
      const runs = [];
      for (let i = 0; i < 3; i++) {
        runs.push(await page.evaluate(async ({ voice, text }) => {
          const started = performance.now();
          const result = await aoikane.synthesize(JSON.stringify({ voice, text, rate: 1, pitch: 1, volume: 1, emphasis: 1 }));
          const ms = performance.now() - started;
          const hash = Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256", result.audio))).map((v) => v.toString(16).padStart(2, "0")).join("");
          return { ms, hash, bytes: result.audio.length, cues: result.cues };
        }, { voice, text }));
      }
      const medianMs = runs.map((r) => r.ms).sort((a, b) => a - b)[1];
      console.log(JSON.stringify({ voice, text, loadingAndFirstSynthesisMs, medianMs, runs }));
    }
  } finally {
    await browser.close();
  }
})().catch((error) => { console.error(error); process.exitCode = 1; });
