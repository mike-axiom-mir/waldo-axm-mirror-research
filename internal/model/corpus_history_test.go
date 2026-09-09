// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openwaldo/waldo/internal/corpus"
	"github.com/openwaldo/waldo/internal/training"
)

func TestSkipCompletedStagesReusesOnlyExactStageWork(t *testing.T) {
	compose := validCompose()
	prepared := preparedFixture(t, compose.Stages[0])
	inspection := completedStageInspection(t, prepared, RunComplete)

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{prepared}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(skipped, []SkippedCorpus{{Stage: "pretrain", Path: "example"}}) {
		t.Fatalf("skipped = %+v", skipped)
	}
	if len(filtered.Stages) != 0 || len(remaining) != 0 {
		t.Fatalf("filtered = %+v, prepared = %+v", filtered, remaining)
	}
	if len(compose.Stages) != 1 || len(compose.Stages[0].Corpora) != 1 {
		t.Fatal("input compose was mutated")
	}
}

func TestSkipCompletedStagesKeepsChangedCorpusAtSamePath(t *testing.T) {
	compose := validCompose()
	historical := preparedFixture(t, compose.Stages[0])
	inspection := completedStageInspection(t, historical, RunComplete)
	current := historical
	current.BOM.Manifests = append([]corpus.ManifestPin(nil), historical.BOM.Manifests...)
	current.BOM.Manifests[0].SHA256 = strings.Repeat("c", 64)

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{current}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Stages) != 1 || len(remaining) != 1 || len(skipped) != 0 {
		t.Fatalf("changed corpus was skipped: filtered = %+v, prepared = %+v, skipped = %+v", filtered, remaining, skipped)
	}
}

func TestSkipCompletedStagesKeepsChangedParameters(t *testing.T) {
	compose := validCompose()
	historical := preparedFixture(t, compose.Stages[0])
	inspection := completedStageInspection(t, historical, RunComplete)
	compose.Stages[0].Parameters.LearningRate = 0.002
	current := historical
	current.Stage = compose.Stages[0]

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{current}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Stages) != 1 || len(remaining) != 1 || len(skipped) != 0 {
		t.Fatalf("changed parameters were skipped: filtered = %+v, prepared = %+v, skipped = %+v", filtered, remaining, skipped)
	}
}

func TestSkipCompletedStagesNeverPartiallyRewritesAStage(t *testing.T) {
	compose := validCompose()
	compose.Stages[0].Corpora = NewCorpusSelections([]string{"example", "science/new"})
	current := preparedFixture(t, compose.Stages[0])
	historical := current
	historical.BOM.Paths = []string{"example"}
	inspection := completedStageInspection(t, historical, RunComplete)

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{current}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Stages) != 1 || len(remaining) != 1 || len(skipped) != 0 || !reflect.DeepEqual(CorpusPaths(filtered.Stages[0].Corpora), []string{"example", "science/new"}) {
		t.Fatalf("stage was partially rewritten: filtered = %+v, prepared = %+v, skipped = %+v", filtered, remaining, skipped)
	}
}

func TestSkipCompletedStagesIgnoresFailedRuns(t *testing.T) {
	compose := validCompose()
	prepared := preparedFixture(t, compose.Stages[0])
	inspection := completedStageInspection(t, prepared, RunFailed)

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{prepared}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Stages) != 1 || len(remaining) != 1 || len(skipped) != 0 {
		t.Fatalf("failed run was reused: filtered = %+v, prepared = %+v, skipped = %+v", filtered, remaining, skipped)
	}
}

func TestSkipCompletedStagesDoesNotReuseWorkBehindNewerWeights(t *testing.T) {
	compose := validCompose()
	requested := preparedFixture(t, compose.Stages[0])
	newerStage := testStage("fine-tune")
	newer := preparedFixture(t, newerStage)
	first := completedStageInspection(t, requested, RunComplete)
	second := completedStageInspection(t, newer, RunComplete)
	inspection := Inspection{
		Model:   ModelRecord{Runs: append(first.Model.Runs, second.Model.Runs...)},
		RunBOMs: append(first.RunBOMs, second.RunBOMs...),
	}

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{requested}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Stages) != 1 || len(remaining) != 1 || len(skipped) != 0 {
		t.Fatalf("historical work behind newer weights was reused: filtered = %+v, prepared = %+v, skipped = %+v", filtered, remaining, skipped)
	}
}

func TestSkipCompletedStagesReusesCurrentSuffixAsComposePrefix(t *testing.T) {
	oldStage := testStage("old")
	currentStage := testStage("current")
	nextStage := testStage("next")
	old := preparedFixture(t, oldStage)
	current := preparedFixture(t, currentStage)
	next := preparedFixture(t, nextStage)
	oldInspection := completedStageInspection(t, old, RunComplete)
	currentInspection := completedStageInspection(t, current, RunComplete)
	inspection := Inspection{
		Model:   ModelRecord{Runs: append(oldInspection.Model.Runs, currentInspection.Model.Runs...)},
		RunBOMs: append(oldInspection.RunBOMs, currentInspection.RunBOMs...),
	}
	compose := validCompose()
	compose.Stages = []Stage{currentStage, nextStage}

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{current, next}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Stages) != 1 || filtered.Stages[0].Name != "next" || len(remaining) != 1 || remaining[0].Stage.Name != "next" || !reflect.DeepEqual(skipped, []SkippedCorpus{{Stage: "current", Path: "example"}}) {
		t.Fatalf("current suffix was not reused as compose prefix: filtered = %+v, prepared = %+v, skipped = %+v", filtered, remaining, skipped)
	}
}

func TestSkipCompletedStagesMatchesEpochDerivedParameters(t *testing.T) {
	compose := validCompose()
	compose.Stages[0].Parameters.Steps = 0
	compose.Stages[0].Parameters.Epochs = 2
	prepared := preparedFixture(t, compose.Stages[0])
	inspection := completedStageInspection(t, prepared, RunComplete)

	filtered, remaining, skipped, err := SkipCompletedStages(compose, []PreparedStage{prepared}, inspection)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Stages) != 0 || len(remaining) != 0 || len(skipped) != 1 {
		t.Fatalf("exact epoch-derived work was not reused: filtered = %+v, prepared = %+v, skipped = %+v", filtered, remaining, skipped)
	}
}

func completedStageInspection(t *testing.T, prepared PreparedStage, state RunState) Inspection {
	t.Helper()
	corpusHash, err := hashJSON(prepared.BOM)
	if err != nil {
		t.Fatal(err)
	}
	parameters, err := prepared.Stage.ResolvePlanningParameters()
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Stage.Parameters.Steps == 0 && prepared.Stage.Parameters.Tokens == 0 {
		parameters, err = prepared.Stage.ResolveParametersForSteps(4)
		if err != nil {
			t.Fatal(err)
		}
	}
	conversation := training.ConversationTransform{}
	if prepared.Stage.Conversation != nil {
		conversation = *prepared.Stage.Conversation
	}
	return Inspection{
		Model: ModelRecord{Runs: []RunPin{{Stage: prepared.Stage.Name, State: state}}},
		RunBOMs: []RunBOM{{
			Stage: prepared.Stage.Name, StageType: prepared.Stage.Type, Objective: prepared.Stage.Objective,
			Conversation: conversation, CorpusBOMSHA256: corpusHash, CorpusBOM: prepared.BOM, Parameters: parameters,
		}},
	}
}
