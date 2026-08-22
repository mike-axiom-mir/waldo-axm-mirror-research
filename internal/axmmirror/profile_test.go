package axmmirror

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestTrainingProfileContractWitnessesCurrentProfiles(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		order     string
		selection string
		weights   map[string]uint64
	}{
		{name: "shuffled", profile: profileShuffled, order: "bounded-shuffle-v1", selection: "lowest-sha256-v1"},
		{name: "balanced", profile: profileBalanced, order: "corpus-balanced-shuffle-v1", selection: "stratified-lowest-sha256-v1"},
		{name: "weighted", profile: profileWeighted, order: "corpus-weighted-shuffle-v1", selection: "stratified-lowest-sha256-v1", weights: map[string]uint64{"research/example": 3}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bom, run := validRunFixture(t, "pytorch", false, "complete")
			setProfileFixture(&bom, &run, test.profile, 1, test.order, test.selection, test.weights)
			contract := profileContractFixture(t, bom, run)
			if contract.State != ProfileContractStateWitnessed || contract.CanonicalProfile != test.profile || contract.AliasUsed {
				t.Fatalf("profile contract = %+v", contract)
			}
			if err := contract.Validate(); err != nil {
				t.Fatalf("contract.Validate() error = %v", err)
			}
		})
	}
}

func TestTrainingProfileContractPreservesLegacyAliases(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		schema    int
		canonical string
		order     string
		selection string
		weights   map[string]uint64
	}{
		{name: "v1", profile: profileLegacyShuffled, schema: 1, canonical: profileShuffled, order: "bounded-shuffle-v1", selection: "lowest-sha256-v1"},
		{name: "v2", profile: profileLegacyBalanced, schema: 2, canonical: profileBalanced, order: "corpus-balanced-shuffle-v1", selection: "stratified-lowest-sha256-v1"},
		{name: "v3", profile: profileLegacyWeighted, schema: 3, canonical: profileWeighted, order: "corpus-weighted-shuffle-v1", selection: "stratified-lowest-sha256-v1", weights: map[string]uint64{"research/example": 5}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bom, run := validRunFixture(t, "pytorch", false, "complete")
			setProfileFixture(&bom, &run, test.profile, test.schema, test.order, test.selection, test.weights)
			contract := profileContractFixture(t, bom, run)
			if contract.State != ProfileContractStateLegacyAlias || !contract.AliasUsed || contract.DeclaredProfile != test.profile || contract.CanonicalProfile != test.canonical {
				t.Fatalf("legacy profile contract = %+v", contract)
			}
		})
	}
}

func TestTrainingProfileContractHoldsUnknownProfile(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	setProfileFixture(&bom, &run, "future-profile", 1, "bounded-shuffle-v1", "lowest-sha256-v1", nil)
	contract := profileContractFixture(t, bom, run)
	if contract.State != ProfileContractStateUnknownProfile || len(contract.Holds) == 0 {
		t.Fatalf("unknown profile contract = %+v", contract)
	}
}

func TestTrainingProfileContractHoldsSchemaMismatch(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	setProfileFixture(&bom, &run, profileBalanced, 2, "corpus-balanced-shuffle-v1", "stratified-lowest-sha256-v1", nil)
	contract := profileContractFixture(t, bom, run)
	if contract.State != ProfileContractStateSchemaMismatch || len(contract.Holds) == 0 {
		t.Fatalf("schema-mismatch profile contract = %+v", contract)
	}
}

func TestTrainingProfileContractHoldsProfileDataMismatch(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	setProfileFixture(&bom, &run, profileWeighted, 1, "bounded-shuffle-v1", "lowest-sha256-v1", nil)
	contract := profileContractFixture(t, bom, run)
	if contract.State != ProfileContractStateParameterMismatch || len(contract.Holds) == 0 {
		t.Fatalf("data-mismatch profile contract = %+v", contract)
	}
}

func TestTrainingProfileContractHoldsDifferentWitness(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	witness := witnessFixture(t, bom, run)
	bom.Parameters.Profile = profileShuffled
	bomData, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := WitnessTrainingProfileContract(bomData, witness)
	if err != nil {
		t.Fatalf("WitnessTrainingProfileContract() error = %v", err)
	}
	if contract.State != ProfileContractStateParameterMismatch || contract.RunBOMDocumentSHA256 == contract.WitnessRunBOMDocumentSHA256 {
		t.Fatalf("different-witness profile contract = %+v", contract)
	}
}

func TestTrainingProfileContractBindsCurrentBudgetAndConversationFields(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	setProfileFixture(&bom, &run, profileShuffled, 1, "bounded-shuffle-v1", "lowest-sha256-v1", nil)
	bom.Objective = "assistant-response-modeling"
	bom.Conversation = waldoConversationTransform{Template: "chatml-v1", SupervisedRoles: []string{"assistant"}}
	bom.Parameters.RequestedTokens = 7
	repinRun(t, bom, &run)
	contract := profileContractFixture(t, bom, run)
	if contract.State != ProfileContractStateWitnessed || contract.RequestedTokens != 7 {
		t.Fatalf("current contract = %+v", contract)
	}
}

func TestTrainingProfileContractSortsAndSealsCorpusWeights(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	setProfileFixture(&bom, &run, profileWeighted, 1, "corpus-weighted-shuffle-v1", "stratified-lowest-sha256-v1", map[string]uint64{"research/b": 2, "research/a": 1})
	contract := profileContractFixture(t, bom, run)
	if contract.CorpusWeights[0].Path != "research/a" || contract.CorpusWeights[1].Path != "research/b" || contract.CorpusWeightsSHA256 == "" {
		t.Fatalf("corpus weights = %+v", contract.CorpusWeights)
	}
	contract.CorpusWeights[0].Weight++
	if err := contract.Validate(); err == nil || !strings.Contains(err.Error(), "receipt digest mismatch") && !strings.Contains(err.Error(), "corpus-weight digest mismatch") {
		t.Fatalf("contract.Validate() tamper error = %v", err)
	}
}

func TestTrainingProfileContractRejectsUnboundedWeightProjection(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	witness := witnessFixture(t, bom, run)
	weights := make(map[string]uint64, maxProfileCorpusWeights+1)
	for i := 0; i <= maxProfileCorpusWeights; i++ {
		weights[fmt.Sprintf("research/%04d", i)] = 1
	}
	bom.Parameters.Profile = profileWeighted
	bom.Parameters.ProfileSchema = 1
	bom.Parameters.Data.Order = "corpus-weighted-shuffle-v1"
	bom.Parameters.Data.CorpusWeights = weights
	bom.Parameters.Evaluation.Selection = "stratified-lowest-sha256-v1"
	bom.EvaluationSet.Selection = "stratified-lowest-sha256-v1"
	bomData, err := json.Marshal(bom)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WitnessTrainingProfileContract(bomData, witness); err == nil || !strings.Contains(err.Error(), "maximum bounded projection") {
		t.Fatalf("unbounded projection error = %v", err)
	}
}

func TestTrainingProfileContractRejectsEscapingWeightPath(t *testing.T) {
	if _, _, err := projectProfileCorpusWeights(map[string]uint64{"../outside": 1}); err == nil || !strings.Contains(err.Error(), "portable relative path") {
		t.Fatalf("escaping corpus-weight path error = %v", err)
	}
}

func TestTrainingProfileContractStatesCausalityCeiling(t *testing.T) {
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	setProfileFixture(&bom, &run, profileShuffled, 1, "bounded-shuffle-v1", "lowest-sha256-v1", nil)
	contract := profileContractFixture(t, bom, run)
	joined := strings.Join(contract.ClaimCeilings, " ")
	if !strings.Contains(joined, "does not prove") || !strings.Contains(joined, "caused a particular output") || !contract.Authority.closed() {
		t.Fatalf("claim ceilings or authority = %+v %+v", contract.ClaimCeilings, contract.Authority)
	}
}

func setProfileFixture(bom *waldoRunBOM, run *waldoRunRecord, profile string, schema int, order, selection string, weights map[string]uint64) {
	bom.Parameters.Profile = profile
	bom.Parameters.ProfileSchema = schema
	bom.Parameters.Data.Order = order
	bom.Parameters.Data.CorpusWeights = weights
	bom.Parameters.Evaluation.Selection = selection
	bom.EvaluationSet.Selection = selection
	if order == "corpus-weighted-shuffle-v1" && len(weights) != 0 {
		bom.CorpusBOM.Paths = make([]string, 0, len(weights))
		for path := range weights {
			bom.CorpusBOM.Paths = append(bom.CorpusBOM.Paths, path)
		}
		sort.Strings(bom.CorpusBOM.Paths)
		corpusData, err := json.Marshal(bom.CorpusBOM)
		if err != nil {
			panic(err)
		}
		bom.CorpusBOMSHA256 = digestBytes(corpusData)
	}
	if order == "corpus-balanced-shuffle-v1" || order == "corpus-weighted-shuffle-v1" {
		run.Observation.Consumption = make([]waldoCorpusConsumption, 0, len(bom.CorpusBOM.Paths))
		remaining := run.Observation.ConsumedTokens
		for i, path := range bom.CorpusBOM.Paths {
			tokens := remaining / int64(len(bom.CorpusBOM.Paths)-i)
			run.Observation.Consumption = append(run.Observation.Consumption, waldoCorpusConsumption{Corpus: path, TokenTargets: tokens})
			remaining -= tokens
		}
	} else {
		run.Observation.Consumption = nil
	}
	repinRunForProfileFixture(bom, run)
}

func repinRunForProfileFixture(bom *waldoRunBOM, run *waldoRunRecord) {
	data, err := json.Marshal(*bom)
	if err != nil {
		panic(err)
	}
	run.BOMSHA256 = digestBytes(data)
}

func profileContractFixture(t *testing.T, bom waldoRunBOM, run waldoRunRecord) TrainingProfileContract {
	t.Helper()
	witness := witnessFixture(t, bom, run)
	bomData, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := WitnessTrainingProfileContract(bomData, witness)
	if err != nil {
		t.Fatalf("WitnessTrainingProfileContract() error = %v", err)
	}
	return contract
}
