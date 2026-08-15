package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

func TestReadStrictJSONRejectsUnknownField(t *testing.T) {
	path := writeTemp(t, `{"known":"value","extra":true}`)
	var target struct {
		Known string `json:"known"`
	}
	if err := readStrictJSON(path, &target); err == nil {
		t.Fatal("readStrictJSON accepted unknown field")
	}
}

func TestReadStrictJSONRejectsTrailingValue(t *testing.T) {
	path := writeTemp(t, `{"known":"value"} {"second":true}`)
	var target struct {
		Known string `json:"known"`
	}
	if err := readStrictJSON(path, &target); err == nil {
		t.Fatal("readStrictJSON accepted trailing JSON value")
	}
}

func TestReadStrictJSONRejectsDuplicateField(t *testing.T) {
	path := writeTemp(t, `{"known":"first","known":"second"}`)
	var target struct {
		Known string `json:"known"`
	}
	if err := readStrictJSON(path, &target); err == nil {
		t.Fatal("readStrictJSON accepted duplicate field")
	}
}

func TestWriteJSONRefusesExistingOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.json")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(path, map[string]string{"state": "replacement"}); err == nil {
		t.Fatal("writeJSON() overwrote an existing receipt")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Fatalf("existing output changed to %q", data)
	}
}

func TestContaminationFileWritesBlockingReport(t *testing.T) {
	comparison := axmmirror.EvaluationComparison{
		Schema: axmmirror.EvaluationComparisonSchema,
		Training: axmmirror.EvidenceInventory{
			Complete: true, RecordSHA256: []string{repeat("a")},
		},
		Evaluation: axmmirror.EvidenceInventory{
			Complete: true, RecordSHA256: []string{repeat("a")},
		},
		Authorship: axmmirror.AuthorshipDeclaration{State: "unknown"},
	}
	data, err := json.Marshal(comparison)
	if err != nil {
		t.Fatal(err)
	}
	input := writeTemp(t, string(data))
	output := filepath.Join(t.TempDir(), "report.json")
	if err := contaminationFile(input, output); err == nil {
		t.Fatal("contaminationFile() did not block an overlap")
	}
	var report axmmirror.ContaminationReport
	if err := readStrictJSON(output, &report); err != nil {
		t.Fatalf("read report: %v", err)
	}
	if report.State != axmmirror.ContaminationStateContaminated {
		t.Fatalf("report state = %q", report.State)
	}
}

func TestLensCorpusFileWritesReadyReceipt(t *testing.T) {
	input := filepath.Join("..", "..", "examples", "axm-mirror", "corpus-bom.json")
	output := filepath.Join(t.TempDir(), "corpus-lens.json")
	if err := lensCorpusFile(input, output); err != nil {
		t.Fatalf("lensCorpusFile() error = %v", err)
	}
	var lens axmmirror.CorpusEvidenceLens
	if err := readStrictJSON(output, &lens); err != nil {
		t.Fatalf("read corpus lens: %v", err)
	}
	if lens.State != axmmirror.CorpusEvidenceStateReady {
		t.Fatalf("corpus lens state = %q", lens.State)
	}
}

func TestWitnessRunFileWritesHeldLifecycleReceipt(t *testing.T) {
	runBOM := filepath.Join("..", "..", "examples", "axm-mirror", "run-bom.json")
	run := writeTemp(t, `{
  "kind":"waldo-training-run",
  "schema":1,
  "id":"example-real-run-0001",
  "state":"planned",
  "bom_sha256":"176558b03e78383529a40fbdee15849d53a96c6272171aeb344df06926aaf0b6",
  "planned":"2026-08-15T03:00:00Z"
}`)
	output := filepath.Join(t.TempDir(), "run-witness.json")
	if err := witnessRunFile(runBOM, run, output); err == nil {
		t.Fatal("witnessRunFile() did not return a HOLD error")
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(output, &witness); err != nil {
		t.Fatalf("read run witness: %v", err)
	}
	if witness.State != axmmirror.RunWitnessStateHold || witness.RunState != "planned" || len(witness.Holds) == 0 {
		t.Fatalf("run witness = %+v", witness)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func repeat(ch string) string {
	value := ""
	for i := 0; i < 64; i++ {
		value += ch
	}
	return value
}
