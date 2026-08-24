package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/inference"
)

type cliMirrorEscalator struct {
	candidate axmmirror.MirrorNeuralCandidate
	calls     int
	prompt    string
}

func (stub *cliMirrorEscalator) Escalate(_ context.Context, prompt string) (axmmirror.MirrorNeuralCandidate, error) {
	stub.calls++
	stub.prompt = prompt
	return stub.candidate, nil
}

func TestMirrorReasonStableDoesNotOpenNeuralModel(t *testing.T) {
	request := cliMirrorRequest(axmmirror.GroundingStable, axmmirror.MirrorEscalationNone)
	request.DeterministicResponse = "known deterministic result"
	path := writeMirrorRequest(t, request)
	previous := openMirrorNeuralEscalator
	openMirrorNeuralEscalator = func(context.Context, string, inference.Options) (axmmirror.MirrorNeuralEscalator, func() error, error) {
		t.Fatal("stable Mirror reasoning opened a neural model")
		return nil, nil, nil
	}
	t.Cleanup(func() { openMirrorNeuralEscalator = previous })
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{path, "--neural"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), axmmirror.MirrorStatusDeterministicResolved) || !strings.Contains(output.String(), "known deterministic result") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestMirrorReasonUnresolvedWithoutOptInHolds(t *testing.T) {
	path := writeMirrorRequest(t, cliMirrorRequest(axmmirror.GroundingUncertain, axmmirror.MirrorEscalationNovelty))
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var receipt axmmirror.MirrorNeuralEscalationReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != axmmirror.MirrorStatusNeuralOptInRequired || receipt.NeuralCalled {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestMirrorReasonNeuralLearningAndTraceAreVisible(t *testing.T) {
	request := cliMirrorRequest(axmmirror.GroundingUncertain, axmmirror.MirrorEscalationAmbiguity)
	identity, err := axmmirror.NewMirrorIdentityCapsule("axiom-mir", "roots-v1", []string{"truth before story", "no hidden control"})
	if err != nil {
		t.Fatal(err)
	}
	request.Identity = &identity
	path := writeMirrorRequest(t, request)
	directory := t.TempDir()
	learningPath := filepath.Join(directory, "learning.jsonl")
	tracePath := filepath.Join(directory, "trace.jsonl")
	stub := &cliMirrorEscalator{candidate: axmmirror.MirrorNeuralCandidate{Text: "neural possibility", Model: "fixture", Backend: "pytorch", Tokens: 2, FinishReason: "max_tokens"}}
	closed := 0
	previous := openMirrorNeuralEscalator
	openMirrorNeuralEscalator = func(_ context.Context, name string, options inference.Options) (axmmirror.MirrorNeuralEscalator, func() error, error) {
		if name != "fixture" || options.MaxTokens != 8 || options.Temperature != 0 || options.TopP != 1 || options.Seed == nil || *options.Seed != 37 {
			t.Fatalf("name/options = %q %+v", name, options)
		}
		return stub, func() error { closed++; return nil }, nil
	}
	t.Cleanup(func() { openMirrorNeuralEscalator = previous })
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{
		path, "--neural", "--model", "fixture", "--max-tokens", "8", "--temperature", "0", "--top-p", "1", "--seed", "37",
		"--learn-to", learningPath, "--learning-mode", "approved", "--trace", tracePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	commandContext.JSON = true
	var output bytes.Buffer
	if err := runMirrorReason(commandContext, args, &output, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var receipt axmmirror.MirrorNeuralEscalationReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != axmmirror.MirrorStatusNeuralCandidate || !receipt.LearningCandidateRecorded || !receipt.LearningLedgerMutation || receipt.ModelMemoryMutation || receipt.TrainingMutation || receipt.IdentityMutation || receipt.LearningRecordSHA256 == "" || stub.calls != 1 || closed != 1 {
		t.Fatalf("receipt = %+v, calls = %d, closed = %d", receipt, stub.calls, closed)
	}
	if !strings.Contains(stub.prompt, "truth before story") || !strings.Contains(stub.prompt, request.Prompt) {
		t.Fatalf("neural prompt = %q", stub.prompt)
	}
	learning, err := os.ReadFile(learningPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(learning, []byte(`"prompt":"What changed?"`)) || !bytes.Contains(learning, []byte(`"response":"neural possibility"`)) || !bytes.Contains(learning, []byte(`"trainingReady":true`)) || !bytes.Contains(learning, []byte(`"text":"User: What changed?\n\nAssistant: neural possibility"`)) {
		t.Fatalf("learning record = %s", learning)
	}
	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(trace, []byte(request.Prompt)) || bytes.Contains(trace, []byte("neural possibility")) || !bytes.Contains(trace, []byte(`"rawTextIncluded":false`)) {
		t.Fatalf("trace = %s", trace)
	}
	for _, name := range []string{learningPath, tracePath} {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != fs.FileMode(0o600) {
			t.Fatalf("%s mode = %o", name, info.Mode().Perm())
		}
	}
}

func TestMirrorReasonLearningModeRequiresVisibleLedger(t *testing.T) {
	request := cliMirrorRequest(axmmirror.GroundingStable, axmmirror.MirrorEscalationNone)
	request.DeterministicResponse = "resolved"
	path := writeMirrorRequest(t, request)
	commandContext, args, err := parseCobraCommand(t, []string{"mirror", "reason"}, []string{path, "--learning-mode", "approved"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runMirrorReason(commandContext, args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "--learn-to") {
		t.Fatalf("err = %v", err)
	}
}

func writeMirrorRequest(t *testing.T, request axmmirror.MirrorNeuralEscalationRequest) string {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func cliMirrorRequest(state, reason string) axmmirror.MirrorNeuralEscalationRequest {
	return axmmirror.MirrorNeuralEscalationRequest{
		Schema:           axmmirror.MirrorNeuralEscalationRequestSchema,
		Prompt:           "What changed?",
		EscalationReason: reason,
		Grounding: axmmirror.LiveNeuralGrounding{
			Schema:      axmmirror.LiveNeuralGroundingSchema,
			State:       state,
			Consequence: axmmirror.ConsequenceLow,
			Authority:   axmmirror.Authority{},
		},
	}
}
