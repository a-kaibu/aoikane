#!/usr/bin/env bash
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
source_dir="$(realpath "$1")"
build="$(realpath -m "$2")"
out="$(realpath -m "$3")"
cmake_bin="${CMAKE:-cmake}"
mkdir -p "$(dirname "$out")"

emcmake "$cmake_bin" -S "$source_dir" -B "$build" -G Ninja \
  -DCMAKE_BUILD_TYPE=MinSizeRel -DOPENZL_INSTALL=OFF -DOPENZL_CPP_INSTALL=OFF \
  -DOPENZL_BUILD_CPP=OFF -DOPENZL_BUILD_CUSTOM_PARSERS=OFF -DOPENZL_BUILD_TOOLS=OFF \
  -DOPENZL_BUILD_CLI=OFF -DOPENZL_BUILD_EXAMPLES=OFF -DOPENZL_ALLOW_INTROSPECTION=OFF \
  -DZSTD_BUILD_SHARED=OFF -DZSTD_MULTITHREAD_SUPPORT=OFF
"$cmake_bin" --build "$build" --target openzl

emcc "$repo/scripts/openzl-decoder.c" -I"$source_dir/include" -I"$build/include" \
  "$build/libopenzl.a" "$build/zstd_build/lib/libzstd.a" "$build/lz4_build/liblz4.a" \
  -Oz -sMEMORY64=2 -sSTANDALONE_WASM -sALLOW_MEMORY_GROWTH=1 -sMAXIMUM_MEMORY=512MB \
  -sEXPORTED_FUNCTIONS='["_aoikane_alloc","_aoikane_free","_aoikane_decode"]' \
  --no-entry -o "$out"
