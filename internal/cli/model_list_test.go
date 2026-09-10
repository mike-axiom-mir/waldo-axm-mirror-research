// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/openwaldo/waldo/internal/model"
)

func TestModelListUsesReadableCardsWhenTableDoesNotFit(t *testing.T) {
	previous := modelListOutputWidth
	modelListOutputWidth = func() int { return 44 }
	t.Cleanup(func() { modelListOutputWidth = previous })

	models := []model.Listing{
		{Name: "a-very-long-local-model-name-for-research", Parameters: 9_500_000, Updated: "2026-09-10T01:05:21.999300664Z"},
		{Name: "tiny", State: "completed", Parameters: 9_500_000, Runs: 2, Updated: "2026-09-10T01:06:00Z"},
	}
	var output bytes.Buffer
	if err := writeModelList(&output, models); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := utf8.RuneCountInString(line); width > 44 {
			t.Fatalf("line width = %d, want <= 44: %q\n%s", width, line, output.String())
		}
	}
	joined := strings.Join(strings.Fields(output.String()), "")
	for _, want := range []string{
		"a-very-long-local-model-name-for-research",
		"2026-09-10T01:05:21.999300664Z",
		"STATEcompleted",
		"NEXTwaldomodelsummary<name>",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("compact list missing %q:\n%s", want, output.String())
		}
	}
}

func TestModelListRetainsTableWhenItFits(t *testing.T) {
	previous := modelListOutputWidth
	modelListOutputWidth = func() int { return 120 }
	t.Cleanup(func() { modelListOutputWidth = previous })

	var output bytes.Buffer
	if err := writeModelList(&output, []model.Listing{{Name: "tiny", Parameters: 9_500_000, Updated: "2026-09-10T01:06:00Z"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "NAME  STATE") || strings.Contains(output.String(), "MODELS  1") {
		t.Fatalf("wide list did not retain the table:\n%s", output.String())
	}
}

func TestEmptyModelListAcknowledgesFiltersAndGivesNextAction(t *testing.T) {
	previous := modelListOutputWidth
	modelListOutputWidth = func() int { return 32 }
	defer func() { modelListOutputWidth = previous }()

	var output bytes.Buffer
	if err := writeEmptyModelList(&output, []string{"a-very-long-missing-model-pattern-*"}); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := utf8.RuneCountInString(line); width > 32 {
			t.Fatalf("line width = %d, want <= 32: %q\n%s", width, line, output.String())
		}
	}
	joined := strings.Join(strings.Fields(output.String()), "")
	for _, want := range []string{"NOMATCHa-very-long-missing-model-pattern-*", "NEXTwaldomodellist"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("empty result missing %q:\n%s", want, output.String())
		}
	}
}

func TestEmptyModelShelfExplainsHowToCreateOne(t *testing.T) {
	previous := modelListOutputWidth
	modelListOutputWidth = func() int { return 44 }
	defer func() { modelListOutputWidth = previous }()

	var output bytes.Buffer
	if err := writeEmptyModelList(&output, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "No local models found.") || !strings.Contains(output.String(), "waldo model init <name> --preset 10m") {
		t.Fatalf("empty shelf lacks orientation:\n%s", output.String())
	}
}
