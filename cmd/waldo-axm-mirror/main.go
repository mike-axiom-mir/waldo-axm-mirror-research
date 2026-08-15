package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "lens-corpus":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = lensCorpusFile(os.Args[2], os.Args[3])
	case "witness-run":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = witnessRunFile(os.Args[2], os.Args[3], os.Args[4])
	case "profile-contract":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = profileContractFile(os.Args[2], os.Args[3], os.Args[4])
	case "anchor":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = anchorFile(os.Args[2], os.Args[3])
	case "lock":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = lockFile(os.Args[2], os.Args[3], os.Args[4])
	case "contamination":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = contaminationFile(os.Args[2], os.Args[3])
	case "context":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = contextFile(os.Args[2], os.Args[3])
	case "gate-claims":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = gateClaimsFile(os.Args[2], os.Args[3], os.Args[4])
	case "seal-evaluation":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = sealEvaluationFile(os.Args[2], os.Args[3])
	case "seal-gated":
		if len(os.Args) != 10 {
			usage()
			os.Exit(2)
		}
		err = sealGatedFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6], os.Args[7], os.Args[8], os.Args[9])
	case "seal":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = sealFile(os.Args[2], os.Args[3])
	case "seal-anchored":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = sealAnchoredFile(os.Args[2], os.Args[3], os.Args[4])
	case "seal-witnessed":
		if len(os.Args) != 6 {
			usage()
			os.Exit(2)
		}
		err = sealWitnessedFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "verify":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		err = verifyFile(os.Args[2])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror lens-corpus <corpus-bom.json> <lens.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror witness-run <run-bom.json> <run.json> <witness.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror profile-contract <run-bom.json> <run-witness.json> <profile.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror anchor <waldo-bom.json> <anchor.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror lock <expected-anchor.json> <observed-bom.json> <receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror contamination <comparison.json> <report.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror context <context-request.json> <context-packet.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror gate-claims <context-packet.json> <claim-submission.json> <assessment.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-evaluation <protocol-draft.json> <protocol-seal.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-gated <anchor.json> <run-witness.json> <profile.json> <context.json> <claims.json> <protocol.json> <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-anchored <anchor.json> <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-witnessed <anchor.json> <run-witness.json> <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror verify <sealed.json>")
}

func lensCorpusFile(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}
	lens, err := axmmirror.LensCorpusBOM(data)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, lens); err != nil {
		return err
	}
	fmt.Println(lens.State, lens.BOMSHA256)
	return nil
}

func witnessRunFile(runBOMPath, runPath, outputPath string) error {
	runBOMData, err := os.ReadFile(runBOMPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", runBOMPath, err)
	}
	runData, err := os.ReadFile(runPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", runPath, err)
	}
	witness, err := axmmirror.WitnessTrainingRun(runBOMData, runData)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, witness); err != nil {
		return err
	}
	fmt.Println(witness.State, witness.RunBOMSHA256)
	if witness.State != axmmirror.RunWitnessStateReady {
		return errors.New("training run witness is HOLD; inspect the written receipt")
	}
	return nil
}

func profileContractFile(runBOMPath, witnessPath, outputPath string) error {
	runBOMData, err := os.ReadFile(runBOMPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", runBOMPath, err)
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		return err
	}
	receipt, err := axmmirror.LensTrainingProfile(runBOMData, witness)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.CanonicalProfile)
	if receipt.State != axmmirror.ProfileStateWitnessed && receipt.State != axmmirror.ProfileStateLegacyWitnessed {
		return fmt.Errorf("training profile contract is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func anchorFile(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}
	anchor, err := axmmirror.AnchorWALDOBOM(data)
	if err != nil {
		return err
	}
	if err := anchor.Validate(); err != nil {
		return err
	}
	if err := writeJSON(outputPath, anchor); err != nil {
		return err
	}
	fmt.Println(anchor.State, anchor.AnsweringIdentitySHA256)
	if anchor.State != axmmirror.AnchorStateAnchored {
		return errors.New("origin anchor is HOLD; inspect the written receipt")
	}
	return nil
}

func lockFile(expectedPath, observedPath, outputPath string) error {
	var expected axmmirror.OriginAnchor
	if err := readStrictJSON(expectedPath, &expected); err != nil {
		return err
	}
	data, err := os.ReadFile(observedPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", observedPath, err)
	}
	observed, err := axmmirror.AnchorWALDOBOM(data)
	if err != nil {
		return err
	}
	receipt, err := axmmirror.CompareIdentity(expected, observed)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.Observed.AnsweringIdentitySHA256)
	if receipt.State != axmmirror.IdentityStateLocked {
		return fmt.Errorf("release identity lock is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func contaminationFile(inputPath, outputPath string) error {
	var comparison axmmirror.EvaluationComparison
	if err := readStrictJSON(inputPath, &comparison); err != nil {
		return err
	}
	report, err := axmmirror.CheckContamination(comparison)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, report); err != nil {
		return err
	}
	fmt.Println(report.State)
	if report.State != axmmirror.ContaminationStateClear {
		return fmt.Errorf("evaluation contamination guard is %s; inspect the written report", report.State)
	}
	return nil
}

func contextFile(inputPath, outputPath string) error {
	var request axmmirror.ProvenanceContextRequest
	if err := readStrictJSON(inputPath, &request); err != nil {
		return err
	}
	packet, err := axmmirror.BuildProvenanceContext(request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, packet); err != nil {
		return err
	}
	fmt.Println(packet.State, packet.PacketSHA256)
	if packet.State != axmmirror.ContextStateReady {
		return fmt.Errorf("provenance context is %s; inspect the written packet", packet.State)
	}
	return nil
}

func gateClaimsFile(contextPath, submissionPath, outputPath string) error {
	var packet axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &packet); err != nil {
		return err
	}
	var submission axmmirror.SourceClaimSubmission
	if err := readStrictJSON(submissionPath, &submission); err != nil {
		return err
	}
	assessment, err := axmmirror.AssessSourceClaims(submission, packet)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, assessment); err != nil {
		return err
	}
	fmt.Println(assessment.State, assessment.ReceiptSHA256)
	if assessment.State != axmmirror.ClaimGateStateConfirmed {
		return fmt.Errorf("source-claim gate is %s; inspect the written assessment", assessment.State)
	}
	return nil
}

func sealEvaluationFile(inputPath, outputPath string) error {
	var draft axmmirror.EvaluationProtocolDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	seal, err := axmmirror.SealEvaluationProtocol(draft)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, seal); err != nil {
		return err
	}
	fmt.Println(seal.State, seal.SealSHA256)
	if seal.State != axmmirror.ProtocolStateReady {
		return fmt.Errorf("evaluation protocol is %s; inspect the written seal", seal.State)
	}
	return nil
}

func sealGatedFile(anchorPath, witnessPath, profilePath, contextPath, claimsPath, protocolPath, draftPath, outputPath string) error {
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		return err
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		return err
	}
	var profile axmmirror.TrainingProfileContract
	if err := readStrictJSON(profilePath, &profile); err != nil {
		return err
	}
	var context axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &context); err != nil {
		return err
	}
	var claims axmmirror.SourceClaimAssessment
	if err := readStrictJSON(claimsPath, &claims); err != nil {
		return err
	}
	var protocol axmmirror.EvaluationProtocolSeal
	if err := readStrictJSON(protocolPath, &protocol); err != nil {
		return err
	}
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(draftPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.SealGated(draft, anchor, witness, profile, context, claims, protocol)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func sealFile(inputPath, outputPath string) error {
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.Seal(draft)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func sealAnchoredFile(anchorPath, inputPath, outputPath string) error {
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		return err
	}
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.SealAnchored(draft, anchor)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func sealWitnessedFile(anchorPath, witnessPath, inputPath, outputPath string) error {
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		return err
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		return err
	}
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.SealWitnessed(draft, anchor, witness)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func verifyFile(path string) error {
	var sealed axmmirror.SealedBehaviorEvidence
	if err := readStrictJSON(path, &sealed); err != nil {
		return err
	}
	if err := sealed.Verify(); err != nil {
		return err
	}
	fmt.Println("OK", sealed.SHA256)
	return nil
}

func readStrictJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := axmmirror.DecodeStrictJSON(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create output directory for %s: %w", path, err)
	}
	temporary, err := os.CreateTemp(directory, ".waldo-axm-mirror-*")
	if err != nil {
		return fmt.Errorf("create temporary output for %s: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary output for %s: %w", path, err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("set output mode for %s: %w", path, err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync output for %s: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close output for %s: %w", path, err)
	}
	// Linking the synced temporary file is an atomic, no-replace commit. It
	// prevents a second run from silently overwriting an earlier receipt.
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("output %s already exists", path)
		}
		return fmt.Errorf("commit output %s: %w", path, err)
	}
	return nil
}
