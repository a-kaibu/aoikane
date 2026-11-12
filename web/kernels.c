// Optional CPU microkernels for the browser. This freestanding module has no
// allocator, stack storage, or private linear memory: pointers refer directly
// to validated Go slices in the imported memory. No model or text leaves it.
#include <wasm_simd128.h>

#define MADD(x, w, a) wasm_f32x4_relaxed_madd((x), (w), (a))

void window_power(float *out,const float *wave,const float *window,int frames) {
  int full=frames/32*32;
  for(int f=0;f<full;f+=4) {
    v128_t a=wasm_f32x4_splat(0),b=a,c=a,d=a;
    for(int j=0;j<2048;j++) {
      const float *s=wave+f*240+j;
      v128_t v=wasm_f32x4_mul(wasm_f32x4_make(s[0],s[240],s[480],s[720]),wasm_f32x4_splat(window[j]));
      a=wasm_f32x4_add(a,wasm_f32x4_mul(v,v));
      if((j+1)%16==0) {
        b=wasm_f32x4_add(b,a);a=wasm_f32x4_splat(0);
        if((j+1)%256==0) {
          c=wasm_f32x4_add(c,b);b=wasm_f32x4_splat(0);
          if((j+1)%4096==0) {d=wasm_f32x4_add(d,c);c=wasm_f32x4_splat(0);}
        }
      }
    }
    wasm_v128_store(out+f,wasm_f32x4_add(wasm_f32x4_add(wasm_f32x4_add(a,b),c),d));
  }
  for(int f=full;f<frames;f++) {
    v128_t a=wasm_f32x4_splat(0),b=a,c=a,d=a;
    for(int row=0;row<512;row++) {
      int j=row*4;
      v128_t v=wasm_f32x4_mul(wasm_v128_load(wave+f*240+j),wasm_v128_load(window+j));
      a=wasm_f32x4_add(a,wasm_f32x4_mul(v,v));
      if((row+1)%16==0) {
        b=wasm_f32x4_add(b,a);a=wasm_f32x4_splat(0);
        if((row+1)%256==0) {c=wasm_f32x4_add(c,b);b=wasm_f32x4_splat(0);}
      }
    }
    a=wasm_f32x4_add(wasm_f32x4_add(wasm_f32x4_add(a,b),c),d);
    float sum=wasm_f32x4_extract_lane(a,0);
    sum+=wasm_f32x4_extract_lane(a,1);sum+=wasm_f32x4_extract_lane(a,2);sum+=wasm_f32x4_extract_lane(a,3);
    out[f]=sum;
  }
}

void projection(float *y,const float *x,const float *w,int inner,int outputs,int row,int count) {
  if(outputs==1) {
    for(int r=row;r<row+count;r++) {
      float sum=0;
      for(int k=0;k<inner;k++) sum=x[r*inner+k]*w[k]+sum;
      y[r]=sum;
    }
    return;
  }
  int packed=count&~3;
  for(int r=row;r<row+packed;r++) {
    for(int c=0;c<outputs;c+=4) {
      v128_t result=wasm_f32x4_splat(0);
      for(int block=0;block<3;block++) {
        int begin=block==0 ? 224:(block-1)*112;
        int end=block==0 ? 256:begin+112;
        v128_t sum=wasm_f32x4_splat(0);
        for(int k=begin;k<end;k++) sum=wasm_f32x4_add(wasm_f32x4_mul(wasm_f32x4_splat(x[r*inner+k]),wasm_v128_load(w+k*outputs+c)),sum);
        result=wasm_f32x4_add(sum,result);
      }
      wasm_v128_store(y+r*outputs+c,result);
    }
  }
  for(int r=row+packed;r<row+count;r++) {
    for(int c=0;c<outputs;c+=4) {
      v128_t a=wasm_f32x4_mul(wasm_f32x4_splat(x[r*inner]),wasm_v128_load(w+c));
      for(int k=1;k<4;k++) a=wasm_f32x4_add(a,wasm_f32x4_mul(wasm_f32x4_splat(x[r*inner+k]),wasm_v128_load(w+k*outputs+c)));
      v128_t b=wasm_f32x4_splat(0),d=b,e=b;
      for(int k=4;k<inner;k+=4) {
        a=wasm_f32x4_add(a,wasm_f32x4_mul(wasm_f32x4_splat(x[r*inner+k]),wasm_v128_load(w+k*outputs+c)));
        b=wasm_f32x4_add(b,wasm_f32x4_mul(wasm_f32x4_splat(x[r*inner+k+1]),wasm_v128_load(w+(k+1)*outputs+c)));
        d=wasm_f32x4_add(d,wasm_f32x4_mul(wasm_f32x4_splat(x[r*inner+k+2]),wasm_v128_load(w+(k+2)*outputs+c)));
        e=wasm_f32x4_add(e,wasm_f32x4_mul(wasm_f32x4_splat(x[r*inner+k+3]),wasm_v128_load(w+(k+3)*outputs+c)));
      }
      v128_t sum=wasm_f32x4_add(wasm_f32x4_add(a,d),wasm_f32x4_add(b,e));
      wasm_v128_store(y+r*outputs+c,wasm_f32x4_add(sum,wasm_f32x4_splat(0)));
    }
  }
}

float madd_probe(float a,float b,float c) {
  return wasm_f32x4_extract_lane(MADD(wasm_f32x4_splat(a),wasm_f32x4_splat(b),wasm_f32x4_splat(c)),0);
}

void lstm_gates(float *out,const float *input,const float *state,const float *w,
                 const float *r,const float *bias,int width,int hidden) {
  int gates=4*hidden;
  for(int g=0;g<gates;g+=4) {
    v128_t a=wasm_f32x4_splat(0),b=a;
    for(int k=0;k<width;k++) a=MADD(wasm_f32x4_splat(input[k]),wasm_v128_load(w+k*gates+g),a);
    for(int k=0;k<hidden;k++) b=MADD(wasm_f32x4_splat(state[k]),wasm_v128_load(r+k*gates+g),b);
    wasm_v128_store(out+g,wasm_f32x4_add(wasm_f32x4_add(a,b),wasm_v128_load(bias+g)));
  }
}

void gemv_normal(float *y,const float *a,const float *x,int m,int n,int lda,int prefix,float alpha) {
  int row=0;
  for(;row<prefix;row++) {
    float sum=y[row];int col=0;
    for(;col+8<=n;col+=8) {
      float first=(a[col*lda+row]*(alpha*x[col]))+sum;
      for(int lane=1;lane<4;lane++) first+=a[(col+lane)*lda+row]*(alpha*x[col+lane]);
      float second=a[(col+4)*lda+row]*(alpha*x[col+4]);
      for(int lane=5;lane<8;lane++) second+=a[(col+lane)*lda+row]*(alpha*x[col+lane]);
      sum=second+first;
    }
    for(;col<n;col++) sum=a[col*lda+row]*(alpha*x[col])+sum;
    y[row]=sum;
  }
  for(;row+4<=m;row+=4) {
    v128_t sum=wasm_v128_load(y+row);
    for(int col=0;col<n;col++) {
      v128_t scaled=wasm_f32x4_splat(alpha*x[col]);
      sum=wasm_f32x4_add(wasm_f32x4_mul(wasm_v128_load(a+col*lda+row),scaled),sum);
    }
    wasm_v128_store(y+row,sum);
  }
  for(;row<m;row++) {
    float sum=y[row];
    for(int col=0;col<n;col++) sum=a[col*lda+row]*(alpha*x[col])+sum;
    y[row]=sum;
  }
}

void gemm_normal(float *y,const float *a,const float *b,const float *initial,int m,int n,
                   int k,int lda,int ldb,int ldc,int align,float alpha,float beta) {
  for(int col=0;col<n;col++) {
    float *out=y+col*m;
    if(beta!=0) {for(int row=0;row<m;row++) {out[row]=initial[col*ldc+row];if(beta!=1)out[row]*=beta;}}
    else {for(int row=0;row<m;row++) out[row]=0;}
    if(alpha!=0) {
      int prefix=(-((align+(col*ldc%4)*4)%16)/4)&3;
      if(m<prefix+4)prefix=0;
      gemv_normal(out,a,b+col*ldb,m,k,lda,prefix,alpha);
    }
  }
}

static void indexed4(float *y,int stride,const float *x,const int *xo,const int *wo,
                     const float *w,const float *bias,int steps,int wc) {
  v128_t a0=wasm_f32x4_splat(bias ? bias[0]:0),a1=wasm_f32x4_splat(bias ? bias[1]:0);
  v128_t a2=wasm_f32x4_splat(bias ? bias[2]:0),a3=wasm_f32x4_splat(bias ? bias[3]:0);
  for(int j=0;j<steps;j++) {
    v128_t xv=wasm_v128_load(x+xo[j]);int wi=wo ? wo[j]:j;
    a0=MADD(xv,wasm_f32x4_splat(w[wi]),a0);
    a1=MADD(xv,wasm_f32x4_splat(w[wc+wi]),a1);
    a2=MADD(xv,wasm_f32x4_splat(w[2*wc+wi]),a2);
    a3=MADD(xv,wasm_f32x4_splat(w[3*wc+wi]),a3);
  }
  wasm_v128_store(y,a0);wasm_v128_store(y+stride,a1);
  wasm_v128_store(y+2*stride,a2);wasm_v128_store(y+3*stride,a3);
}

static void indexed8(float *y,int stride,const float *x,const int *xo,const int *wo,
                     const float *w,const float *bias,int steps,int wc) {
  v128_t a0=wasm_f32x4_splat(bias ? bias[0]:0),b0=a0;
  v128_t a1=wasm_f32x4_splat(bias ? bias[1]:0),b1=a1;
  v128_t a2=wasm_f32x4_splat(bias ? bias[2]:0),b2=a2;
  v128_t a3=wasm_f32x4_splat(bias ? bias[3]:0),b3=a3;
  for(int j=0;j<steps;j++) {
    v128_t xv=wasm_v128_load(x+xo[j]),xu=wasm_v128_load(x+xo[j]+4);int wi=wo ? wo[j]:j;
    v128_t v0=wasm_f32x4_splat(w[wi]),v1=wasm_f32x4_splat(w[wc+wi]);
    v128_t v2=wasm_f32x4_splat(w[2*wc+wi]),v3=wasm_f32x4_splat(w[3*wc+wi]);
    a0=MADD(xv,v0,a0);b0=MADD(xu,v0,b0);
    a1=MADD(xv,v1,a1);b1=MADD(xu,v1,b1);
    a2=MADD(xv,v2,a2);b2=MADD(xu,v2,b2);
    a3=MADD(xv,v3,a3);b3=MADD(xu,v3,b3);
  }
  wasm_v128_store(y,a0);wasm_v128_store(y+4,b0);
  wasm_v128_store(y+stride,a1);wasm_v128_store(y+stride+4,b1);
  wasm_v128_store(y+2*stride,a2);wasm_v128_store(y+2*stride+4,b2);
  wasm_v128_store(y+3*stride,a3);wasm_v128_store(y+3*stride+4,b3);
}

void indexed_conv(float *y,int frames,int channels,const float *x,const int *xo,
                  const int *wo,const float *w,const float *bias,int steps,int wc) {
  for(int c=0;c<channels;c+=4) {
    int t=0;
    for(;t+8<=frames;t+=8) indexed8(y+c*frames+t,frames,x+t,xo,wo,w+c*wc,bias ? bias+c:0,steps,wc);
    for(;t+4<=frames;t+=4) indexed4(y+c*frames+t,frames,x+t,xo,wo,w+c*wc,bias ? bias+c:0,steps,wc);
    if(t<frames) indexed4(y+c*frames+frames-4,frames,x+frames-4,xo,wo,w+c*wc,bias ? bias+c:0,steps,wc);
  }
}

// Packed weights retain the original reduction order but remove the weight
// gather and make all four coefficient streams contiguous in the hot loop.
void ordered_conv(float *y,int frames,int channels,const float *x,const int *xo,
                   const float *w,const float *bias,int steps) {
  for(int c=0;c<channels;c+=4) {
    int t=0;
    for(;t+8<=frames;t+=8) indexed8(y+c*frames+t,frames,x+t,xo,0,w+c*steps,bias ? bias+c:0,steps,steps);
    for(;t+4<=frames;t+=4) indexed4(y+c*frames+t,frames,x+t,xo,0,w+c*steps,bias ? bias+c:0,steps,steps);
    if(t<frames) indexed4(y+c*frames+frames-4,frames,x+frames-4,xo,0,w+c*steps,bias ? bias+c:0,steps,steps);
  }
}

void gather(float *y, const float *x, const int *positions, int channels, int frames) {
  for (int c=0;c<channels;c++) {
    for (int t=0;t<frames;t++) {
      int p=positions[t];
      y[c*frames+t]=p<0 ? 0 : x[c*frames+p];
    }
  }
}

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
  int r=0;
  for (; r+8 <= rows; r+=8) {
    v128_t a0=wasm_v128_load(acc+r), b0=wasm_v128_load(acc+r+4);
    v128_t a1=wasm_v128_load(acc+rows+r), b1=wasm_v128_load(acc+rows+r+4);
    v128_t a2=wasm_v128_load(acc+2*rows+r), b2=wasm_v128_load(acc+2*rows+r+4);
    v128_t a3=wasm_v128_load(acc+3*rows+r), b3=wasm_v128_load(acc+3*rows+r+4);
    for (int j=0;j<steps;j++) {
      v128_t xv=wasm_v128_load(x+base+xoff[j]+r), xu=wasm_v128_load(x+base+xoff[j]+r+4);
      int wi=woff ? woff[j] : j*wstep;
      v128_t v0=wasm_f32x4_splat(w0[wi]),v1=wasm_f32x4_splat(w1[wi]);
      v128_t v2=wasm_f32x4_splat(w2[wi]),v3=wasm_f32x4_splat(w3[wi]);
      if(fused) {
        a0=MADD(xv,v0,a0);b0=MADD(xu,v0,b0);
        a1=MADD(xv,v1,a1);b1=MADD(xu,v1,b1);
        a2=MADD(xv,v2,a2);b2=MADD(xu,v2,b2);
        a3=MADD(xv,v3,a3);b3=MADD(xu,v3,b3);
      } else {
        a0=wasm_f32x4_add(wasm_f32x4_mul(xv,v0),a0);b0=wasm_f32x4_add(wasm_f32x4_mul(xu,v0),b0);
        a1=wasm_f32x4_add(wasm_f32x4_mul(xv,v1),a1);b1=wasm_f32x4_add(wasm_f32x4_mul(xu,v1),b1);
        a2=wasm_f32x4_add(wasm_f32x4_mul(xv,v2),a2);b2=wasm_f32x4_add(wasm_f32x4_mul(xu,v2),b2);
        a3=wasm_f32x4_add(wasm_f32x4_mul(xv,v3),a3);b3=wasm_f32x4_add(wasm_f32x4_mul(xu,v3),b3);
      }
    }
    wasm_v128_store(acc+r,a0);wasm_v128_store(acc+r+4,b0);
    wasm_v128_store(acc+rows+r,a1);wasm_v128_store(acc+rows+r+4,b1);
    wasm_v128_store(acc+2*rows+r,a2);wasm_v128_store(acc+2*rows+r+4,b2);
    wasm_v128_store(acc+3*rows+r,a3);wasm_v128_store(acc+3*rows+r+4,b3);
  }
  for (; r+4 <= rows; r += 4) {
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
