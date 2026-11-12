// Go's JS test runner with the same optional kernels installed before package
// initialization. Use: GOROOT=$(go env GOROOT) go test -exec='node ...' ...
const fs = require("node:fs");
const path = require("node:path");
globalThis.fs = fs;
globalThis.crypto ??= require("node:crypto").webcrypto;
require(path.join(process.env.GOROOT, "lib/wasm/wasm_exec.js"));

(async () => {
  const go = new Go();
  go.argv = process.argv.slice(2);
  go.env = process.env;
  let code = 0;
  go.exit = (value) => { code = value; };
  const result = await WebAssembly.instantiate(fs.readFileSync(go.argv[0]), go.importObject);
  const module = await WebAssembly.compile(fs.readFileSync(process.env.TEST_WEB_KERNELS || path.resolve(__dirname, "../dist/kernels.wasm")));
  const source = fs.readFileSync(path.resolve(__dirname, "../web/kernels.js"), "utf8");
  const { installKernels } = await import(`data:text/javascript;base64,${Buffer.from(source).toString("base64")}`);
  if (!await installKernels(result.instance.exports.mem, module)) throw new Error("kernels were not installed");
  await go.run(result.instance);
  process.exit(code);
})().catch((error) => { console.error(error); process.exit(1); });
