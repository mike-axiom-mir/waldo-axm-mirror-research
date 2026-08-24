package axmmirror

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	waldorecord "github.com/openwaldo/waldo/internal/record"
)

func TestExperienceTrajectoryLearnsFromFailureWithoutImitatingIt(t *testing.T) {
	record := MirrorExperienceTrajectoryRecord{
		Schema: MirrorExperienceTrajectorySchema, ID: "website-v051-source-hold",
		SourceReceiptSHA256: strings.Repeat("a", 64), ObservedAt: time.Date(2026, 8, 24, 20, 0, 0, 0, time.UTC),
		OutcomeSignal: MirrorExperienceInconclusive, Prompt: "Create a website.",
		AttemptTrace: "Planning passed and source author was absent.", ObservedOutcome: "No website artifact existed.",
		Lesson:     "Retain the failed attempt and restore the WALDO source author before resuming.",
		TargetKind: MirrorExperienceReflectionTarget, TrainingObjective: "assistant-response-modeling",
		ReflectionTargetSupervised: true, Authority: Authority{},
	}
	if err := SealMirrorExperienceTrajectory(&record); err != nil {
		t.Fatal(err)
	}
	projection, receipt, err := ProjectMirrorExperienceTrajectory(record)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Messages        []MirrorGroundMessage `json:"messages"`
		SupervisedRoles []string              `json:"supervisedRoles"`
	}
	if err := json.NewDecoder(bytes.NewReader(projection)).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Messages) != 3 || decoded.Messages[0].Role != "user" || decoded.Messages[1].Role != "tool" || decoded.Messages[2].Role != "assistant" {
		t.Fatalf("trajectory messages = %+v", decoded.Messages)
	}
	if decoded.Messages[2].Content != record.Lesson || strings.Contains(decoded.Messages[2].Content, record.AttemptTrace) {
		t.Fatalf("supervised reflection = %+v", decoded.Messages[2])
	}
	if len(decoded.SupervisedRoles) != 1 || decoded.SupervisedRoles[0] != "assistant" {
		t.Fatalf("supervised roles = %v", decoded.SupervisedRoles)
	}
	conversation := waldorecord.Conversation{}
	for _, message := range decoded.Messages {
		conversation.Messages = append(conversation.Messages, waldorecord.Message{Role: message.Role, Content: message.Content})
	}
	if err := conversation.Validate(); err != nil {
		t.Fatalf("normal WALDO conversation projection: %v", err)
	}
	if receipt.State != "EXPERIENCE_REFLECTION_PROJECTED" || !receipt.ObservedExperience || receipt.CompletionRequired || receipt.FailedAttemptSupervised || !receipt.ReflectionTargetSupervised {
		t.Fatalf("receipt = %+v", receipt)
	}
	if receipt.TrainingInvoked || receipt.WeightsChanged || len(receipt.ReceiptSHA256) != 64 {
		t.Fatalf("receipt truth = %+v", receipt)
	}

	t.Run("tamper refused", func(t *testing.T) {
		tampered := record
		tampered.Lesson = "changed"
		if err := tampered.Validate(); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
			t.Fatalf("tamper error = %v", err)
		}
	})
	t.Run("completion gate refused", func(t *testing.T) {
		invalid := record
		invalid.CompletionRequired = true
		if err := SealMirrorExperienceTrajectory(&invalid); err == nil || !strings.Contains(err.Error(), "without requiring completion") {
			t.Fatalf("completion error = %v", err)
		}
	})
	t.Run("failed imitation refused", func(t *testing.T) {
		invalid := record
		invalid.FailedAttemptSupervised = true
		if err := SealMirrorExperienceTrajectory(&invalid); err == nil || !strings.Contains(err.Error(), "without requiring completion") {
			t.Fatalf("imitation error = %v", err)
		}
	})
}
