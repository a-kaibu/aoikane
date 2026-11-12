// Runs without npm dependencies. Exercise the optional shared-memory module
// independently of Go, including misaligned tails and memory.grow().
const fs = require("node:fs");
const assert = require("node:assert/strict");

(async () => {
  const memory = new WebAssembly.Memory({ initial: 32 });
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(process.argv[2] || "dist/kernels.wasm"), { env: { memory } });
  const k = instance.exports;
  let top;
  const alloc = (n, integers = false) => {
    const ptr = top;
    top += n * 4 + 16;
    return { ptr, data: integers ? new Int32Array(memory.buffer, ptr, n) : new Float32Array(memory.buffer, ptr, n) };
  };
  let seed = 123;
  const fill = (array) => {
    for (let i = 0; i < array.length; i++) {
      seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0;
      array[i] = (seed % 65536 - 32768) / 32768;
    }
  };
  const close = (a, b, label) => {
    assert.ok(Number.isFinite(a) && Math.abs(a - b) <= 2e-5 * Math.max(1, Math.abs(b)), `${label}: ${a} != ${b}`);
  };
  for (const frames of [24, 25, 31, 32, 77, 256]) {
    for (const channels of [1, 4, 8, 12]) {
      for (const step of [1, 4]) {
        top = 64;
        const steps = 19, wcol = steps * step;
        const x = alloc(frames + steps), offsets = alloc(steps, true);
        const w = alloc(channels * wcol), bias = alloc(channels), y = alloc(channels * frames + 8);
        fill(x.data); fill(w.data); fill(bias.data);
        offsets.data.forEach((_, j) => { offsets.data[j] = j; });
        y.data.fill(12345);
        k.conv(y.ptr, frames, channels, x.ptr, offsets.ptr, w.ptr, bias.ptr, steps, wcol, step);
        for (let c = 0; c < channels; c++) {
          for (let t = 0; t < frames; t++) {
            let expected = bias.data[c];
            for (let j = 0; j < steps; j++) expected = Math.fround(x.data[t+j] * w.data[c*wcol+j*step] + expected);
            close(y.data[c*frames+t], expected, `conv ${channels}x${frames}/${step}`);
          }
        }
        assert.ok(y.data.slice(channels * frames).every(v => v === 12345), "conv overwrote tail");
      }
    }
  }
  for (const rows of [4, 7, 8, 12, 24, 255, 256]) {
    for (const step of [0, 1, 3]) {
      for (const indexed of [false, true]) {
        for (const fused of [false, true]) {
          top = 64;
          const steps = 7, base = 3;
          const x = alloc(rows+30), xoff = alloc(steps, true), woff = alloc(steps, true);
          xoff.data.set([-3, 0, 4, 17, 9, 8, 2]);
          woff.data.set([5, 2, 8, 0, 10, 1, 4]);
          const weights = Array.from({length:4}, () => alloc(24));
          const acc = alloc(rows * 4), before = new Float32Array(acc.data.length);
          fill(x.data); weights.forEach(w => fill(w.data)); fill(acc.data); before.set(acc.data);
          k.chains(acc.ptr, rows, x.ptr, base, xoff.ptr, ...weights.map(w=>w.ptr), steps, step, indexed ? woff.ptr : 0, Number(fused));
          for (let c=0;c<4;c++) for (let r=0;r<rows;r++) {
            let expected = before[c*rows+r];
            if (r < (rows & ~3)) {
              for(let j=0;j<steps;j++) {
                const v=x.data[base+xoff.data[j]+r], w=weights[c].data[indexed ? woff.data[j] : j*step];
                expected=Math.fround((fused ? v*w : Math.fround(v*w)) + expected);
              }
            }
            if (fused) close(acc.data[c*rows+r],expected,`chains ${rows}/${step}/${indexed}/${fused}`);
            else assert.equal(acc.data[c*rows+r], expected, 'unfused chains');
          }
        }
      }
    }
  }
  for (const n of [128, 129, 1001]) {
    for (const an of [1,n]) for (const bn of [1,n]) {
      top=64;
      const a=alloc(an), b=alloc(bn), out=alloc(n);
      fill(a.data);fill(b.data);
      for(let i=0;i<bn;i++) if(Math.abs(b.data[i])<0.01) b.data[i]=0.01;
      for(let op=1;op<=4;op++) {
        k.binary(out.ptr,a.ptr,b.ptr,n,an,bn,op);
        for(let i=0;i<n;i++) {
          const x=a.data[an===1 ? 0:i], y=b.data[bn===1 ? 0:i];
          const expected=Math.fround(op===1 ? x+y : op===2 ? x-y : op===3 ? x*y : x/y);
          assert.equal(out.data[i], expected, 'binary');
        }
      }
      const original=Float32Array.from(out.data);
      k.leaky(out.ptr,n,0.125);
      for(let i=0;i<n;i++) assert.equal(out.data[i], original[i]<0 ? Math.fround(original[i]*0.125) : original[i]);
    }
  }
  memory.grow(1);
  const x = new Float32Array(memory.buffer, 64, 8);
  x.fill(-2);
  k.leaky(64,8,0.125);
  assert.ok(x.every(v=>v===-0.25),"kernel lost grown memory");
  console.log("WebAssembly shared-memory kernels: PASS");
})().catch(error => { console.error(error); process.exitCode=1; });
