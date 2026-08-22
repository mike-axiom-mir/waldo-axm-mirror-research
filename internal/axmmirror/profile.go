package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	TrainingProfileContractSchema = "axm.waldo-witness.training-profile-contract/v0.1"

	ProfileContractStateWitnessed         = "PROFILE_CONTRACT_WITNESSED"
	ProfileContractStateLegacyAlias       = "LEGACY_ALIAS_WITNESSED"
	ProfileContractStateUnknownProfile    = "HOLD_UNKNOWN_PROFILE_CONTRACT"
	ProfileContractStateSchemaMismatch    = "HOLD_PROFILE_SCHEMA_MISMATCH"
	ProfileContractStateParameterMismatch = "HOLD_PARAMETER_WITNESS_MISMATCH"

	maxProfileCorpusWeights   = 1024
	maxProfileCorpusPathBytes = 4096
)

const (
	profileShuffled       = "causal-pretrain-shuffled"
	profileBalanced       = "causal-pretrain-balanced"
	profileWeighted       = "causal-pretrain-weighted"
	profileLegacyShuffled = "causal-pretrain-v1"
	profileLegacyBalanced = "causal-pretrain-v2"
	profileLegacyWeighted = "causal-pretrain-v3"
)

type ProfileCorpusWeight struct {
	Path   string `json:"path"`
	Weight uint64 `json:"weight"`
}

type ProfileEvaluationContract struct {
	Selection  string  `json:"selection"`
	Fraction   float64 `json:"fraction"`
	MaxRecords int     `json:"max_records"`
	MaxBytes   int64   `json:"max_bytes"`
}

// TrainingProfileContract projects the data-selection behavior encoded in one
// immutable WALDO run BOM. The exact BOM and its already verified run witness
// remain authoritative; this receipt only makes the profile contract explicit.
type TrainingProfileContract struct {
	Schema                      string                     `json:"schema"`
	State                       string                     `json:"state"`
	RunID                       string                     `json:"run_id"`
	ModelID                     string                     `json:"model_id"`
	RunWitnessState             string                     `json:"run_witness_state"`
	RunBOMSHA256                string                     `json:"run_bom_sha256"`
	RunBOMDocumentSHA256        string                     `json:"run_bom_document_sha256"`
	WitnessRunBOMSHA256         string                     `json:"witness_run_bom_sha256"`
	WitnessRunBOMDocumentSHA256 string                     `json:"witness_run_bom_document_sha256"`
	RunWitnessReceiptSHA256     string                     `json:"run_witness_receipt_sha256"`
	ParametersSHA256            string                     `json:"parameters_sha256"`
	WitnessParametersSHA256     string                     `json:"witness_parameters_sha256"`
	DeclaredProfile             string                     `json:"declared_profile"`
	CanonicalProfile            string                     `json:"canonical_profile"`
	AliasUsed                   bool                       `json:"alias_used"`
	ProfileSchema               int                        `json:"profile_schema"`
	Epochs                      int64                      `json:"epochs"`
	RequestedTokens             int64                      `json:"requested_tokens,omitempty"`
	Steps                       int64                      `json:"steps"`
	PlannedTokenCapacity        int64                      `json:"planned_token_capacity"`
	Seed                        uint64                     `json:"seed"`
	DataOrder                   string                     `json:"data_order"`
	ShuffleBufferRecords        int                        `json:"shuffle_buffer_records"`
	ShuffleBufferBytes          int64                      `json:"shuffle_buffer_bytes"`
	Packing                     string                     `json:"packing"`
	CorpusWeightsSHA256         string                     `json:"corpus_weights_sha256"`
	CorpusWeights               []ProfileCorpusWeight      `json:"corpus_weights"`
	Evaluation                  *ProfileEvaluationContract `json:"evaluation,omitempty"`
	Holds                       []string                   `json:"holds,omitempty"`
	ClaimCeilings               []string                   `json:"claim_ceilings"`
	ReceiptSHA256               string                     `json:"receipt_sha256,omitempty"`
	Authority                   Authority                  `json:"authority"`
}

// WitnessTrainingProfileContract accepts one exact documented RUN-BOM.json and
// its existing witness receipt. Valid but unsupported or mismatched contracts
// produce a typed HOLD receipt instead of being normalized silently.
func WitnessTrainingProfileContract(runBOMData []byte, witness TrainingRunWitness) (TrainingProfileContract, error) {
	if err := witness.Validate(); err != nil {
		return TrainingProfileContract{}, fmt.Errorf("training run witness: %w", err)
	}
	var runBOM waldoRunBOM
	if err := decodeJSON(runBOMData, &runBOM, false); err != nil {
		return TrainingProfileContract{}, fmt.Errorf("decode training run BOM: %w", err)
	}
	if err := validateRunBOM(runBOM); err != nil {
		return TrainingProfileContract{}, err
	}
	canonicalRunBOM, err := json.Marshal(runBOM)
	if err != nil {
		return TrainingProfileContract{}, fmt.Errorf("canonicalize training run BOM: %w", err)
	}
	parameterData, err := json.Marshal(runBOM.Parameters)
	if err != nil {
		return TrainingProfileContract{}, fmt.Errorf("canonicalize training parameters: %w", err)
	}
	weights, weightsDigest, err := projectProfileCorpusWeights(runBOM.Parameters.Data.CorpusWeights)
	if err != nil {
		return TrainingProfileContract{}, err
	}

	canonicalProfile, expectedSchema, aliasUsed, knownProfile := resolveProfileIdentity(runBOM.Parameters.Profile)
	contract := TrainingProfileContract{
		Schema: TrainingProfileContractSchema,
		RunID:  runBOM.ID, ModelID: runBOM.ModelID, RunWitnessState: witness.State,
		RunBOMSHA256: digestBytes(canonicalRunBOM), RunBOMDocumentSHA256: digestBytes(runBOMData),
		WitnessRunBOMSHA256: witness.RunBOMSHA256, WitnessRunBOMDocumentSHA256: witness.RunBOMDocumentSHA256,
		RunWitnessReceiptSHA256: witness.ReceiptSHA256,
		ParametersSHA256:        digestBytes(parameterData), WitnessParametersSHA256: witness.Parameters.SHA256,
		DeclaredProfile: runBOM.Parameters.Profile, CanonicalProfile: canonicalProfile,
		AliasUsed: aliasUsed, ProfileSchema: runBOM.Parameters.ProfileSchema,
		Epochs: runBOM.Parameters.Epochs, RequestedTokens: runBOM.Parameters.RequestedTokens,
		Steps: runBOM.Parameters.Steps, PlannedTokenCapacity: runBOM.Parameters.PlannedTokenCapacity,
		Seed:                 runBOM.Parameters.Seed,
		DataOrder:            runBOM.Parameters.Data.Order,
		ShuffleBufferRecords: runBOM.Parameters.Data.ShuffleBufferRecords,
		ShuffleBufferBytes:   runBOM.Parameters.Data.ShuffleBufferBytes,
		Packing:              runBOM.Parameters.Data.Packing,
		CorpusWeightsSHA256:  weightsDigest, CorpusWeights: weights,
		ClaimCeilings: []string{
			"the receipt witnesses a declared training-selection contract, not the quality or safety of the resulting model",
			"corpus selection or weight does not prove that a source caused a particular output",
			"recorded license assertions are not legal conclusions",
		},
		Authority: Authority{},
	}
	if policy := runBOM.Parameters.Evaluation; policy != nil {
		contract.Evaluation = &ProfileEvaluationContract{
			Selection: policy.Selection, Fraction: policy.Fraction,
			MaxRecords: policy.MaxRecords, MaxBytes: policy.MaxBytes,
		}
	}

	parameterMismatch := false
	addParameterHold := func(reason string) {
		parameterMismatch = true
		contract.Holds = append(contract.Holds, reason)
	}
	if contract.RunID != witness.RunID || contract.ModelID != witness.ModelID {
		addParameterHold("run or model identity does not match the supplied training-run witness")
	}
	if contract.RunBOMSHA256 != witness.RunBOMSHA256 || contract.RunBOMDocumentSHA256 != witness.RunBOMDocumentSHA256 {
		addParameterHold("run BOM identity does not match the supplied training-run witness")
	}
	if contract.ParametersSHA256 != witness.Parameters.SHA256 ||
		contract.DeclaredProfile != witness.Parameters.Profile ||
		contract.ProfileSchema != witness.Parameters.ProfileSchema ||
		contract.Epochs != witness.Parameters.Epochs ||
		contract.Steps != witness.Parameters.Steps ||
		contract.Seed != witness.Parameters.Seed ||
		contract.PlannedTokenCapacity != witness.Parameters.PlannedTokenCapacity {
		addParameterHold("training parameters do not match the supplied training-run witness")
	}
	if knownProfile {
		for _, reason := range profileDataContractHolds(canonicalProfile, runBOM.Parameters, runBOM.CorpusBOM.Paths) {
			addParameterHold(reason)
		}
	}

	switch {
	case parameterMismatch:
		contract.State = ProfileContractStateParameterMismatch
	case !knownProfile:
		contract.State = ProfileContractStateUnknownProfile
		contract.Holds = append(contract.Holds, "training profile has no witnessed behavior contract")
	case contract.ProfileSchema != expectedSchema:
		contract.State = ProfileContractStateSchemaMismatch
		contract.Holds = append(contract.Holds, fmt.Sprintf("profile schema %d does not match the witnessed schema %d for %s", contract.ProfileSchema, expectedSchema, contract.DeclaredProfile))
	case aliasUsed:
		contract.State = ProfileContractStateLegacyAlias
	default:
		contract.State = ProfileContractStateWitnessed
	}

	contract.ReceiptSHA256, err = profileContractReceiptDigest(contract)
	if err != nil {
		return TrainingProfileContract{}, err
	}
	if err := contract.Validate(); err != nil {
		return TrainingProfileContract{}, fmt.Errorf("generated training profile contract: %w", err)
	}
	return contract, nil
}

func (contract TrainingProfileContract) Validate() error {
	if contract.Schema != TrainingProfileContractSchema || !oneOf(contract.State,
		ProfileContractStateWitnessed,
		ProfileContractStateLegacyAlias,
		ProfileContractStateUnknownProfile,
		ProfileContractStateSchemaMismatch,
		ProfileContractStateParameterMismatch,
	) {
		return fmt.Errorf("unsupported training profile contract identity %q state %q", contract.Schema, contract.State)
	}
	if strings.TrimSpace(contract.RunID) == "" || strings.TrimSpace(contract.ModelID) == "" || strings.TrimSpace(contract.DeclaredProfile) == "" || strings.TrimSpace(contract.CanonicalProfile) == "" {
		return errors.New("training profile contract has incomplete run or profile identity")
	}
	if !oneOf(contract.RunWitnessState, RunWitnessStateReady, RunWitnessStateHold) {
		return fmt.Errorf("training profile contract has unsupported run witness state %q", contract.RunWitnessState)
	}
	for _, item := range []struct{ name, value string }{
		{"profile contract.run_bom_sha256", contract.RunBOMSHA256},
		{"profile contract.run_bom_document_sha256", contract.RunBOMDocumentSHA256},
		{"profile contract.witness_run_bom_sha256", contract.WitnessRunBOMSHA256},
		{"profile contract.witness_run_bom_document_sha256", contract.WitnessRunBOMDocumentSHA256},
		{"profile contract.run_witness_receipt_sha256", contract.RunWitnessReceiptSHA256},
		{"profile contract.parameters_sha256", contract.ParametersSHA256},
		{"profile contract.witness_parameters_sha256", contract.WitnessParametersSHA256},
		{"profile contract.corpus_weights_sha256", contract.CorpusWeightsSHA256},
		{"profile contract.receipt_sha256", contract.ReceiptSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	if contract.ProfileSchema < 1 || contract.Epochs < 1 || contract.Steps < 1 || contract.PlannedTokenCapacity < 1 || contract.ShuffleBufferRecords < 1 || contract.ShuffleBufferBytes < 1 || strings.TrimSpace(contract.DataOrder) == "" || strings.TrimSpace(contract.Packing) == "" {
		return errors.New("training profile contract has invalid profile, budget, or data-plan fields")
	}
	if contract.RequestedTokens < 0 || contract.RequestedTokens > contract.PlannedTokenCapacity {
		return errors.New("training profile contract has invalid requested token budget")
	}
	if len(contract.CorpusWeights) > maxProfileCorpusWeights {
		return fmt.Errorf("training profile contract exceeds the %d-entry corpus-weight projection bound", maxProfileCorpusWeights)
	}
	previous := ""
	for i, item := range contract.CorpusWeights {
		if strings.TrimSpace(item.Path) == "" || len(item.Path) > maxProfileCorpusPathBytes || item.Weight == 0 || i > 0 && item.Path <= previous {
			return fmt.Errorf("training profile contract has invalid or unsorted corpus weight at position %d", i)
		}
		if err := validateRelativePath(fmt.Sprintf("training profile contract corpus weight %d path", i), item.Path); err != nil {
			return err
		}
		previous = item.Path
	}
	weightsDigest, err := profileCorpusWeightsDigest(contract.CorpusWeights)
	if err != nil {
		return err
	}
	if weightsDigest != contract.CorpusWeightsSHA256 {
		return fmt.Errorf("training profile corpus-weight digest mismatch: expected %s, got %s", contract.CorpusWeightsSHA256, weightsDigest)
	}
	if contract.Evaluation != nil {
		policy := contract.Evaluation
		if strings.TrimSpace(policy.Selection) == "" || !finite(policy.Fraction) || policy.Fraction < 0 || policy.Fraction >= 1 || policy.MaxRecords < 0 || policy.MaxBytes < 0 {
			return errors.New("training profile contract has invalid evaluation policy")
		}
	}
	canonical, expectedSchema, aliasUsed, known := resolveProfileIdentity(contract.DeclaredProfile)
	if canonical != contract.CanonicalProfile || aliasUsed != contract.AliasUsed {
		return errors.New("training profile contract declared and canonical identities are inconsistent")
	}
	switch contract.State {
	case ProfileContractStateWitnessed:
		if !known || aliasUsed || contract.ProfileSchema != expectedSchema || len(contract.Holds) != 0 || contract.Evaluation == nil {
			return errors.New("witnessed training profile contract has unresolved identity or evidence")
		}
	case ProfileContractStateLegacyAlias:
		if !known || !aliasUsed || contract.ProfileSchema != expectedSchema || len(contract.Holds) != 0 || contract.Evaluation == nil {
			return errors.New("legacy training profile contract has unresolved identity or evidence")
		}
	default:
		if len(contract.Holds) == 0 {
			return errors.New("held training profile contract requires at least one reason")
		}
	}
	if len(contract.ClaimCeilings) == 0 {
		return errors.New("training profile contract must state its claim ceilings")
	}
	for i, ceiling := range contract.ClaimCeilings {
		if strings.TrimSpace(ceiling) == "" {
			return fmt.Errorf("training profile contract claim ceiling %d is empty", i)
		}
	}
	if !contract.Authority.closed() {
		return errors.New("training profile contract must carry closed authority")
	}
	actual, err := profileContractReceiptDigest(contract)
	if err != nil {
		return err
	}
	if actual != contract.ReceiptSHA256 {
		return fmt.Errorf("training profile contract receipt digest mismatch: expected %s, got %s", contract.ReceiptSHA256, actual)
	}
	return nil
}

func projectProfileCorpusWeights(input map[string]uint64) ([]ProfileCorpusWeight, string, error) {
	if len(input) > maxProfileCorpusWeights {
		return nil, "", fmt.Errorf("training profile has %d corpus weights; maximum bounded projection is %d", len(input), maxProfileCorpusWeights)
	}
	weights := make([]ProfileCorpusWeight, 0, len(input))
	for path, weight := range input {
		if strings.TrimSpace(path) == "" || len(path) > maxProfileCorpusPathBytes || weight == 0 {
			return nil, "", fmt.Errorf("training profile has invalid corpus weight for path %q", path)
		}
		if err := validateRelativePath("training profile corpus weight path", path); err != nil {
			return nil, "", err
		}
		weights = append(weights, ProfileCorpusWeight{Path: path, Weight: weight})
	}
	sort.Slice(weights, func(i, j int) bool { return weights[i].Path < weights[j].Path })
	digest, err := profileCorpusWeightsDigest(weights)
	if err != nil {
		return nil, "", err
	}
	return weights, digest, nil
}

func profileCorpusWeightsDigest(weights []ProfileCorpusWeight) (string, error) {
	data, err := json.Marshal(weights)
	if err != nil {
		return "", fmt.Errorf("encode training profile corpus weights: %w", err)
	}
	return digestBytes(data), nil
}

func resolveProfileIdentity(declared string) (canonical string, expectedSchema int, aliasUsed, known bool) {
	switch declared {
	case profileShuffled, profileBalanced, profileWeighted:
		return declared, 1, false, true
	case profileLegacyShuffled:
		return profileShuffled, 1, true, true
	case profileLegacyBalanced:
		return profileBalanced, 2, true, true
	case profileLegacyWeighted:
		return profileWeighted, 3, true, true
	default:
		return declared, 0, false, false
	}
}

func profileDataContractHolds(canonical string, parameters waldoResolvedParameters, selectedPaths []string) []string {
	var holds []string
	expectedOrder, expectedSelection := "", ""
	switch canonical {
	case profileShuffled:
		expectedOrder, expectedSelection = "bounded-shuffle-v1", "lowest-sha256-v1"
		if len(parameters.Data.CorpusWeights) != 0 {
			holds = append(holds, "shuffled profile must not carry corpus weights")
		}
	case profileBalanced:
		expectedOrder, expectedSelection = "corpus-balanced-shuffle-v1", "stratified-lowest-sha256-v1"
		if len(parameters.Data.CorpusWeights) != 0 {
			holds = append(holds, "balanced profile must not carry corpus weights")
		}
	case profileWeighted:
		expectedOrder, expectedSelection = "corpus-weighted-shuffle-v1", "stratified-lowest-sha256-v1"
		if len(parameters.Data.CorpusWeights) == 0 {
			holds = append(holds, "weighted profile requires a positive corpus-weight map")
		} else {
			selected := make(map[string]bool, len(selectedPaths))
			for _, path := range selectedPaths {
				selected[path] = true
				if parameters.Data.CorpusWeights[path] == 0 {
					holds = append(holds, fmt.Sprintf("weighted profile does not declare selected corpus %q", path))
				}
			}
			for path := range parameters.Data.CorpusWeights {
				if !selected[path] {
					holds = append(holds, fmt.Sprintf("weighted profile declares unselected corpus %q", path))
				}
			}
		}
	}
	if parameters.Data.Order != expectedOrder {
		holds = append(holds, fmt.Sprintf("data order %q does not match profile contract %q", parameters.Data.Order, expectedOrder))
	}
	if parameters.Data.Packing != "continuous-eos-v1" {
		holds = append(holds, fmt.Sprintf("packing contract %q is not witnessed by this profile schema", parameters.Data.Packing))
	}
	if parameters.Evaluation == nil {
		holds = append(holds, "resolved evaluation policy is not provided")
	} else if parameters.Evaluation.Selection != expectedSelection {
		holds = append(holds, fmt.Sprintf("evaluation selection %q does not match profile contract %q", parameters.Evaluation.Selection, expectedSelection))
	}
	return holds
}

func profileContractReceiptDigest(contract TrainingProfileContract) (string, error) {
	contract.ReceiptSHA256 = ""
	data, err := json.Marshal(contract)
	if err != nil {
		return "", fmt.Errorf("encode training profile contract receipt: %w", err)
	}
	return digestBytes(data), nil
}
