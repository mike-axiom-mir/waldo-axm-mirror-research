// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"
	"reflect"

	"github.com/openwaldo/waldo/internal/training"
)

// SkippedCorpus identifies a compose selection omitted because its complete
// stage work identity already has a successful run in this model.
type SkippedCorpus struct {
	Stage string `json:"stage"`
	Path  string `json:"path"`
}

// SkipCompletedStages removes complete stages whose immutable work identity
// matches a successful historical run. A stage is never partially rewritten:
// changed corpus bytes, parameters, objective, or conversation handling keep
// the entire declared stage executable.
func SkipCompletedStages(compose Compose, prepared []PreparedStage, inspection Inspection) (Compose, []PreparedStage, []SkippedCorpus, error) {
	if len(compose.Stages) != len(prepared) {
		return Compose{}, nil, nil, fmt.Errorf("compose has %d stages but preflight prepared %d", len(compose.Stages), len(prepared))
	}
	for index, stage := range compose.Stages {
		if prepared[index].Stage.Name != stage.Name {
			return Compose{}, nil, nil, fmt.Errorf("compose stage %d is %s but preflight prepared %s", index+1, stage.Name, prepared[index].Stage.Name)
		}
	}

	latestComplete := -1
	for position, pin := range inspection.Model.Runs {
		if pin.State == RunComplete && position < len(inspection.RunBOMs) {
			latestComplete = position
		}
	}
	matched := 0
	for count := min(len(prepared), latestComplete+1); count > 0; count-- {
		start := latestComplete - count + 1
		exact := true
		for offset := 0; offset < count; offset++ {
			position := start + offset
			if inspection.Model.Runs[position].State != RunComplete {
				exact = false
				break
			}
			matches, err := preparedStageMatchesRun(prepared[offset], inspection.RunBOMs[position])
			if err != nil {
				return Compose{}, nil, nil, fmt.Errorf("compare stage %s with completed run %d: %w", prepared[offset].Stage.Name, position+1, err)
			}
			if !matches {
				exact = false
				break
			}
		}
		if exact {
			matched = count
			break
		}
	}

	filtered := compose
	filtered.Stages = append([]Stage(nil), compose.Stages[matched:]...)
	filteredPrepared := append([]PreparedStage(nil), prepared[matched:]...)
	var skipped []SkippedCorpus
	for _, stage := range compose.Stages[:matched] {
		for _, selection := range stage.Corpora {
			skipped = append(skipped, SkippedCorpus{Stage: stage.Name, Path: selection.Path})
		}
	}
	return filtered, filteredPrepared, skipped, nil
}

func preparedStageMatchesRun(prepared PreparedStage, bom RunBOM) (bool, error) {
	corpusHash, err := hashJSON(prepared.BOM)
	if err != nil {
		return false, err
	}
	parameters, err := prepared.Stage.ResolvePlanningParameters()
	if err != nil {
		return false, err
	}
	if prepared.Stage.Parameters.Steps == 0 && prepared.Stage.Parameters.Tokens == 0 {
		parameters, err = prepared.Stage.ResolveParametersForSteps(bom.Parameters.Steps)
		if err != nil {
			return false, err
		}
	}
	if parameters.Data.Order == "corpus-weighted-shuffle-v1" {
		parameters.Data.CorpusWeights, err = resolveCorpusWeights(parameters.Data.CorpusWeights, prepared.BOM.Paths)
		if err != nil {
			return false, err
		}
	}
	conversation := training.ConversationTransform{}
	if prepared.Stage.Conversation != nil {
		conversation = *prepared.Stage.Conversation
	}
	return bom.Stage == prepared.Stage.Name &&
		bom.StageType == prepared.Stage.Type &&
		bom.Objective == prepared.Stage.Objective &&
		reflect.DeepEqual(bom.Conversation, conversation) &&
		bom.CorpusBOMSHA256 == corpusHash &&
		equivalentTrainingParameters(bom.Parameters, parameters), nil
}
