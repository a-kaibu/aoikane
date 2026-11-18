let decoder;

export async function decodeOpenZL(bytes, size) {
  if (!Number.isSafeInteger(size) || size <= 0 || size > 96 * 1024 * 1024) {
    throw new Error("OpenZLモデルの展開サイズが不正です");
  }
  decoder ??= fetch(new URL("openzl.wasm", import.meta.url)).then(async (response) => {
    if (!response.ok) throw new Error(`OpenZLデコーダーを取得できませんでした (${response.status})`);
    return WebAssembly.compile(await response.arrayBuffer());
  }).catch((error) => {
    decoder = undefined;
    throw error;
  });
  // Keep the compiled module, not the instance's large linear memory.
  const { exports } = await WebAssembly.instantiate(await decoder, {
    env: { emscripten_notify_memory_growth() {} },
    wasi_snapshot_preview1: { fd_close: () => 8, fd_seek: () => 8, fd_write: () => 8 },
  });
  exports._initialize();
  const src = exports.aoikane_alloc(bytes.length);
  const dst = exports.aoikane_alloc(size);
  try {
    if (!src || !dst) throw new Error("OpenZLモデルを展開するメモリが不足しています");
    new Uint8Array(exports.memory.buffer, src, bytes.length).set(bytes);
    if (!exports.aoikane_decode(dst, size, src, bytes.length)) {
      throw new Error("OpenZLモデルを展開できませんでした");
    }
    return new Uint8Array(exports.memory.buffer, dst, size).slice();
  } finally {
    if (src) exports.aoikane_free(src);
    if (dst) exports.aoikane_free(dst);
  }
}
