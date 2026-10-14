# Random original comparison: first Windows Actions run

Run: https://github.com/a-kaibu/aoikane-dll/actions/runs/37883278634

Implementation: `1013bce`; generator version 1; seeds 1-4; 8 cases per seed;
maximum 6 units; Aoi default controls; Windows/amd64; CGO_ENABLED=0; Go 1.26.8.

## Observed results

All 32 original calls and all 32 Go calls succeeded. Complete WAV equality:
2/32 (6.25%). Both equal cases are punctuation/whitespace inputs yielding silence.
Every case with spoken audio differed on these runners. This sample is not a
project completion percentage or a population-wide success-rate estimate.

Seven cases have different WAV lengths; 23 have equal lengths but different PCM.
Equal-length PCM differences range from 1 to 360 in signed 16-bit sample values.
There is no tolerance in the test: all 30 differences correctly fail CI.

Runtime restoration and artifact upload succeeded in every job. Credentials
and license files were restored from AOIKANE_ORIGINAL_CREDENTIALS; only inputs,
comparison reports and WAVs were uploaded as result artifacts.

| Seed | CPU | Equal | Length differs | Same length, PCM differs |
| --- | --- | ---: | ---: | ---: |
| 1 | Intel64 Family 6 Model 106 Stepping 6, GenuineIntel | 0 | 2 | 6 |
| 2 | AMD64 Family 25 Model 1 Stepping 1, AuthenticAMD | 1 | 2 | 5 |
| 3 | AMD64 Family 25 Model 1 Stepping 1, AuthenticAMD | 1 | 2 | 5 |
| 4 | AMD64 Family 25 Model 1 Stepping 1, AuthenticAMD | 0 | 1 | 7 |

## Reproduction cases

Use the environment setup in `ci/original-runtime/README.md`, with
AOIKANE_DIFF_COUNT=8 and AOIKANE_DIFF_MAX_RUNES=6, then set the seed and case
index below. Texts are JSON-escaped to preserve newline/tab characters.

| Seed | Case (zero-based) | Text | Original bytes | Go bytes | First differing byte |
| ---: | ---: | --- | ---: | ---: | ---: |
| 1 | 0 | `"うんこぞ。"` | 143564 | 143564 | 5586 |
| 1 | 1 | `"89。"` | 186452 | 186452 | 7492 |
| 1 | 2 | `"−－。？é？。"` | 163244 | 81644 | 4 |
| 1 | 3 | `"行く読む学校ますますの。"` | 294374 | 294374 | 4538 |
| 1 | 4 | `"ふぺきけ。"` | 150284 | 150284 | 6482 |
| 1 | 5 | `"声八一7。"` | 226772 | 229578 | 4 |
| 1 | 6 | `"漢。"` | 110924 | 110924 | 6288 |
| 1 | 7 | `"がを。"` | 119564 | 119564 | 5680 |
| 2 | 0 | `"ぜとてて。"` | 145912 | 145912 | 7068 |
| 2 | 1 | `"四火葉琴ｃ。"` | 201644 | 201644 | 5332 |
| 2 | 2 | `"漢〜 。"` | 110924 | 110924 | 6288 |
| 2 | 3 | `"一人行く魚ABC。"` | 292830 | 292828 | 4 |
| 2 | 4 | `"くざぷうぺ。"` | 167398 | 167398 | 5966 |
| 2 | 5 | `"水土Ａ7二2。"` | 220176 | 279212 | 4 |
| 2 | 6 | `"～\t。"` | 81644 | 81644 | -1 |
| 2 | 7 | `"12が猫。"` | 180650 | 180650 | 7782 |
| 3 | 0 | `"けにーちゆあ。"` | 178012 | 178012 | 6078 |
| 3 | 1 | `"！A。"` | 201644 | 120044 | 4 |
| 3 | 2 | `"？。"` | 81644 | 81644 | -1 |
| 3 | 3 | `"生。"` | 124364 | 124364 | 5066 |
| 3 | 4 | `"ょねめ。"` | 133494 | 133494 | 5758 |
| 3 | 5 | `"開c月下Ｃa。"` | 255000 | 255000 | 7928 |
| 3 | 6 | `"😀０～\n\t\t。"` | 224170 | 142570 | 4 |
| 3 | 7 | `"行く大きな今日。"` | 202512 | 202512 | 5490 |
| 4 | 0 | `"ぺのつ。"` | 127244 | 127244 | 5944 |
| 4 | 1 | `"Ｃ一ｃ人木1。"` | 250136 | 250136 | 8316 |
| 4 | 2 | `"Ж\n。"` | 125804 | 125804 | 7084 |
| 4 | 3 | `"今日。"` | 122924 | 122924 | 9380 |
| 4 | 4 | `"ぺおぐ。"` | 130514 | 130514 | 6462 |
| 4 | 5 | `"？4。"` | 198764 | 117164 | 4 |
| 4 | 6 | `"Ω0é。"` | 157484 | 157484 | 5924 |
| 4 | 7 | `"生開ける駅開ける。"` | 230468 | 230468 | 5550 |

## Next investigations

- Sentence segmentation and empty-reading pauses: seed 3 / case 1, seed 4 /
  case 5, and seed 3 / case 6 differ by 81,600 WAV bytes (40,800 samples,
  850 ms at 48 kHz). The last case also reproduced locally with the same lengths.
  Investigate original grouping before changing how empty sentences are emitted.
- Numeral/Latin-letter processing and duration: seed 1 / case 5 and seed 2 /
  case 5 have larger structural differences. Seed 2 / case 3 differs by one sample.
- Equal-length spoken outputs: compare intermediate reading, duration, acoustic
  model output and waveform stages. Different CPU instruction paths and floating
  point behavior are hypotheses, not established causes. AMD and Intel runners
  both exhibited failures. Do not round PCM, accept a tolerance or replace Go
  synthesis with recorded/native outputs to make the tests pass.

The earlier approximate 90% progress estimate was based on limited selected
cases and does not establish arbitrary-text or cross-CPU waveform equivalence.
The complete pure-Go equivalence goal remains unfinished.


## Prepared sentences CI baseline (2026-10-09)

Run [37884559980](https://github.com/a-kaibu/aoikane-dll/actions/runs/37884559980)
tested commit `785fcce` on eight Windows 2022 shards. All 79 texts in
`tools/corpus/sentences.txt` completed both original and independent Go synthesis;
there were no synthesis errors. Complete WAV equality was **0/79**: 70 had the
same length with PCM differences, and nine Go outputs were one sample shorter.
The unequal-length indices are 8, 31, 34, 37, 39, 49, 56, 58 and 63.

Case 0, `こんにちは。`, has 78,960 samples in both outputs; 21,311 PCM samples differ,
with maximum absolute difference 26. The prior local call for this text matched
all WAV bytes. This demonstrates an unresolved environment-dependent difference;
CPU dispatch and compiler version remain hypotheses, not established causes.
The CI report identifies Go 1.26.8. Shard 2 used AMD Family 26 Model 2;
the other seven used AMD Family 25 Model 1.

Across the 79 files, 4,456,562 of 12,634,122 aligned samples differ. The maximum
absolute aligned difference is 4,760, including the one-sample-length cases;
this does not imply that their upstream waveform amplitudes differ by that amount
without a timing shift. Both WAVs and per-case reports were downloaded to
`private/prepared-original-ci-37884559980/`; `pcm-summary.json` records the
sample counts, differing sample counts and maximum aligned differences.

This CI run predates the local sentence-ending, mixed-integer and decimal
contraction fixes. Those have independent local original comparisons recorded
in `engine-reconstruction.md`. The CI baseline must not be represented as a
completed cross-CPU compatibility test or weakened with a PCM tolerance.


## Original CPU kernel diagnostics

`Original CPU kernel comparison` is a Windows 2022 workflow triggered by updates
to its diagnostic script/workflow, with manual dispatch also supported once the
workflow is present on the default branch. It runs
with two independent runner allocations (actual CPU identifiers are recorded;
there is no guarantee that the CPUs differ). `tools/compare_cpu_original.py`
runs deterministic synthetic inputs through the SHA256-pinned original Torch
DLL and compares every output bit against the audited local reference captures.
It covers Exp, Log, Sin, Sqrt, Tanh, both GEMV orientations, all four GEMM
orientations and the two small-K workers. Log's exhaustive subnormal digest is
also compared. Every kernel is attempted and its result is written immediately;
any exception, parameter mismatch or numeric difference fails the job.

Artifacts contain synthetic kernel captures, mismatch examples, CPU metadata,
and an independent original/Go WAV comparison for `こんにちは。`. The licensed
runtime remains in the job workspace and is never included in artifacts. The
old captures' `worker_rva` annotations identify the locally audited workers;
these scripts call dispatcher entries and do not observe the worker chosen
on a remote CPU. A mismatch will need further dispatch/stage observation.

For prepared case 0, the existing local Go 1.26.3 and CI Go 1.26.8 WAVs have
the same SHA256 `d22ec328e5937a17f1f8159db0d6381d4e81826ffe57e712ff977c6f6539ff93`.
The local original WAV has that same hash, whereas the CI original hash is
`516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2`.
Thus this observed mismatch is between original runtime environments, while
Go output stayed constant. This does not yet establish the causal native
kernel or prove that all other texts behave the same way.


## CPU diagnostic results (2026-10-09)

Run [37890189907](https://github.com/a-kaibu/aoikane-dll/actions/runs/37890189907)
completed on AMD Family 25 Model 1 and Model 17. On each CPU, all 13 synthetic
kernel types matched the audited local references: 919,832 output values,
plus Log's 8,388,607 subnormal inputs. Both original/Go sentence comparisons
still failed, producing exactly the same pair of WAV hashes as the prepared
case-0 CI baseline above. Captures are saved in
`private/cpu-original-ci-37890189907/`.

Locally, Go 1.26.8 still produced an original-identical complete WAV for both
case 0 (`こんにちは。`, 157,964 bytes) and case 8 (`彼は走るのが速いです。`, 258,124
bytes). Local/CI Go outputs are identical for both texts. Case 8's local
original also has 258,124 bytes, whereas the earlier CI original has 258,126.
The length difference therefore comes from the original environments in this
observed case, rather than the Go compiler upgrade. The cause remains unproven;
these isolated-kernel checks do not reproduce the native model's scheduling,
thread partitioning, complete dispatch state or all kernel geometries.

The next diagnostic captures a normal original sentence's model tensors,
intermediate frontend results and ATen thread configuration with the existing
`trace_models.py`. `replay_onnx` compares the captured FeatNet outputs with
pure-Go inference using original kernels, with strict bitwise equality.
`tools/corpus/model-contracts.json` contains only the exported input/output
names. Raw decoded model files are excluded from the diagnostic artifact;
tensors and event metadata remain available to locate the first differing
stage. This replay is diagnostic, while the separate sentence test continues
to require independent text-to-WAV generation on both engines.


## Model-stage localization (2026-10-09)

Run [37890647782](https://github.com/a-kaibu/aoikane-dll/actions/runs/37890647782)
successfully captured normal original inference on both runners. Each original
reported **1 ATen thread**, whereas the same local normal inference reported
**6 threads**. All 15 logical FeatNet input tensors are byte-identical between
local and both CI captures, including dictionary tokens and all controls.
The observed WAV hashes equal the uninstrumented originals on their respective
hosts, so this instrumentation did not create the known WAV difference.

Local Go replay matches all seven original FeatNet outputs exactly. Both CI
replays show the following identical mismatch counts; strict comparison fails
even though the legacy approximate `close` field is true:

| Output | Differing float32 values | Maximum absolute difference |
| --- | ---: | ---: |
| duration | 7 | 0.0001373291015625 |
| phoneme_f0 | 6 | 0.00030517578125 |
| phoneme_power | 3 | 0.00000005960464477539063 |
| f0 | 114 | 0.001068115234375 |
| power | 69 | 0.00000017881393432617188 |
| bap | 686 | 0.00000004470348358154297 |
| spc | 10924 | 0.0000019073486328125 |

The reconstructed original MKL/oneDNN paths currently assume six threads
(`originalMKLPartitions`, `originalDNNLConvPlan`, original Conv/ConvTranspose).
This is an observed configuration difference and the next implementation target;
causality is not yet proven by a matching one-thread Go replay. Do not force the
original to six threads or replace its tensors in the ordinary comparison to
hide the environment difference. Captures are saved in
`private/cpu-stage-ci-37890647782/`.

`replay_onnx` also accepts `-pure-vocoder -original-kernels -strict`, enabling
independent stage comparison without loading ONNX Runtime. Local original
vocoder calls of 26,400 and 23,760 values pass with zero differing bits. Its
FeatNet and vocoder graph caches are separate when both pure flags are used.
The diagnostic CLI cross-builds for Linux CGO=0, but this wrapper retains the
legacy optional native-ORT path and its libc loader; the minimal Docker Desktop
WSL distribution lacked that loader, so wrapper execution there was not verified.
This limitation concerns the diagnostic wrapper, not a new portable-engine
Linux equivalence claim. `go vet` also reports pre-existing `unsafe.Pointer`
usage in its imported optional `internal/infer` FFI; all other vet checks pass
with `-unsafeptr=false`.

## One-thread FeatNet reconstruction (2026-10-09)

The preceding stage mismatch is now reproduced and fixed for FeatNet by selecting
its original numerical thread profile. An isolated local probe uses ATen's normal
`set_num_threads(1)` API before the first FeatNet call (explicit
`trace_models.py --torch-threads 1`; default tracing leaves configuration untouched).
Its final WAV for `こんにちは。` is 157,964 bytes, SHA256
`516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2`,
identical to both ordinary CI originals. Inputs and weights are unchanged.
This establishes the thread configuration as a cause for this case; it does not
establish arbitrary-text waveform equivalence.

`Graph.RunOriginalKernelsWithThreads(..., 1)` preserves the audited packed MKL
accumulation order and selects a single numerical partition. It also reproduces
the single-thread 1x1 convolution backend choice observed in the original.
The selector is consistent with [PyTorch 2.0.1 Convolution.cpp](https://github.com/pytorch/pytorch/blob/v2.0.1/aten/src/ATen/native/Convolution.cpp).
The number of Go workers and shared graph state are unchanged.

Strict replay from the ordinary CI model inputs now matches all seven FeatNet
outputs, 13,366 float32 values, with zero bit differences. The checked-in
`internal/onnx/testdata/thread1_featnet.json` contains those entry inputs and
final outputs, source run 37890647782, and SHA-pinned runtime/model provenance.
The continuous regression uses no original intermediate activations or DLL.
Separate private captures validate all 13 Linear and 100 Conv operations.
These comparisons pass on Windows and Linux amd64 with CGO=0.

The existing six-thread default remains unchanged. Its original kernel
regressions pass, and an ordinary local text-to-WAV comparison for `こんにちは。`
still matches all 157,964 bytes (SHA256
`d22ec328e5937a17f1f8159db0d6381d4e81826ffe57e712ff977c6f6539ff93`).
One-thread wide oneDNN and ConvTranspose profiles explicitly return unsupported
errors pending reconstruction. The normal portable synthesis path still uses
six threads numerically; automatic host profile selection, one-thread vocoder,
and ordinary CI final-WAV equality remain unfinished.

## One-thread vocoder reconstruction (2026-10-09)

The next numerical profile step parameterizes wide oneDNN's cache/block search
and worker balancing, MKL ConvTranspose partitioning, and no-copy N,T GEMM
regions. Existing six-thread arithmetic is retained. Ordinary CI input replay
now matches both original vocoder waves exactly: 26,400 + 23,760 float32 values,
zero differing bits. `testdata/thread1_vocoder.json` records those ordinary
CI inputs and final output bits with pinned provenance. Full Go graph regression
passes on Windows/Linux amd64 with CGO=0 and no native runtime.

An independent isolated local original one-thread capture maps all 126 native
Conv/ConvTranspose operations to model weights/biases. Continuous Go execution
from the three model entry inputs matches every captured input/output/weight/bias,
46,918,503 values, as well as the final 26,400-value wave. Intermediate native
activations are comparison references only. Its final wave matches the ordinary
CI baseline before the manifest is exported. Three existing six-thread continuous
vocoder captures (native, short, tail) also pass unchanged. `go vet ./internal/onnx`
and `git diff --check` pass.

Both thread profiles are available through `RunOriginalKernelsWithThreads` and
`replay_onnx -original-threads`. Ordinary portable synthesis still defaults to
six; passing a profile through the synthesis API, selecting the original host's
default configuration, and whole-WAV CI verification remain the next steps.

## Thread profile connected to text-to-WAV synthesis (2026-10-09)

`portable.Options.OriginalThreads` now passes the selected numerical profile
through both FeatNet and vocoder. `synthesis.OpenWithThreads` supports 1 and 6;
existing `Open` callers and zero-valued portable options retain six. Unsupported
profiles are rejected before assets load. No Go worker/thread settings change.

`TestOriginalThreadProfilesWAV` supplies only `こんにちは。` and the normal Aoi
request. The one-thread complete 157,964-byte WAV matches the ordinary CI
original SHA256 `516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2`;
six matches the ordinary local original SHA256
`d22ec328e5937a17f1f8159db0d6381d4e81826ffe57e712ff977c6f6539ff93`.
Both pass on Windows/Linux amd64 with CGO=0, using no original runtime or native
model inputs. Acoustic-feature regressions and synthesis/portable vet pass.

This is one text with two independently recorded ordinary original references,
not a completed cross-host corpus comparison. Host-default selection is not yet
inferred or changed. The next work is establishing that mapping and rerunning
prepared/random ordinary-original comparisons without configuring the original.

## Ordinary profile observation for corpus comparisons (2026-10-09)

Do not infer the original OpenMP profile from `logical CPUs / 2`. Audited c10.dll
SHA256 `f1f61b24f52f58082cb2d0b63be78f89b06bc699b338c048615b62b548be5b66`
RVA 0x34470 calls MSVCP140's `_Thrd_hardware_concurrency` and shifts right by one,
but the original Torch OpenMP initialization takes a different path.
Torch RVAs 0x68640f0 (`init_num_threads`) and 0x6864050 (`get_num_threads`)
call the MKL maximum-thread getter through 0x1010 -> 0x1000 -> 0x3ddb0.
This matches [PyTorch 2.0.1 ParallelOpenMP.cpp](https://github.com/pytorch/pytorch/blob/v2.0.1/aten/src/ATen/ParallelOpenMP.cpp).
New observation metadata records logical concurrency, c10's default, and MKL's
maximum independently. Local ordinary OMP_NUM_THREADS=8 reports 12, 6, 6 and
ATen 6; its FeatNet still matches the six-thread Go replay exactly. Reading
OMP_NUM_THREADS directly would select the wrong numerical profile in this case.

Normal startup OMP_NUM_THREADS=1 (without an ATen setter) produces the same
157,964-byte original WAV as ordinary CI. The direct original-vs-Go prepared
comparison with the one-thread Go profile also passes, SHA256
`516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2`.
Normal OMP=1 plus MKL_NUM_THREADS=6 reports ATen 6 and the original six-thread
WAV SHA256 `d22ec328e5937a17f1f8159db0d6381d4e81826ffe57e712ff977c6f6539ff93`.

`tools/select_original_profile.py` accepts only completed ordinary FeatNet
observations and audited profiles 1/6. It rejects synthetic thread/control/F0
probes. CPU diagnostics and prepared/random Windows Actions observe the normal
original first, then pass that profile to Go through
`AOIKANE_DIFF_ORIGINAL_THREADS`. This configures only Go's numerical partitioning;
no original configuration is changed. Reports record the chosen profile and
observation metadata. The CPU workflow also checks both vocoder waves strictly.
Decoded model files from corpus preflight stay outside the uploaded report/WAV
directories. Profile selector checks, portable vet, Python compilation, and
Actionlint 1.7.7 pass locally. Runtime automatic detection remains unfinished.

An attempted generalization to 2/3/4/8 was not retained: intermediate ATen
setter probes finished their WAVs but timed out during shutdown, and a normal
OMP=3 capture exposed additional FeatNet differences in the generic partition
trial. OMP=8 was actually capped to six by native MKL. These are not passing
compatibility results. Captures are private; wider numerical profiles require
further original planner observation before support can be claimed.

## Profile-aware CI and remaining differences (2026-10-09)

Run [37895512284](https://github.com/a-kaibu/aoikane-dll/actions/runs/37895512284)
completed after the stage selector was allowed to run despite independent numeric
probe failures. The AMD Family 25 Model 1 job passes all 13 kernels, ordinary
FeatNet (13,366 values), both ordinary vocoder waves (50,160 values), and the
complete ordinary `こんにちは。` WAV with the observed one-thread Go profile.
The Intel Family 6 Model 173 job fails strictly. Earlier run
[37895253823](https://github.com/a-kaibu/aoikane-dll/actions/runs/37895253823) also
sampled Intel Family 6 Model 106 with the same unary-kernel differences.
Both Intel models and AMD report two logical processors, c10 default 1,
MKL maximum 1, and ATen 1. This leaves a CPU arithmetic difference after thread
partitioning is matched, not an unknown thread count.

| Synthetic kernel | Intel differing values / total | AMD differing values |
| --- | ---: | ---: |
| Exp | 290 / 21,810 | 0 |
| Log | 116 / 33,548 | 0 |
| Sin | 8 / 138,474 | 0 |
| Sqrt | 11,594 / 138,210 | 0 |
| Tanh | 172 / 139,794 | 0 |

All eight GEMV/GEMM/small-K probes match on Intel too. Log's exhaustive positive
subnormal digest matches on AMD and differs on Intel. The `worker_rva` fields
in the unary reference scripts are fixed metadata, not observed dispatch targets;
do not infer the selected Intel implementation from those labels. Sqrt's current
Go approximation table explicitly models Ryzen's RSQRTPS, and actual VML dispatch
and seed differences need separate observation. Replacing it with scalar IEEE
sqrt does not fix the captured Intel dataset: 446 of 68,834 positive finite
outputs still differ. Private captures are in `private/cpu-profile-ci-37895512284/`
and `private/cpu-profile-ci-37895253823/`.

Local ordinary OMP=1 seed 1, eight random inputs: seven complete WAVs match;
`漢。` differs at byte 8,042 despite equal 110,924-byte lengths. Original SHA256
`5d7b00b7b10edb89ca667431b17cf9506376810a1ccc256ba2fdf7c8b0681383`,
Go `9a4c5c5e2ebd4d3d7bd6eba4a2f36fe047db2b477ae5f7c6abcf7b71a42eca0b`.
This sample is not a completion percentage. Its complete FeatNet replay matches
all seven outputs, but vocoder has 23,104 differing values out of 24,240.
The all-126-operator continuous diagnostic locates the first difference at
`/Conv_5_output_0`, index 1,009: Go `befdfe25`, original `befdfe26`.
That is channel 1's last frame of a 505-frame convolution. Native call capture
shows seven 72-row N,N GEMMs and a different remaining one-row path; generic MKL
GEMV hooks observe no calls there. The next local reconstruction target is this
oneDNN tail path. Operator capture/manifest is `private/thread1-kan-vocoder-ops/`
(43,264,206 input/output/weight/bias values); GEMM and call traces are in
`private/thread1-kan-vocoder-dnnl/` and `private/thread1-kan-vocoder-calls/`.
`export_vocoder_prefix.py --expected-threads 1` now audits ordinary one-thread
captures explicitly, preserving the default six-thread check.

Ghidra imported the audited Torch without whole-program analysis into
`private/ghidra/threads/torchthreads`. Decompilation of MKL maximum-thread getter
0x3ddb0 identifies `GetLogicalProcessorInformationEx` (RelationAll) and
`GetActiveProcessorCount`, plus dynamic/thread-local/MPI/environment branches.
It counts processor-core records and clamps the OpenMP maximum when dynamic
selection is enabled. This is the concrete basis for future pure-Go host detection;
no guessed `runtime.NumCPU()/2` default was committed.

The four random artifacts from run
[37895253763](https://github.com/a-kaibu/aoikane-dll/actions/runs/37895253763) are
already complete while prepared shards continue. All 32 original and Go calls
succeeded, with nine full WAV matches; one pair has different lengths. Every
report records the observed one-thread profile.

| Seed | CPU | Full WAV matches |
| --- | --- | ---: |
| 1 | AMD Family 25 Model 17 | 1 / 8 |
| 2 | AMD Family 25 Model 1 | 7 / 8 |
| 3 | Intel Family 6 Model 207 | 1 / 8 |
| 4 | Intel Family 6 Model 207 | 0 / 8 |

Seed 2's mismatch is `漢〜 。`. On seed 1, every Go WAV hash equals the local
one-thread Go hash for that same input; every spoken original WAV differs from
the local original. This exposes another original host-dependent difference
on AMD Model 17 and does not establish that all AMD CPUs share one profile.
Its exact model-stage cause remains unproven. Inputs, both WAVs and reports are
saved in `private/profile-random-ci-37895253763/`. These counts are corpus evidence,
not project completion percentages.


### One-row GEMV reconstruction and completed prepared results

The ordinary local one-thread seed-1 comparison now passes all eight inputs
(previously seven). Both independent original and Go synthesis produce the
same complete WAV for `漢。`: 110,924 bytes, SHA256
`5d7b00b7b10edb89ca667431b17cf9506376810a1ccc256ba2fdf7c8b0681383`.
All 126 vocoder operators / 43,264,206 reference values are bit-identical
on Windows and Linux amd64 with CGO=0. Its captured convolution tail and
text-to-WAV hash are now public regression fixtures. Local reports and both
WAVs are in `private/thread1-random-seed1-gemv/`. This local result does not
resolve the differences on the other original CPU variants.

The prepared part of run 37895253763 is also complete (before the GEMV fix).
All 79 original and Go calls succeeded; all pairs have equal byte lengths,
and 19 whole WAVs match. Downloaded reports and both WAVs are in
`private/profile-prepared-ci-37895253763/`.

| Shards | Original CPU | Whole WAV matches |
| --- | --- | ---: |
| 0 / 1 / 2 | AMD Family 26 Model 2 | 0 / 30 |
| 3 / 5 | AMD Family 25 Model 1 | 19 / 20 |
| 4 | Intel Family 6 Model 207 | 0 / 10 |
| 6 / 7 | AMD Family 25 Model 17 | 0 / 19 |

The sole AMD Family 25 Model 1 mismatch is index 43,
`昔々あるところにおじいさんとおばあさんが住んでいました。`:
525,654 bytes, first difference 143,952. Its stage cause has not yet been
located. The other host variants still need actual original dispatch and
stage comparisons; the identical one-thread count alone is insufficient.

Local ordinary OMP=1 replay of seed 2, case 2 (`漢〜 。`) also now matches
the full 110,924-byte original WAV. Reports and both outputs are preserved in
`private/thread1-random-seed2-gemv/`; the earlier CI mismatch has not been
retested on its original runner.

Local ordinary OMP=1 replay of prepared index 43 after the same GEMV fix
also passes: all 525,654 bytes are identical, SHA256
`64df7d229623e17a2971428b8d452bfa5452a570827e734cd09459a72b64b353`.
This is the same original digest recorded on the AMD Family 25 Model 1 CI
runner. Both WAVs/report are in `private/thread1-prepared43-gemv/`;
an independently observed native stage trace is in
`private/thread1-prepared43-stage/` (ATen=1, MKL maximum=1, healthy exit).
The 19/20 CI count above describes the pre-fix run; only its failed text
was rerun locally here, not all 20 cases on that CI host.


### CPU reciprocal-square-root seed diagnostics

`probe_rsqrt.py` preserves its strict Ryzen digest/2048-input-interval audit
by default. `--diagnostic --report ... --output ...bin.gz` records other Windows
x64 CPUs, the authored RSQRTPS instruction bytes, every one of the 16,777,216
normalized float32 inputs in [1,4), and a digest of all uncompressed output bits.
The gzip capture retains every value even if a CPU does not have the Ryzen
quantization intervals. It does not observe the original VML dispatch target.

`compare_sqrt_seed.py` validates that capture and the audited original VML
Sqrt reference, then replays the current Go float32 refinement with both
Ryzen seeds and the actual runner seeds. It reports both difference counts
and examples. A zero host-seed count would support this arithmetic hypothesis;
it does not prove the original uses the same worker. Other unary kernels and
end-to-end comparisons remain strict and continue to fail on a mismatch.

The CPU workflow runs these diagnostics even when its kernel comparison fails,
and saves the compressed seed capture, host report and comparison report.
Python 3.12 / NumPy 2.2.6 are pinned for the native-instruction collector.

Locally, the full seed digest is unchanged:
`bf80a4b9ac32018fee1d8b19dd4ac6173f15585a3e35faa23bb8f9706b31555e`.
A fresh original DLL Sqrt capture (45 cases / 138,210 values, output digest
`3e36d63b58fba88bbc935a9a2c925990adb9d0db3a19919376216c717c58d71a`)
has zero differences using either replay. Inputs include alignment, exceptional
values, subnormals, random normal inputs and high-finite-value boundaries.
Captures and reports are in `private/cpu-rsqrt-local/`. Intel and other AMD
seed/refinement compatibility remain unproven until their runner results arrive.


### Observed seed results: run 37919171908

Both original CPU jobs completed with a strict mismatch. Captures are saved
in `private/cpu-rsqrt-ci-37919171908/` and the failed-step log beside it.

| CPU | RSQRTPS normalized SHA256 | Original Sqrt differences with Ryzen / host seeds |
| --- | --- | ---: |
| Intel Family 6 Model 106 | `daa30c19851bb01752026f3c050489c97b2c2837e7923454739d49b36389e279` | 11,594 / 11,352 |
| AMD Family 25 Model 17 | `bf80a4b9ac32018fee1d8b19dd4ac6173f15585a3e35faa23bb8f9706b31555e` | 0 / 0 |

Both captures retain constant 2,048-input seed intervals. Swapping Intel seeds
alone leaves 11,352 of 138,210 Sqrt outputs different; its original refinement
or dispatch still needs reconstruction. No seed substitution was committed to
the production engine.

On AMD Model 17, all 13 kernel datasets (and Log subnormal digest) match.
Ordinary one-thread FeatNet duration, phoneme_f0, phoneme_power, f0 and power
match, but bap has 633 differing values and spc 10,864. Both vocoder blocks
still differ. Its normal `こんにちは。` WAV digest is
`59d1eec127cc579fd751a025b9d1d85122996436e12233332a5f8bf8bb1af193`.
ATen reports AVX2 usage on both hosts; this does not establish the ISA selected
inside MKL/oneDNN. A temporary 1 MiB oneDNN cache-plan experiment leaves the
same bap/spc differences and does not fix either vocoder block; it was reverted.

The next CPU capture records all 118 observed FeatNet operators (100 Conv,
13 Linear, four Exp and one Log), so the first mismatching layer can be located.
The expanded observer was checked locally with ordinary OMP=1 startup:
118 input/output pairs, healthy process exit, and the unchanged whole WAV
`516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2`.
Its capture is `private/cpu-featnet-layer-local-normal/`. The earlier attempt
with oneDNN-only GEMM hooks observed no such GEMMs and fails that observer's
requested-call gate; the successful workflow uses the plain operation observer.

The complete local portable regression now passes at CGO=0 with a 30-minute
limit (679.563 seconds); the initial 10-minute limit was insufficient for all
private long-WAV fixtures. Full ONNX and synthesis package regressions pass too.


### Continuous FeatNet operator comparison and first host-dependent layer

`export_torch_operations.py --operator-manifest` validates a complete ordinary
118-operator capture, every Conv/Linear weight, Conv bias and Linear bias Add,
and maps the four Exp / one Log in the audited graph order. It writes compact
native tensor references to `featnet-operators.json`, with model/trace/DLL hashes,
thread profile and the ordered input/output value digest. The original dense
`operations.json` / `--only-linears` export remains available.

`tools/verify_original_featnet -original-threads 1 -operators ...
-operator-report ...` requests all those input/output tensors in one graph run
from only the original model entry tensors. Intermediate native tensors are
references only. It validates their trace IDs/shapes/strides, counts, and value
digest, and writes per-operator bit counts plus the first differing input or
output. Bias-bearing Linear outputs are compared at their graph Add outputs.
The final seven model outputs are still checked. A mismatch returns failure.

Windows/Linux amd64 CGO=0 replay of the local ordinary one-thread capture
matches all 118 operators / 6,217,808 input/output values, digest
`bf41b3ecb4f13839c85b6ba2d0634a6e6de4a2dfce07fc89ffc79ba1e0744ece`,
and all 13,366 final values. Regression tests also locate a deliberately
perturbed intermediate bit and reject a wrong thread profile or native-value
digest; existing six-thread/control replay regressions pass. `go vet` and the
workflow validator pass. Pinned Python 3.12 / NumPy 2.2.6 / ONNX 1.18.0 produce
the same manifest digest.

Run [37920590780](https://github.com/a-kaibu/aoikane-dll/actions/runs/37920590780)
completed: AMD Family 25 Model 1 passes all strict checks; AMD Family 26 Model 2
passes all 13 numeric kernel datasets but fails model stages and the complete
WAV. Captures are in `private/cpu-layer-ci-37920590780/`.
The first difference on Family 26 Model 2 is operation 72:
`/model/dec_bap/0.0/conv1/Conv`, output index 0, Go `bf8eeec6`,
original `bf8eeecb` (94,175 / 101,888 output values differ).
All inputs/outputs of operators 1 through 71 and operation 72's input match.
This Conv has 256 input channels, 512 output channels, width 7 and 199 frames.
No native intermediate was supplied as a Go input to this continuous comparison.
Its final FeatNet tensors are bit-identical to the independently captured
AMD Family 25 Model 17 tensors (digest
`be1136bf1b4e817a30c32a545e19607256228918dc4942b2ca40214656ca1229`),
but that does not locate Model 17's first differing layer.

Experimental input-channel blocks 8/16/32/64/128/256, with bias before or after
accumulation, all still differ from that original Conv; matching index 0 alone
is insufficient. The experiment did not change the production kernel.
Native call tracing for operation 72 was verified locally: full 118 input/output
pairs, two call-event batches, healthy exit, and unchanged ordinary whole WAV
SHA256 `516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2`.
This trace is `private/cpu-featnet-calls72-local/`.

The CPU workflow now exports/replays this manifest, saves the JSON layer report
and operation-72 native calls, and uses four hosted replicas to improve hardware
coverage (the runner CPU remains assigned by GitHub). Strict final model/WAV
checks remain enabled, even when a kernel or layer comparison fails.


### Capture the executed JIT kernels, not only DLL call labels

Run [37922552131](https://github.com/a-kaibu/aoikane-dll/actions/runs/37922552131)
completed with three AMD Family 25 Model 1 replicas passing every strict
comparison and one AMD Family 25 Model 17 replica failing. Model 17's first
continuous difference is also operation 72, index 0 (`bf8eeec6` vs `bf8eeecb`),
with 94,224 of 101,888 outputs different. Its count differs from Model 26's
94,175, so matching the first value/final tensors does not establish identical
intermediate arithmetic. Artifacts: `private/cpu-calls-ci-37922552131/`.

The Model 17 call path includes different oneDNN-region functions, including
RVAs `0xaec8140`, `0xaecb0b0`, `0xaecb9d0`, versus the local path's
`0xac5fa80`. Ghidra decompilation of these entry/configuration functions and
JIT dispatch wrappers is in `private/ghidra/thread-c/`. Some earlier candidates
are reorder execution/dispatch, not the convolution arithmetic itself; their
presence alone must not be treated as proof of a selected numeric ISA.

`trace_models.py --torch-call-code` (requires `--torch-call-operation`) now
records actual anonymous executable call targets observed by Stalker. It reads
at most 65,536 bytes from the target within its mapped executable range, records
the capture extent/truncation, and saves content-addressed `jit-<sha256>.bin`
files. The normal observer remains unchanged unless explicitly requested.
No code/model/tensor is substituted, and a requested capture with no observed
JIT target fails its diagnostic gate.

`disassemble_jit_calls.py` validates each code extent/digest and follows static
conditional/unconditional branches from its observed entry point, stopping at
returns and recording undecodable/indirect/out-of-capture branches as unknowns.
It does not execute captured code or follow arbitrary external call targets.
Assembly, instruction digest/counts, mnemonic counts and YMM/ZMM usage are
saved privately. A changed code byte was verified to fail before disassembly.

The ordinary local one-thread operation-72 capture has three executable targets.
Two decode to 113 instructions each with no FMA; the numerical target decodes
to 1,223 instructions, including 608 `vfmadd231ps` instructions and actual YMM
register use (no ZMM). All three have no unknown static branches within this
capture. This establishes the observed local vector width; it does not establish
Model 17/26's ISA until their code is collected.

With this code observer active, all 118 operation input/output pairs, 6,217,808
values, still match the continuous pure-Go replay (digest
`bf41b3ecb4f13839c85b6ba2d0634a6e6de4a2dfce07fc89ffc79ba1e0744ece`),
as do the final seven outputs and entire ordinary WAV
`516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2`.
The process exits normally. Capture/assembly/replay report:
`private/cpu-featnet-jit72-local/`. The CPU workflow now captures/disassembles
the executed targets too, using pinned Capstone 5.0.6. Python compile checks and
workflow validation pass; the production Go kernel is unchanged in this step.


### AVX512 profile audit of the saved AMD Model 17 corpus (2026-10-10)

The ordinary originals from run 37895253763, seed 1, were reused without another
Actions run. Pure Go receives only the eight generated texts. Windows/Linux
amd64 with CGO=0 both reproduce six complete original WAVs. The other two remain
strict failures: case 3 has 14 differing PCM samples and case 5 has 1,283,
each with maximum absolute difference 1 and equal file lengths. All eight Go
WAV hashes agree across Windows/Linux. This is a corpus result, not completion.

The unchanged original text/length/hash references are in
internal/portable/testdata/original_avx512_random.json.
Set AOIKANE_AVX512_REFERENCE_AUDIT=1 and run
go test ./internal/portable -run '^TestOriginalAVX512RandomWAV$' -count=1 -v -timeout 20m.
This opt-in audit includes both unresolved cases and currently returns failure;
it does not rewrite expectations or skip individual mismatches.
AOIKANE_AVX512_REFERENCE_OUTPUT optionally retains every Go WAV.

Live original random/prepared tests accept the explicit boolean
AOIKANE_DIFF_ORIGINAL_AVX512_CONVOLUTIONS with AOIKANE_DIFF_ORIGINAL_THREADS=1.
The report records both fields. This does not automatically infer an ISA from
the CPU identifier, and the six-thread AVX512 combination is rejected.


## 2026-10-10: Akane random and prepared-text comparison

The ordinary original comparison can now select the reconstructed Kansai voice.
Set AOIKANE_DIFF_VOICE=KotonohaAkane_ns_48 for both TestRandomOriginalWAV and
TestPreparedOriginalWAV. An unset variable retains Aoi. Unknown voice IDs fail
before the original engine is opened. Text generation, exact WAV comparison,
and error handling are unchanged; every mismatch still fails.

The workflow_dispatch voice choice applies to both jobs and to the native
profile observation. No new voice matrix or automatic job multiplication was
added. Akane output files live under the selected output root's
KotonohaAkane_ns_48 directory; reproduction metadata also records the voice.
No new Actions run was launched for this change.


Local ordinary Akane synthesis (Windows/amd64, AMD Family 25 Model 33,
Go 1.26.8, CGO=0, six-thread numerical profile) passed all 8 seed-1 inputs,
with a maximum of 6 generator units. The original was called directly through
its ordinary engine API, with no Frida or injected intermediate values.
The actual original and Go files were independently reread: all 8 whole WAVs
are byte-identical, totaling 684,676 PCM samples. This proves the selected
corpus on this native profile; it does not prove arbitrary-text/all-CPU equality.

| Case | Text | Whole WAV bytes | Equal |
| ---: | --- | ---: | --- |
| 0 | うんこぞ。 | 143564 | yes |
| 1 | 89。 | 186452 | yes |
| 2 | −－。？é？。 | 163244 | yes |
| 3 | 行く読む学校ますますの。 | 268584 | yes |
| 4 | ふぺきけ。 | 151724 | yes |
| 5 | 声八一7。 | 225168 | yes |
| 6 | 漢。 | 110924 | yes |
| 7 | がを。 | 120044 | yes |

The prepared-text path also passed Akane corpus case 0 (こんにちは。),
158,444 whole-WAV bytes. Only this selected prepared case was run; this is
not an equality result for all 79 prepared sentences.

Saved ordinary original digests and exact generator/profile metadata are in
internal/portable/testdata/akane_random_original.json. Set
AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT=1 to run TestAkaneRandomReferenceWAV
without the original DLL on supported Go platforms. This longer eight-case
replay is opt-in; it does not add default CI synthesis jobs.

Linux amd64・CGO=0でも保存済み茜ランダムcase=6「漢。」のWAV全体が原本と一致。
Linuxで8件すべてを再実行した結果ではない。macOS ARM64はテストバイナリのコンパイル成功まで。

## 2026-10-13: Preserve full failed shards when rechecking synthesis fixes

`tools/verify_random_wav_shards.py --rechecks MANIFEST` can audit an explicit
complete rerun of an assigned voice/shard. The optional version-1 JSON manifest
contains a `rechecks` list with `voice`, `shard`, `directory`, `log`, `reason`,
`superseded_report_sha256`, and `superseded_log_sha256`. The previous complete
report and log remain on disk and must still match both pinned hashes. The
audit output records the old mismatching indices and the manifest digest.

The rerun must have a successful terminal log, the entire assigned input list,
the same corpus/voice/profile, and exact original/Go WAVs independently matching
the prior native capture. Supplying only the repaired case cannot pass. Without
an explicit recheck manifest, the historical failed log still fails the audit.
The original result files are never rewritten to manufacture a passing shard.

Six integrity tests cover complete reruns, historical failure rejection,
failure-only coverage, changed WAV bytes, changed prior evidence, and duplicate
or unknown assignments. Run `python -m unittest discover -s tools/tests -p
test_verify_random_wav_shards.py`. These tests validate the audit, not speech
synthesis.

For seed19, the prepared manifest is `private/seed19-full-shard-rechecks.json`.
After both final shards and the full Aoi shard-5 rerun finish, require all 256 WAVs:

```powershell
python tools/verify_random_wav_shards.py --root private/random-seed19-full-wave --corpus private/random-seed19-reading-inputs.json --captures tools/compare_reading/testdata --logs private --log-prefix random-seed19-full --shards 8 --rechecks private/seed19-full-shard-rechecks.json --output private/random-seed19-full-audit.json
```

Incomplete runs and any remaining mismatch still produce `complete: false`
and a nonzero exit. No successful 256-case claim is made while these jobs run.

## 2026-10-13: Seed19 full 256-WAV audit completed

Both final voice shards and the complete Aoi shard-5 rerun have now finished
successfully. The strict audit above passed for every assigned input: 128 texts
per voice, 256 complete WAVs, 52,834,199 PCM16 samples. The saved original/Go WAVs
were reread and matched byte for byte, and all 256 independent native capture
files and their report hashes were also rechecked. The original one-sample
failure remains archived and pinned in the explicit full-shard recheck record;
no failing case was removed from coverage. [Full evidence](evidence/seed19-full-wav.json).

This proves the retained seed19 corpus on AMD Family 25 Model 33, Go 1.26.8,
CGO=0, original threads=6 and AVX512 convolutions disabled. It does not establish
arbitrary-input, all-CPU or all-setting fidelity. Separate numeral corpora still
have known reading differences.
