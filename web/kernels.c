// Optional CPU microkernels for the browser. This freestanding module has no
// allocator, stack storage, or private linear memory: pointers refer directly
// to validated Go slices in the imported memory. No model or text leaves it.
#include <wasm_simd128.h>

#define MADD(x, w, a) wasm_f32x4_relaxed_madd((x), (w), (a))

void leaky(float *x, int n, float alpha) {
  v128_t a = wasm_f32x4_splat(alpha), zero = wasm_f32x4_splat(0);
  int i = 0;
  for (; i+4 <= n; i+=4) {
    v128_t v = wasm_v128_load(x+i);
    wasm_v128_store(x+i,wasm_v128_bitselect(wasm_f32x4_mul(v,a),v,wasm_f32x4_lt(v,zero)));
  }
  for (; i<n; i++) if (x[i]<0) x[i]*=alpha;
}

void binary(float *out, const float *a, const float *b, int n, int an, int bn, int op) {
  for (int i=0; i<n; i++) {
    float x=a[an==1 ? 0:i], y=b[bn==1 ? 0:i];
    if (op==1) out[i]=x+y;
    else if (op==2) out[i]=x-y;
    else if (op==3) out[i]=x*y;
    else out[i]=x/y;
  }
}

void chains(float *acc, int rows, const float *x, int base, const int *xoff,
            const float *w0, const float *w1, const float *w2, const float *w3,
            int steps, int wstep, const int *woff, int fused) {
  for (int r = 0; r+4 <= rows; r += 4) {
    v128_t a0 = wasm_v128_load(acc+r);
    v128_t a1 = wasm_v128_load(acc+rows+r);
    v128_t a2 = wasm_v128_load(acc+2*rows+r);
    v128_t a3 = wasm_v128_load(acc+3*rows+r);
    for (int j = 0; j < steps; j++) {
      v128_t xv = wasm_v128_load(x+base+xoff[j]+r);
      int wi = woff ? woff[j] : j*wstep;
      if (fused) {
        a0 = MADD(xv,wasm_f32x4_splat(w0[wi]),a0);
        a1 = MADD(xv,wasm_f32x4_splat(w1[wi]),a1);
        a2 = MADD(xv,wasm_f32x4_splat(w2[wi]),a2);
        a3 = MADD(xv,wasm_f32x4_splat(w3[wi]),a3);
      } else {
        a0 = wasm_f32x4_add(wasm_f32x4_mul(xv,wasm_f32x4_splat(w0[wi])),a0);
        a1 = wasm_f32x4_add(wasm_f32x4_mul(xv,wasm_f32x4_splat(w1[wi])),a1);
        a2 = wasm_f32x4_add(wasm_f32x4_mul(xv,wasm_f32x4_splat(w2[wi])),a2);
        a3 = wasm_f32x4_add(wasm_f32x4_mul(xv,wasm_f32x4_splat(w3[wi])),a3);
      }
    }
    wasm_v128_store(acc+r,a0);
    wasm_v128_store(acc+rows+r,a1);
    wasm_v128_store(acc+2*rows+r,a2);
    wasm_v128_store(acc+3*rows+r,a3);
  }
  // Callers leave up to three scalar tail rows to Go's exact fallback.
}

void conv(float *y, int frames, int channels, const float *x,
          const int *xoff, const float *w, const float *bias,
          int steps, int wcol, int wstep) {
  if (channels == 1) {
    for (int t = 0; t < frames; t += 8) {
      int from = t < frames - 8 ? t : frames - 8;
      v128_t a = wasm_f32x4_splat(bias[0]), b = a;
      for (int j = 0; j < steps; j++) {
        const float *src = x + from + xoff[j];
        v128_t weight = wasm_f32x4_splat(w[j*wstep]);
        a = MADD(wasm_v128_load(src),weight,a);
        b = MADD(wasm_v128_load(src+4),weight,b);
      }
      wasm_v128_store(y+from,a);
      wasm_v128_store(y+from+4,b);
    }
    return;
  }
  for (int c = 0; c < channels; c += 4) {
    for (int t = 0; t < frames; t += 8) {
      int from = t < frames - 8 ? t : frames - 8;
      v128_t a0 = wasm_f32x4_splat(bias[c]), b0 = a0;
      v128_t a1 = wasm_f32x4_splat(bias[c+1]), b1 = a1;
      v128_t a2 = wasm_f32x4_splat(bias[c+2]), b2 = a2;
      v128_t a3 = wasm_f32x4_splat(bias[c+3]), b3 = a3;
      const float *wc = w + c*wcol;
      for (int j = 0; j < steps; j++) {
        const float *src = x + from + xoff[j];
        v128_t xv = wasm_v128_load(src), xu = wasm_v128_load(src+4);
        int wi = j*wstep;
        v128_t v0 = wasm_f32x4_splat(wc[wi]);
        v128_t v1 = wasm_f32x4_splat(wc[wcol+wi]);
        v128_t v2 = wasm_f32x4_splat(wc[2*wcol+wi]);
        v128_t v3 = wasm_f32x4_splat(wc[3*wcol+wi]);
        a0 = MADD(xv,v0,a0); b0 = MADD(xu,v0,b0);
        a1 = MADD(xv,v1,a1); b1 = MADD(xu,v1,b1);
        a2 = MADD(xv,v2,a2); b2 = MADD(xu,v2,b2);
        a3 = MADD(xv,v3,a3); b3 = MADD(xu,v3,b3);
      }
      wasm_v128_store(y+c*frames+from,a0);
      wasm_v128_store(y+c*frames+from+4,b0);
      wasm_v128_store(y+(c+1)*frames+from,a1);
      wasm_v128_store(y+(c+1)*frames+from+4,b1);
      wasm_v128_store(y+(c+2)*frames+from,a2);
      wasm_v128_store(y+(c+2)*frames+from+4,b2);
      wasm_v128_store(y+(c+3)*frames+from,a3);
      wasm_v128_store(y+(c+3)*frames+from+4,b3);
    }
  }
}
