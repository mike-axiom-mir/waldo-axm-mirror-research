package axmmirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortableCapabilitySpineFixturesPinInitialReceipts(t *testing.T) {
	examples := filepath.Join("..", "..", "examples", "axm-mirror")

	var censusRequest SelfCapabilityCensusRequest
	readCapabilityFixture(t, filepath.Join(examples, "self-capability-census-request.json"), &censusRequest)
	self, err := CensusSelfCapabilities(censusRequest)
	if err != nil {
		t.Fatal(err)
	}
	if self.SnapshotSHA256 != "e6bd00f50f5d3f5c636a546a1185fbe2ed999c7488f8c161c85c2a1491f7e6ca" {
		t.Fatalf("self capability snapshot digest = %s", self.SnapshotSHA256)
	}

	var externalSnapshot ExternalCapabilitySnapshot
	readCapabilityFixture(t, filepath.Join(examples, "external-capability-snapshot.json"), &externalSnapshot)
	external, err := IntakeExternalCapabilities(externalSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if external.State != ExternalCapabilitySourceOnly || external.ReceiptSHA256 != "e35df8a87901637987278b382bedfb2f370bb40cf515fd63d7e8e1f79ab0a21b" {
		t.Fatalf("external capability receipt = %s %s", external.State, external.ReceiptSHA256)
	}

	var request CapabilityGapRequest
	readCapabilityFixture(t, filepath.Join(examples, "capability-gap-request.json"), &request)
	plan, err := PlanCapabilityHandoff(self, external, request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.State != HandoffStateSourceOnly || plan.PlanSHA256 != "53accce73f893b5d1ec8e7c4604095fe7de2cc7c78e28d192876dc6f87d6690a" {
		t.Fatalf("capability handoff plan = %s %s", plan.State, plan.PlanSHA256)
	}

	var declaration CapabilityTranslationDeclaration
	readCapabilityFixture(t, filepath.Join(examples, "capability-translation-declaration.json"), &declaration)
	translation, err := SealCapabilityTranslation(plan, declaration)
	if err != nil {
		t.Fatal(err)
	}
	if translation.State != TranslationStateNotRequired || translation.ReceiptSHA256 != "50873dc0e1da8bc02ea5171cfa641327f883956aa3b936aafbe158142feb6883" {
		t.Fatalf("translation loss receipt = %s %s", translation.State, translation.ReceiptSHA256)
	}

	var draft CapabilityReturnDraft
	readCapabilityFixture(t, filepath.Join(examples, "capability-return-draft.json"), &draft)
	returned, err := VerifyCapabilityReturn(plan, translation, draft)
	if err != nil {
		t.Fatal(err)
	}
	if returned.State != CapabilityReturnPlanHold || returned.ReceiptSHA256 != "a2dffea92f43f44d4519eac7bb1952f7b8c853e28416ce1bf9d15cb89f1859c9" {
		t.Fatalf("capability return receipt = %s %s", returned.State, returned.ReceiptSHA256)
	}
}

func readCapabilityFixture(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := DecodeStrictJSON(data, target); err != nil {
		t.Fatal(err)
	}
}
