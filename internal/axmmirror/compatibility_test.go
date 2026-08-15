package axmmirror_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/openwaldo/waldo/internal/axmmirror"
	"github.com/openwaldo/waldo/internal/corpus"
	"github.com/openwaldo/waldo/internal/model"
)

func TestCorpusLensDigestMatchesWALDOSchemaOneHash(t *testing.T) {
	data := readExample(t, "corpus-bom.json")
	var bom corpus.BOM
	if err := json.Unmarshal(data, &bom); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	lens, err := axmmirror.LensCorpusBOM(data)
	if err != nil {
		t.Fatal(err)
	}
	if lens.BOMSHA256 != sha256Hex(canonical) {
		t.Fatalf("lens BOM digest = %s, WALDO schema-1 digest = %s", lens.BOMSHA256, sha256Hex(canonical))
	}
}

func TestRunWitnessDigestMatchesWALDOSchemaOneHash(t *testing.T) {
	runBOMData := readExample(t, "run-bom.json")
	runData := readExample(t, "run.json")
	var bom model.RunBOM
	if err := json.Unmarshal(runBOMData, &bom); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	witness, err := axmmirror.WitnessTrainingRun(runBOMData, runData)
	if err != nil {
		t.Fatal(err)
	}
	if witness.RunBOMSHA256 != sha256Hex(canonical) {
		t.Fatalf("witness run BOM digest = %s, WALDO schema-1 digest = %s", witness.RunBOMSHA256, sha256Hex(canonical))
	}
}

func readExample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "axm-mirror", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
