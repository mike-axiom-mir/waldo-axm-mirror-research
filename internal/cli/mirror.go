package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/inference"
	"github.com/openwaldo/waldo/internal/model"
)

const mirrorLearningModeCandidate = "candidate"
const mirrorLearningModeApproved = "approved"

var mirrorReasonInput io.Reader = os.Stdin

var openMirrorNeuralEscalator = func(ctx context.Context, name string, options inference.Options) (axmmirror.MirrorNeuralEscalator, func() error, error) {
	root, err := configuredModelRoot()
	if err != nil {
		return nil, nil, err
	}
	inspection, err := model.Inspect(root, name)
	if err != nil {
		return nil, nil, err
	}
	if len(inspection.Model.Runs) == 0 && inspection.Origin == nil {
		return nil, nil, fmt.Errorf("model %q is untrained", name)
	}
	// Reverse Mirror escalation intentionally opens the existing raw local
	// session. The Mirror layer owns candidate status and authority here; routing
	// the request back through the v0.35 response wrapper would create a loop.
	opened, err := inference.Open(ctx, inspection)
	if err != nil {
		return nil, nil, err
	}
	escalator := &localMirrorNeuralEscalator{
		session:     opened.Session,
		description: opened.Description,
		interaction: inspection.Model.Interaction,
		options:     options,
	}
	return escalator, opened.Session.Close, nil
}

type localMirrorNeuralEscalator struct {
	session     inference.Session
	description inference.Description
	interaction model.Interaction
	options     inference.Options
}

func (escalator *localMirrorNeuralEscalator) Escalate(ctx context.Context, prompt string) (axmmirror.MirrorNeuralCandidate, error) {
	options := escalator.options
	options.Stop = escalator.interaction.Stops()
	renderedPrompt := escalator.interaction.Prompt("", prompt)
	result, err := escalator.session.Generate(ctx, renderedPrompt, options, nil)
	if err != nil {
		return axmmirror.MirrorNeuralCandidate{}, err
	}
	return axmmirror.MirrorNeuralCandidate{
		Text:         escalator.interaction.TrimResponse(result.Text),
		Model:        escalator.description.Model,
		Backend:      escalator.description.Backend,
		SourceType:   escalator.description.SourceType,
		SourceID:     escalator.description.SourceID,
		RunID:        escalator.description.RunID,
		Tokens:       result.Tokens,
		FinishReason: result.FinishReason,
		DurationMS:   result.DurationMS,
	}, nil
}

func runMirrorReason(commandContext Context, args []string, stdout, _ io.Writer) error {
	request, err := loadMirrorReasonRequest(args[0])
	if err != nil {
		return err
	}
	options := inference.Options{
		MaxTokens:   intOption(commandContext, "max-tokens"),
		Temperature: float64Option(commandContext, "temperature"),
		TopP:        float64Option(commandContext, "top-p"),
	}
	if optionChanged(commandContext, "seed") {
		seed := uint64Option(commandContext, "seed")
		options.Seed = &seed
	}
	if err := options.Validate(); err != nil {
		return err
	}
	timeoutSeconds := intOption(commandContext, "timeout-seconds")
	if timeoutSeconds < 1 || timeoutSeconds > 900 {
		return errors.New("--timeout-seconds must be in 1..900")
	}
	neuralOptIn := boolOption(commandContext, "neural")
	modelName := strings.TrimSpace(stringOption(commandContext, "model"))
	var escalator axmmirror.MirrorNeuralEscalator
	closeEscalator := func() error { return nil }
	if request.NeedsNeuralEscalation() && neuralOptIn {
		if modelName == "" {
			return errors.New("--model is required when unresolved Mirror reasoning opts into --neural")
		}
		escalator, closeEscalator, err = openMirrorNeuralEscalator(commandContext.Execution, modelName, options)
		if err != nil {
			return fmt.Errorf("open local WALDO model %q for Mirror escalation: %w", modelName, err)
		}
	}
	ctx, cancel := context.WithTimeout(commandContext.Execution, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	receipt, runErr := axmmirror.RunMirrorNeuralEscalation(ctx, request, neuralOptIn, escalator)
	closeErr := closeEscalator()

	learnPath := strings.TrimSpace(stringOption(commandContext, "learn-to"))
	learningMode := strings.ToLower(strings.TrimSpace(stringOption(commandContext, "learning-mode")))
	if learningMode == "" {
		learningMode = mirrorLearningModeCandidate
	}
	if learnPath == "" && optionChanged(commandContext, "learning-mode") {
		return errors.New("--learning-mode requires --learn-to")
	}
	if learnPath != "" && runErr == nil {
		disposition, dispositionErr := mirrorLearningDisposition(learningMode)
		if dispositionErr != nil {
			return dispositionErr
		}
		record, recordErr := axmmirror.BuildMirrorChatLearningRecord(receipt, request.Prompt, disposition)
		if recordErr != nil {
			return recordErr
		}
		line, recordErr := record.JSONLine()
		if recordErr != nil {
			return recordErr
		}
		if recordErr := appendPrivateLine(learnPath, line); recordErr != nil {
			return fmt.Errorf("append visible Mirror chat learning candidate: %w", recordErr)
		}
		receipt.LearningCandidateRecorded = true
		receipt.LearningRecordSHA256 = record.LearningRecordSHA256
		receipt.LearningLedgerMutation = true
	}

	tracePath := strings.TrimSpace(stringOption(commandContext, "trace"))
	if tracePath != "" && receipt.Schema != "" {
		line, traceErr := mirrorTraceLine(receipt)
		if traceErr != nil {
			return traceErr
		}
		if traceErr := appendPrivateLine(tracePath, line); traceErr != nil {
			return fmt.Errorf("append Mirror escalation trace: %w", traceErr)
		}
	}
	if receipt.Schema != "" {
		if err := writeMirrorReceipt(commandContext, stdout, receipt); err != nil {
			return err
		}
	}
	return errors.Join(runErr, closeErr)
}

func loadMirrorReasonRequest(path string) (axmmirror.MirrorNeuralEscalationRequest, error) {
	if path == "-" {
		return axmmirror.LoadMirrorNeuralEscalationRequest(mirrorReasonInput)
	}
	file, err := os.Open(path)
	if err != nil {
		return axmmirror.MirrorNeuralEscalationRequest{}, fmt.Errorf("open Mirror reasoning request %s: %w", path, err)
	}
	defer file.Close()
	request, err := axmmirror.LoadMirrorNeuralEscalationRequest(file)
	if err != nil {
		return request, fmt.Errorf("load Mirror reasoning request %s: %w", path, err)
	}
	return request, nil
}

func mirrorLearningDisposition(mode string) (string, error) {
	switch mode {
	case mirrorLearningModeCandidate:
		return axmmirror.MirrorLearningCaptureCandidate, nil
	case mirrorLearningModeApproved:
		return axmmirror.MirrorLearningApproveTraining, nil
	default:
		return "", errors.New("--learning-mode must be candidate or approved")
	}
}

func writeMirrorReceipt(commandContext Context, output io.Writer, receipt axmmirror.MirrorNeuralEscalationReceipt) error {
	if commandContext.JSON {
		return writeJSON(output, receipt)
	}
	fmt.Fprintf(output, "Mirror: %s\n", receipt.Status)
	fmt.Fprintln(output, "Authority: NONE")
	switch receipt.Status {
	case axmmirror.MirrorStatusDeterministicResolved:
		fmt.Fprintln(output, "Source: deterministic Mirror")
		fmt.Fprintln(output)
		fmt.Fprintln(output, receipt.DeterministicResponse)
	case axmmirror.MirrorStatusNeuralOptInRequired:
		fmt.Fprintln(output, "Neural escalation was not opted in; unresolved reasoning remains on HOLD.")
	case axmmirror.MirrorStatusNeuralCandidate:
		fmt.Fprintf(output, "Source: local WALDO neural candidate (%s via %s)\n", receipt.NeuralCandidate.Model, receipt.NeuralCandidate.Backend)
		fmt.Fprintln(output, "Candidate only: no permission, execution, identity mutation, training, promotion, or CANON.")
		fmt.Fprintln(output)
		fmt.Fprintln(output, strings.ToValidUTF8(receipt.NeuralCandidate.Text, "�"))
	case axmmirror.MirrorStatusNeuralFailed:
		fmt.Fprintln(output, "Local neural escalation failed closed; unresolved reasoning remains on HOLD.")
	}
	if receipt.LearningCandidateRecorded {
		fmt.Fprintf(output, "Learning record: %s\n", receipt.LearningRecordSHA256)
	}
	return nil
}

type mirrorEscalationTrace struct {
	Schema                    string    `json:"schema"`
	Status                    string    `json:"status"`
	DeterministicPrimary      bool      `json:"deterministicPrimary"`
	NeuralOptIn               bool      `json:"neuralOptIn"`
	NeuralCalled              bool      `json:"neuralCalled"`
	EscalationReason          string    `json:"escalationReason"`
	Consequence               string    `json:"consequence"`
	RequestSHA256             string    `json:"requestSha256"`
	PromptSHA256              string    `json:"promptSha256"`
	NeuralPromptSHA256        string    `json:"neuralPromptSha256,omitempty"`
	GroundingSHA256           string    `json:"groundingSha256"`
	IdentitySHA256            string    `json:"identitySha256,omitempty"`
	NeuralCandidateSHA256     string    `json:"neuralCandidateSha256,omitempty"`
	NeuralErrorSHA256         string    `json:"neuralErrorSha256,omitempty"`
	LearningCandidateRecorded bool      `json:"learningCandidateRecorded"`
	LearningRecordSHA256      string    `json:"learningRecordSha256,omitempty"`
	LearningLedgerMutation    bool      `json:"learningLedgerMutation"`
	ModelMemoryMutation       bool      `json:"modelMemoryMutation"`
	TrainingMutation          bool      `json:"trainingMutation"`
	IdentityMutation          bool      `json:"identityMutation"`
	RawTextIncluded           bool      `json:"rawTextIncluded"`
	Authority                 string    `json:"authority"`
	Timestamp                 time.Time `json:"timestamp"`
}

func mirrorTraceLine(receipt axmmirror.MirrorNeuralEscalationReceipt) ([]byte, error) {
	trace := mirrorEscalationTrace{
		Schema:                    "axm.waldo.mirror-neural-escalation-trace/v0.37",
		Status:                    receipt.Status,
		DeterministicPrimary:      receipt.DeterministicPrimary,
		NeuralOptIn:               receipt.NeuralOptIn,
		NeuralCalled:              receipt.NeuralCalled,
		EscalationReason:          receipt.EscalationReason,
		Consequence:               receipt.Consequence,
		RequestSHA256:             receipt.RequestSHA256,
		PromptSHA256:              receipt.PromptSHA256,
		NeuralPromptSHA256:        receipt.NeuralPromptSHA256,
		GroundingSHA256:           receipt.GroundingSHA256,
		IdentitySHA256:            receipt.IdentitySHA256,
		NeuralCandidateSHA256:     receipt.NeuralCandidateSHA256,
		NeuralErrorSHA256:         receipt.NeuralErrorSHA256,
		LearningCandidateRecorded: receipt.LearningCandidateRecorded,
		LearningRecordSHA256:      receipt.LearningRecordSHA256,
		LearningLedgerMutation:    receipt.LearningLedgerMutation,
		ModelMemoryMutation:       receipt.ModelMemoryMutation,
		TrainingMutation:          receipt.TrainingMutation,
		IdentityMutation:          receipt.IdentityMutation,
		RawTextIncluded:           false,
		Authority:                 "NONE",
		Timestamp:                 receipt.Timestamp,
	}
	payload, err := json.Marshal(trace)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func appendPrivateLine(path string, line []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(line); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
