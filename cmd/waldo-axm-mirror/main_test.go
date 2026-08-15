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

func TestGatedCloneFoundationCLIFlow(t *testing.T) {
	examples := filepath.Join("..", "..", "examples", "axm-mirror")
	directory := t.TempDir()
	runBOM := filepath.Join(examples, "run-bom.json")
	run := filepath.Join(examples, "run.json")
	witnessPath := filepath.Join(directory, "run-witness.json")
	if err := witnessRunFile(runBOM, run, witnessPath); err != nil {
		t.Fatalf("witnessRunFile() error = %v", err)
	}
	profilePath := filepath.Join(directory, "profile-contract.json")
	if err := profileContractFile(runBOM, witnessPath, profilePath); err != nil {
		t.Fatalf("profileContractFile() error = %v", err)
	}
	anchorPath := filepath.Join(directory, "anchor.json")
	if err := anchorFile(filepath.Join(examples, "model-release-bom.json"), anchorPath); err != nil {
		t.Fatalf("anchorFile() error = %v", err)
	}
	contaminationPath := filepath.Join(directory, "contamination.json")
	if err := contaminationFile(filepath.Join(examples, "evaluation-comparison.json"), contaminationPath); err != nil {
		t.Fatalf("contaminationFile() error = %v", err)
	}

	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		t.Fatal(err)
	}
	var profile axmmirror.TrainingProfileContract
	if err := readStrictJSON(profilePath, &profile); err != nil {
		t.Fatal(err)
	}
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		t.Fatal(err)
	}
	var contamination axmmirror.ContaminationReport
	if err := readStrictJSON(contaminationPath, &contamination); err != nil {
		t.Fatal(err)
	}
	lock, err := axmmirror.CompareIdentity(anchor, anchor)
	if err != nil {
		t.Fatal(err)
	}
	request := axmmirror.ProvenanceContextRequest{
		Schema: axmmirror.ProvenanceContextRequestSchema, RequestID: "cli-context-0001",
		MaxFactBytes: axmmirror.MaxProvenanceFactBytes,
		Fields: []string{
			"answering.identity_sha256", "corpus.paths", "corpus.licenses", "training.profile.canonical",
			"evaluation.contamination_state", "evaluation.comparison_sha256",
		},
		Anchor: anchor, Corpus: witness.Corpus, RunWitness: &witness, ProfileContract: &profile,
		IdentityLock: &lock, Contamination: &contamination,
	}
	requestPath := writeValueTemp(t, request)
	contextPath := filepath.Join(directory, "context.json")
	if err := contextFile(requestPath, contextPath); err != nil {
		t.Fatalf("contextFile() error = %v", err)
	}
	var context axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &context); err != nil {
		t.Fatal(err)
	}

	submission := axmmirror.SourceClaimSubmission{
		Schema: axmmirror.SourceClaimSubmissionSchema, SubmissionID: "cli-claims-0001",
		OutputSHA256: repeat("a"), ContextPacketSHA256: context.PacketSHA256,
		Claims: []axmmirror.SourceClaim{
			{ID: "identity", Kind: axmmirror.ClaimKindAnsweringIdentity, Value: anchor.AnsweringIdentitySHA256},
			{ID: "license", Kind: axmmirror.ClaimKindLicenseAssertion, Value: "CC0-1.0"},
			{ID: "path", Kind: axmmirror.ClaimKindCorpusPathMembership, Value: "research/example-corpus"},
			{ID: "profile", Kind: axmmirror.ClaimKindTrainingProfile, Value: "causal-pretrain-shuffled"},
			{ID: "evaluation", Kind: axmmirror.ClaimKindEvaluationIndependent, Value: axmmirror.ClaimValueEvaluationClear},
		},
	}
	submissionPath := writeValueTemp(t, submission)
	assessmentPath := filepath.Join(directory, "assessment.json")
	if err := gateClaimsFile(contextPath, submissionPath, assessmentPath); err != nil {
		t.Fatalf("gateClaimsFile() error = %v", err)
	}
	var assessment axmmirror.SourceClaimAssessment
	if err := readStrictJSON(assessmentPath, &assessment); err != nil {
		t.Fatal(err)
	}
	if assessment.State != axmmirror.ClaimGateStateConfirmed {
		t.Fatalf("assessment state = %q", assessment.State)
	}

	draft := axmmirror.EvaluationProtocolDraft{
		Schema: axmmirror.EvaluationProtocolDraftSchema, ProtocolID: "cli-protocol-0001",
		Pack: axmmirror.EvaluationPackIdentity{
			SHA256: repeat("1"), DocumentSHA256: repeat("2"), CaseCount: 2,
			CaseOrderSHA256: repeat("3"), AuthorshipState: "outside-authored",
			AuthorshipEvidenceSHA256: repeat("6"), AnswerKeySHA256: repeat("4"),
		},
		TargetAnsweringIdentitySHA256: anchor.AnsweringIdentitySHA256,
		Anchor:                        anchor, Context: context, Contamination: contamination,
		RequestSetSHA256: repeat("5"), AllowedMetrics: []string{"exact-match", "refusal-state"},
		ComparisonDimensions: []string{"output", "verification"}, AnswerKeyVisibility: axmmirror.AnswerKeyWithheld,
		Limits:          axmmirror.EvaluationExecutionLimits{MaxCases: 2, MaxOutputBytes: 4096, TimeoutMillis: 30000},
		PermissionState: "allowed", RequestedAuthority: axmmirror.Authority{},
	}
	draftPath := writeValueTemp(t, draft)
	sealPath := filepath.Join(directory, "protocol-seal.json")
	if err := sealEvaluationFile(draftPath, sealPath); err != nil {
		t.Fatalf("sealEvaluationFile() error = %v", err)
	}
	var seal axmmirror.EvaluationProtocolSeal
	if err := readStrictJSON(sealPath, &seal); err != nil {
		t.Fatal(err)
	}
	if seal.State != axmmirror.ProtocolStateReady {
		t.Fatalf("protocol state = %q", seal.State)
	}
	behaviorDraft := axmmirror.BehaviorEvidenceDraft{
		Schema: axmmirror.BehaviorEvidenceSchema, ExperimentID: seal.ProtocolID,
		WALDO: axmmirror.WALDOLineage{
			CorpusBOMSHA256:  witness.Corpus.BOMSHA256,
			RunBOMSHA256:     witness.RunBOMSHA256,
			ReleaseBOMSHA256: anchor.BOMSHA256,
		},
		ContextSHA256: context.PacketSHA256, RequestSHA256: seal.RequestSetSHA256,
		OutputSHA256: assessment.OutputSHA256, PermissionState: "allowed", OutcomeState: "pass",
		Authority: axmmirror.Authority{},
	}
	behaviorDraftPath := writeValueTemp(t, behaviorDraft)
	sealedPath := filepath.Join(directory, "gated-behavior.json")
	if err := sealGatedFile(anchorPath, witnessPath, profilePath, contextPath, assessmentPath, sealPath, behaviorDraftPath, sealedPath); err != nil {
		t.Fatalf("sealGatedFile() error = %v", err)
	}
	var sealed axmmirror.SealedBehaviorEvidence
	if err := readStrictJSON(sealedPath, &sealed); err != nil {
		t.Fatal(err)
	}
	if err := sealed.Verify(); err != nil {
		t.Fatalf("gated behavior verification: %v", err)
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

func writeValueTemp(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return writeTemp(t, string(data))
}

func repeat(ch string) string {
	value := ""
	for i := 0; i < 64; i++ {
		value += ch
	}
	return value
}
