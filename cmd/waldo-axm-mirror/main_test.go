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

func TestInnerAssetCLIFlow(t *testing.T) {
	input := filepath.Join("..", "..", "examples", "axm-mirror", "inner-asset-recipe.json")
	output := filepath.Join(t.TempDir(), "witness-orb.axmasset")
	if err := forgeInnerAssetFile(input, output); err != nil {
		t.Fatalf("forgeInnerAssetFile() error = %v", err)
	}
	if err := verifyInnerAssetFile(output); err != nil {
		t.Fatalf("verifyInnerAssetFile() error = %v", err)
	}
	if err := forgeInnerAssetFile(input, output); err == nil {
		t.Fatal("forgeInnerAssetFile() overwrote an existing candidate")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := axmmirror.VerifyInnerAssetBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.State != axmmirror.InnerAssetStateReady || !candidate.CandidateOnly || candidate.VisualStatus != axmmirror.InnerAssetVisualPending {
		t.Fatalf("inner asset candidate = %+v", candidate)
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

	// Add the situated layer without changing the earlier provenance packet or
	// gated behavior schemas.
	subject := axmmirror.EvidenceSubject{ID: "cli-case-0001", Kind: "evaluation-case", SHA256: repeat("c")}
	sensoryDraft := axmmirror.SensoryEvidenceDraft{
		Schema: axmmirror.SensoryEvidenceDraftSchema, ObservationID: "cli-touch-0001", ClaimID: "cli-claim-0001",
		SenseID: "touch-environment-probe", Capability: "sense.environment.touch/v1", Subject: subject,
		SeatID: "workshop-seat", BackendID: "cli-fixture", ObservedAt: "2026-08-15T10:00:00Z",
		SealedAt: "2026-08-15T10:00:00.100Z", AssessedAt: "2026-08-15T10:00:00.200Z", TTLMillis: 60_000,
		ObservationState: axmmirror.SensoryObservationObserved, Verdict: "PASS",
		TypedObservationSHA256: repeat("d"), SpecificReceiptSchema: "axm.touch-fixture/v1",
		SpecificReceiptSHA256: repeat("e"), CleanupComplete: true,
	}
	sensoryDraftPath := writeValueTemp(t, sensoryDraft)
	sensoryPath := filepath.Join(directory, "sensory.json")
	if err := intakeSensoryFile(sensoryDraftPath, sensoryPath); err != nil {
		t.Fatalf("intakeSensoryFile() error = %v", err)
	}
	var sensory axmmirror.SensoryEvidenceReceipt
	if err := readStrictJSON(sensoryPath, &sensory); err != nil {
		t.Fatal(err)
	}

	skillItems := []axmmirror.SkillInventoryItem{
		{ID: "mirror-organ-library-115", Version: "03f3d0cf", UseClass: axmmirror.SkillUseKnowledge, SourceSHA256: repeat("1"), Status: "TEST", ExecutionStatus: "INERT", ProofStatus: "UNTESTED", CompatibilityStatus: "UNVERIFIED", MemberCount: 115},
		{ID: "sensorium-portable-skills", Version: "1.4.0", UseClass: axmmirror.SkillUseInstruction, SourceSHA256: repeat("2"), InstructionSHA256: repeat("3"), Status: "TEST", ExecutionStatus: "INERT", ProofStatus: "CONTRACT_PASS", CompatibilityStatus: "UNVERIFIED", MemberCount: 13},
	}
	skillRequest := axmmirror.SkillContinuityRequest{
		Schema: axmmirror.SkillContinuityRequestSchema, AssessmentID: "cli-skills-0001",
		TargetAnsweringIdentitySHA256: anchor.AnsweringIdentitySHA256,
		Requirements: []axmmirror.SkillRequirement{
			{ID: "mirror-organ-library-115", UseClass: axmmirror.SkillUseKnowledge},
			{ID: "sensorium-portable-skills", UseClass: axmmirror.SkillUseInstruction},
		},
		Current: axmmirror.SkillInventorySet{SetID: "cli-current", CapturedAt: "2026-08-15T10:00:00Z", Complete: true, Items: skillItems},
		Backup:  axmmirror.SkillInventorySet{SetID: "cli-backup", CapturedAt: "2026-08-15T09:00:00Z", Complete: true, Items: append([]axmmirror.SkillInventoryItem(nil), skillItems...)},
	}
	skillRequestPath := writeValueTemp(t, skillRequest)
	skillPath := filepath.Join(directory, "skills.json")
	if err := assessSkillsFile(skillRequestPath, skillPath); err != nil {
		t.Fatalf("assessSkillsFile() error = %v", err)
	}
	var skills axmmirror.SkillContinuityReceipt
	if err := readStrictJSON(skillPath, &skills); err != nil {
		t.Fatal(err)
	}

	discoveryRequest := axmmirror.DiscoveryStanceRequest{
		Schema: axmmirror.DiscoveryStanceRequestSchema, RequestID: "cli-discovery-0001", Subject: subject,
		AISeams: []axmmirror.DiscoverySeam{{
			ID: "freshness-visible", Stance: "evidence", Severity: "medium", Status: "OPEN",
			Statement: "The task depends on a bounded sensory receipt.", EvidenceRefs: []string{sensory.ReceiptSHA256},
			DisconfirmingCheck: "Re-run the exact observation after its TTL and compare typed receipts.", Blocks: []string{"unbounded-current-state-claim"},
		}},
		Human: &axmmirror.HumanDiscoveryInput{ExplicitInvocation: true, Entries: []axmmirror.HumanDiscoveryEntry{{
			ID: "human-use-0001", Stage: "blindSpots", Text: "Explain stale evidence in plain language.", ClaimLabel: "VALUE",
		}}},
	}
	discoveryRequestPath := writeValueTemp(t, discoveryRequest)
	discoveryPath := filepath.Join(directory, "discovery.json")
	if err := discoveryFile(discoveryRequestPath, discoveryPath); err != nil {
		t.Fatalf("discoveryFile() error = %v", err)
	}
	var discovery axmmirror.DiscoveryStancePacket
	if err := readStrictJSON(discoveryPath, &discovery); err != nil {
		t.Fatal(err)
	}

	situatedRequest := axmmirror.SituatedContextRequest{
		Schema: axmmirror.SituatedContextRequestSchema, EnvelopeID: "cli-situated-0001",
		TargetAnsweringIdentitySHA256: anchor.AnsweringIdentitySHA256, Subject: subject,
		ContextPacketSHA256: context.PacketSHA256, RequiredSenses: []string{sensory.SenseID},
		SensoryEvidence: []axmmirror.SensoryEvidenceReceipt{sensory}, SkillContinuity: skills, DiscoveryStance: discovery,
	}
	situatedRequestPath := writeValueTemp(t, situatedRequest)
	situatedPath := filepath.Join(directory, "situated.json")
	if err := situatedContextFile(contextPath, situatedRequestPath, situatedPath); err != nil {
		t.Fatalf("situatedContextFile() error = %v", err)
	}
	var situated axmmirror.SituatedContextEnvelope
	if err := readStrictJSON(situatedPath, &situated); err != nil {
		t.Fatal(err)
	}
	if situated.State != axmmirror.SituatedContextReady {
		t.Fatalf("situated state = %q", situated.State)
	}

	situatedSealedPath := filepath.Join(directory, "situated-behavior.json")
	if err := sealSituatedFile(anchorPath, witnessPath, profilePath, contextPath, assessmentPath, sealPath, situatedPath, behaviorDraftPath, situatedSealedPath); err != nil {
		t.Fatalf("sealSituatedFile() error = %v", err)
	}
	var situatedSealed axmmirror.SealedBehaviorEvidence
	if err := readStrictJSON(situatedSealedPath, &situatedSealed); err != nil {
		t.Fatal(err)
	}
	if err := situatedSealed.Verify(); err != nil {
		t.Fatalf("situated behavior verification: %v", err)
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
