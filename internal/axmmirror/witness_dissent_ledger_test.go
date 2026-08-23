package axmmirror

import (
	"bytes"
	"os"
	"testing"
)

func TestWitnessDissentLedgerObservedReceipt(t *testing.T) {
	data, err := os.ReadFile("testdata/witness-dissent-ledger-v0.14.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := VerifyWitnessDissentLedger(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.Source.PeerHolds != 2 || len(r.AppendOnlyLedger) != 3 {
		t.Fatalf("unexpected topology: holds=%d entries=%d", r.Source.PeerHolds, len(r.AppendOnlyLedger))
	}
	w, err := WitnessWitnessDissentLedger(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.State != "DISSENT_HISTORY_WITNESSED" || w.PeerHistoricalHolds != 2 || w.LedgerEntries != 3 || w.Authority != "NONE" {
		t.Fatalf("unexpected witness: %#v", w)
	}
}

func TestWitnessDissentLedgerRejectsHistoricalRewrite(t *testing.T) {
	data, err := os.ReadFile("testdata/witness-dissent-ledger-v0.14.json")
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(data, []byte(`"witnessBHistoricalHoldPreserved": true`), []byte(`"witnessBHistoricalHoldPreserved": false`), 1)
	if bytes.Equal(data, tampered) {
		t.Fatal("tamper did not change fixture")
	}
	if _, err := VerifyWitnessDissentLedger(tampered); err == nil {
		t.Fatal("expected historical rewrite rejection")
	}
}

func TestWitnessDissentLedgerRejectsFinalJudge(t *testing.T) {
	data, err := os.ReadFile("testdata/witness-dissent-ledger-v0.14.json")
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(data, []byte(`"finalJudgeAssigned": false`), []byte(`"finalJudgeAssigned": true`), 1)
	if bytes.Equal(data, tampered) {
		t.Fatal("tamper did not change fixture")
	}
	if _, err := VerifyWitnessDissentLedger(tampered); err == nil {
		t.Fatal("expected final judge rejection")
	}
}
