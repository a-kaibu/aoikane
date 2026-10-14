package main

import (
	"github.com/a-kaibu/aoikane/internal/onnx"
	"encoding/json"
	"flag"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Execute the whole original graph from original inputs, without any DLL.
// The captured output includes all seven tensors, not isolated op inputs.
func TestOriginalFeatNetCaptured(t *testing.T) {
	for _, name := range []string{"torch-unary-structured", "torch-controls-f0", "torch-controls-mixed", "torch-controls-overrides"} {
		t.Run(name, func(t *testing.T) { testOriginalFeatNetTrace(t, name) })
	}
}

func testOriginalFeatNetTrace(t *testing.T, name string) {
	trace := filepath.Join("..", "..", "private", name)
	if _, err := os.Stat(filepath.Join(trace, "events.json")); os.IsNotExist(err) {
		t.Skip("private original FeatNet capture unavailable")
	} else if err != nil {
		t.Fatal(err)
	}
	oldFlags, oldArgs := flag.CommandLine, os.Args
	defer func() { flag.CommandLine, os.Args = oldFlags, oldArgs }()
	flag.CommandLine = flag.NewFlagSet("original-featnet-test", flag.ContinueOnError)
	os.Args = []string{"original-featnet-test", "-trace", trace, "-model", "../../assets/synth/featnet.onnx", "-contracts", "../../private/model-trace/model-contracts.json"}
	if err := run(); err != nil {
		t.Fatal(err)
	}
}

func TestOriginalOperatorContinuousReplay(t *testing.T) {
	trace := "../../private/cpu-featnet-layer-local-normal"
	path := filepath.Join(trace, "featnet-operators.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("private original full-layer manifest unavailable")
	}
	oldFlags, oldArgs := flag.CommandLine, os.Args
	defer func() { flag.CommandLine, os.Args = oldFlags, oldArgs }()
	flag.CommandLine = flag.NewFlagSet("operator-test", flag.ContinueOnError)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	os.Args = []string{"operator-test", "-trace", trace, "-model", "../../assets/synth/featnet.onnx", "-contracts", "../../tools/corpus/model-contracts.json", "-original-threads", "1", "-operators", path, "-operator-report", reportPath}
	if err := run(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report operatorReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.Different != 0 || report.First != nil || len(report.Operators) != 118 || report.Values != 6217808 || report.ValueHash != "bf41b3ecb4f13839c85b6ba2d0634a6e6de4a2dfce07fc89ffc79ba1e0744ece" {
		t.Fatal("original full-layer comparison differs")
	}
}

func TestOriginalOperatorDetectsIntermediateDifference(t *testing.T) {
	trace := "../../private/cpu-featnet-layer-local-normal"
	path := filepath.Join(trace, "featnet-operators.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("private original full-layer manifest unavailable")
	}
	m, err := readOperators(path, trace, "../../assets/synth/featnet.onnx", 1)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]onnx.Value{}
	// Start with independently observed references to isolate the verifier:
	// final-output checks cannot detect a corrupt intermediate after execution.
	for _, r := range m.Operators {
		for i, ref := range []tensor{r.Input, r.Output} {
			name := r.InputName
			if i == 1 {
				name = r.OutputName
			}
			v, err := load(trace, ref)
			if err != nil {
				t.Fatal(err)
			}
			got[name] = v
		}
	}
	name := m.Operators[0].OutputName
	v := got[name]
	v.F = append([]float32(nil), v.F...)
	v.F[0] = math.Float32frombits(math.Float32bits(v.F[0]) ^ 1)
	got[name] = v
	report, err := m.compare(trace, got)
	if err != nil {
		t.Fatal(err)
	}
	if report.First == nil || report.First.Operation != 1 || report.First.Port != "output" || report.First.Difference.First != 0 || report.Different < 1 {
		t.Fatal("intermediate bit corruption was not located")
	}
	if _, err = readOperators(path, trace, "../../assets/synth/featnet.onnx", 6); err == nil {
		t.Fatal("mismatched thread profile accepted")
	}
	bad := *m
	bad.ValueHash = "invalid"
	if _, err = bad.compare(trace, got); err == nil {
		t.Fatal("mismatched native value digest accepted")
	}
}

func TestRejectUnsupportedAVX512Configuration(t *testing.T) {
	oldFlags, oldArgs := flag.CommandLine, os.Args
	defer func() { flag.CommandLine, os.Args = oldFlags, oldArgs }()
	for _, args := range [][]string{
		{"profile-test", "-avx512-convolutions", "-original-threads", "6"},
		{"profile-test", "-avx512-convolutions", "-original-threads", "1", "-onnx-order"},
	} {
		flag.CommandLine = flag.NewFlagSet("profile-test", flag.ContinueOnError)
		os.Args = args
		if err := run(); err == nil || err.Error() != "invalid original thread profile or missing operator manifest" {
			t.Fatalf("unsupported combination did not fail before reading capture: %v", err)
		}
	}
}
