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
	if self.SnapshotSHA256 != "8a6829e4aa7a9c5216489828cc8bc18be6012611ba09268b75fad5bd46c71d6c" {
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
	if plan.State != HandoffStateSourceOnly || plan.PlanSHA256 != "222dbe59220065182e896c71308c4cd8cb56413e70e474facfa3e3012e6014e1" {
		t.Fatalf("capability handoff plan = %s %s", plan.State, plan.PlanSHA256)
	}

	var declaration CapabilityTranslationDeclaration
	readCapabilityFixture(t, filepath.Join(examples, "capability-translation-declaration.json"), &declaration)
	translation, err := SealCapabilityTranslation(plan, declaration)
	if err != nil {
		t.Fatal(err)
	}
	if translation.State != TranslationStateNotRequired || translation.ReceiptSHA256 != "7a1243ccf07a59b26c180e460740778ded473cb0ebe44e0379e7aa51cb2acf48" {
		t.Fatalf("translation loss receipt = %s %s", translation.State, translation.ReceiptSHA256)
	}

	var draft CapabilityReturnDraft
	readCapabilityFixture(t, filepath.Join(examples, "capability-return-draft.json"), &draft)
	returned, err := VerifyCapabilityReturn(plan, translation, draft)
	if err != nil {
		t.Fatal(err)
	}
	if returned.State != CapabilityReturnPlanHold || returned.ReceiptSHA256 != "eb72dad9ecb2804213b753b72c2693a7ad68323dfb5ffd747dfc7fe8c0479899" {
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
