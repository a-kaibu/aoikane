const CACHE_NAME = "aoikane-models-v1";

// HTTP caches commonly decline files as large as the dictionaries. Cache
// Storage explicitly retains model responses across reloads. Content hashes,
// not application versions, invalidate only the model files that changed.
export function createModelCache(baseURL) {
  const stored = new Set();
  let opening;
  let unavailable = !globalThis.caches || !globalThis.crypto?.subtle;
  let writeFailed = false;
  const hashed = (file) => /^[a-f0-9]{64}$/.test(file.sha256 || "");
  const key = (file) => {
    const url = new URL(file.path, baseURL);
    if (hashed(file)) url.searchParams.set("aoikane_sha256", file.sha256);
    return url.href;
  };
  async function open() {
    if (unavailable) return null;
    if (!opening)
      opening = caches.open(CACHE_NAME).catch((error) => {
        unavailable = true;
        console.warn("Model storage unavailable; using network loading", error);
        return null;
      });
    return opening;
  }
  async function valid(bytes, file) {
    if (bytes.byteLength !== file.size) return false;
    if (!hashed(file) || !globalThis.crypto?.subtle) return true;
    const digest = await crypto.subtle.digest("SHA-256", bytes);
    const actual = Array.from(new Uint8Array(digest))
      .map((v) => v.toString(16).padStart(2, "0"))
      .join("");
    return actual === file.sha256;
  }
  async function readResponse(response, file, progress, cached) {
    if (!response.body || file.size === 0) {
      const bytes = new Uint8Array(await response.arrayBuffer());
      progress(bytes.byteLength, cached);
      return bytes;
    }
    const bytes = new Uint8Array(file.size);
    const reader = response.body.getReader();
    let offset = 0;
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      if (offset + value.byteLength > bytes.length) {
        await reader.cancel();
        throw new Error(`モデルのサイズが一覧と一致しません: ${file.path}`);
      }
      bytes.set(value, offset);
      offset += value.byteLength;
      progress(offset, cached);
    }
    if (offset !== bytes.length)
      throw new Error(`モデルの取得が途中で終了しました: ${file.path}`);
    return bytes;
  }
  return {
    stored,
    get unavailable() {
      return unavailable;
    },
    get writeFailed() {
      return writeFailed;
    },
    async inventory(files) {
      stored.clear();
      const cache = await open();
      if (!cache) return;
      try {
        const keys = new Set(
          (await cache.keys()).map((request) => request.url),
        );
        for (const file of files)
          if (hashed(file) && keys.has(key(file))) stored.add(file.path);
      } catch (error) {
        unavailable = true;
        console.warn("Model cache index unavailable", error);
      }
    },
    async read(file, progress = () => {}) {
      if (!Number.isSafeInteger(file.size) || file.size < 0)
        throw new Error("モデルサイズが不正です");
      const cache = hashed(file) ? await open() : null;
      const url = key(file);
      if (cache) {
        try {
          const response = await cache.match(url);
          if (response) {
            const bytes = await readResponse(response, file, progress, true);
            if (await valid(bytes, file)) {
              stored.add(file.path);
              return bytes;
            }
            await cache.delete(url);
          }
        } catch (error) {
          await cache.delete(url).catch(() => {});
          console.warn("Discarding unreadable model cache", file.path, error);
        }
        stored.delete(file.path);
      }
      const response = await fetch(url, { cache: "default" });
      if (!response.ok)
        throw new Error(
          `モデルの取得に失敗しました (${response.status}): ${file.path}`,
        );
      const bytes = await readResponse(response, file, progress, false);
      if (!(await valid(bytes, file)))
        throw new Error(`モデルのチェックサムが一致しません: ${file.path}`);
      if (cache && !writeFailed) {
        try {
          await cache.put(
            url,
            new Response(bytes, {
              headers: {
                "Content-Type": "application/octet-stream",
                "Content-Length": String(bytes.length),
              },
            }),
          );
          stored.add(file.path);
        } catch (error) {
          writeFailed = true;
          console.warn(
            "Model could not be saved; synthesis continues without persistent storage",
            error,
          );
        }
      }
      return bytes;
    },
  };
}
