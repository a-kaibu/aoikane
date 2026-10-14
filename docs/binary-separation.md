# Separate DLL and pure-Go binaries

2026-10-13. Both use the same CLI input, output-name defaults, HTTP client/server and subtitle serializers. The backend is fixed at build time.

| Binary | Build tags | Default local engine | Embedded payload |
|---|---|---|---|
| aoikane | portable | reconstructed Go engine | dictionaries and Go neural models |
| aoikane-dll | none | original Windows SDK | none; requires --engine-dir |
| personal aoikane-dll | personal | original Windows SDK | personal original runtime ZIP |

The portable CLI has no dependency on `internal/bundle`; its engine package excludes `native_windows.go` and uses the unavailable-native stub. Even combining `personal,portable` excludes the personal ZIP. The DLL CLI has no dependency on `assets`, `internal/portable`, `internal/onnx` or `internal/synthesis`. The portable default is `embedded`; `--assets DIR` selects external data. The old `--portable` flag is a compatibility alias in that binary. DLL options and DLL packing are unavailable in the portable binary. Version output names the backend.

`build.ps1` creates both Windows binaries, supports `-PortableOnly` / `-DLLOnly`, and cross-compiles pure-Go binaries with `-Clients`. Release configuration has separate build/archive IDs and filenames. Its YAML parsed successfully; the GoReleaser executable was unavailable for a full `check`. No GitHub Actions run was started.

## Verification

Go 1.26.8 / CGO_ENABLED=0. DLL and portable CLI tests passed; portable engine tests and CLI vet passed. Dependency graphs were checked for the exclusions above. PowerShell build-script syntax parsed successfully.

Separate Windows executables were built and run without backend-selection flags. Both read `零秒。` with `KotonohaAkane_ns_48` and wrote WAV/SRT/VTT. A separated Linux amd64 portable executable did the same. All three WAVs were 145596 bytes, with original SHA-256 `97fe3699bab69e0b3d6054c192cc8ff5746d34eb2ab4ffb885ab67e1271c52a9`, matching the independently captured SDK fixture `internal/portable/testdata/rei_seconds_wave_original.json`. Both subtitle files were byte-identical across all three. Portable `voices` without options listed Aoi and Akane.

The WSL default cache initially ran out of space; the Linux executable was retried with `XDG_CACHE_HOME` pointing to a workspace directory on the host drive. This retained the embedded default engine. This is a CLI/backend separation check, not proof of arbitrary input or CPU/settings fidelity.
