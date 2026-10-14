# SRT / WebVTT output

`aoikane synth --portable embedded --file script.txt -o speech.wav --srt speech.srt --vtt speech.vtt`

The `synth` command can be omitted. `aoikane script.txt --portable embedded --srt --vtt` derives all three paths from the input (`script.wav`, `script.srt`, `script.vtt`). An explicit audio output provides the default subtitle stem. `--format mp3` derives `script.mp3`. Text/stdin uses `speech` as the default stem. Both flags are optional and can be combined. Files are UTF-8. The portable engine measures each sentence boundary from the PCM produced by the same synthesis, including begin/end pauses. It does not resynthesize sentences or alter audio. Empty initial sentences retain their silence offset. Repeated sentence punctuation follows the existing frontend sentence extraction. Timestamps round to milliseconds consistently, so adjacent cues share a boundary.

Native DLL and HTTP backends currently emit one cue covering the whole text and actual WAV duration. MP3 timing uses the PCM before encoding; player handling of encoder delay is outside this timing. Output paths must be distinct; subtitles cannot use stdout.

## Verification (2026-10-13)

Go 1.26.8, CGO_ENABLED=0, Windows amd64, AMD Family 25 Model 33, six-thread original numerical profile. Timed synthesis reproduced four complete original WAVs (both voices, single and multiple sentences) using the existing independently captured SHA-256 fixtures in `internal/portable/testdata/zero_unit_seconds_wave_original.json` and `tab_sentence_wave_original.json`.

- `0京秒1。`: both voices, 175184 bytes, SHA-256 `720f38c865d54aef04297d01ebf330c77901e0566d034b7a479f3fa66f2516cf`, cue 0–1.824375 seconds.
- `あ。 。い。`: Akane, 274604 bytes, SHA-256 `46dc082bb156ea3c63731907126273d5fdd91f007c0516ad82990452c2632d4f`, boundaries 0 / 1.030 / 1.880 / 2.860 seconds.
- Same text: Aoi, 273644 bytes, SHA-256 `b57d0fe7f0f81312421bce04b360bf7698f4b0c7b7b0bab84966c328e5cbee19`, boundaries 0 / 1.025 / 1.875 / 2.850 seconds.

The built CLI also emitted both subtitle files for the Akane multiple-sentence case; the WAV was independently rehashed and both subtitle files matched the expected UTF-8 contents. Serializer and HTTP CLI tests cover timestamp continuity, escaping, invalid cues, output collisions, and WAV/MP3 sidecars. Run the original WAV tests with `AOIKANE_SUBTITLE_REFERENCE_AUDIT=1 go test ./internal/portable -run TestTimed -count=1 -v`.

The pure-Go audio tests also passed on Linux amd64 under WSL.

`go vet` passed for the changed audio/CLI/portable packages. An additional check of the unchanged native engine package reports its existing unsafe.Pointer warnings.

## Default CLI paths (2026-10-13)

The built CLI was invoked without `synth` or output paths: `aoikane private/cli-defaults-input.txt --portable embedded --voice KotonohaAkane_ns_48 --srt --vtt`. It wrote the three files next to the input. The input was `零秒。`; the full 145596-byte WAV matched the independently captured original SHA-256 `97fe3699bab69e0b3d6054c192cc8ff5746d34eb2ab4ffb885ab67e1271c52a9`. Both UTF-8 subtitles had the expected cue 0–1.516 seconds. CLI tests cover positional filenames, `--file` / `--input`, `-o` / `--output`, inferred MP3 paths, explicit subtitle paths, omitted `synth`, legacy `synth`, bare invocation with stdin, help and version.
