// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestModelSummaryViewKeepsExactFactsInsideNarrowTerminal(t *testing.T) {
	raw := strings.Join([]string{
		"NAME:          a-very-long-local-model-name-for-summary-review",
		"STATE:         completed",
		"ARCHITECTURE:  0123456789ab transformer, 24 layers, width 1024, 16/8 heads",
		"RUN 0003:      supervised-finetuning completed 1.2M tokens, checkpoint step 125000, 2 attempts",
		"  EVALUATION:  held-out loss initial 3.1415, best 2.7182, final 2.8000; reloaded artifact 2.8000",
		"  CORPUS:      research/a-very-long-corpus-name/2026-09-10 987.6k token targets",
		"  TELEMETRY:   /home/researcher/.waldo/models/a-very-long-local-model-name-for-summary-review/runs/0003/TELEMETRY.json",
		"ADVICE:        continue — the interrupted stage has a verified checkpoint and can be resumed without changing the declared model architecture",
		"  - verify the retained checkpoint evidence before continuing the run",
	}, "\n") + "\n"

	var output bytes.Buffer
	if err := writeModelSummaryView(&output, raw, 44); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := utf8.RuneCountInString(line); width > 44 {
			t.Fatalf("line width = %d, want <= 44: %q\n%s", width, line, output.String())
		}
	}
	if compactSummaryLayout(output.String()) != compactSummaryLayout(raw) {
		t.Fatalf("narrow realization lost or invented non-whitespace content:\n%s", output.String())
	}
	for _, label := range []string{"ARCHITECTURE:", "RUN 0003:", "  EVALUATION:", "  CORPUS:", "  TELEMETRY:", "ADVICE:", "  - "} {
		if !strings.Contains(output.String(), label) {
			t.Fatalf("narrow realization lost label %q:\n%s", label, output.String())
		}
	}
}

func TestModelSummaryViewPreservesHumanOutputByteForByteWhenItFits(t *testing.T) {
	raw := "NAME:          canary\nSTATE:         untrained\nADVICE:        train — add verified corpus data\n"
	var output bytes.Buffer
	if err := writeModelSummaryView(&output, raw, 120); err != nil {
		t.Fatal(err)
	}
	if output.String() != raw {
		t.Fatalf("wide summary changed despite fitting:\nwant %q\n got %q", raw, output.String())
	}
}

func TestModelSummaryViewBreaksLongUnspacedEvidenceWithoutLoss(t *testing.T) {
	raw := "TELEMETRY:   /this/is/a/single/very/long/path/component-without-helpful-spaces/TELEMETRY.json\n"
	var output bytes.Buffer
	if err := writeModelSummaryView(&output, raw, 32); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
		if width := utf8.RuneCountInString(line); width > 32 {
			t.Fatalf("line width = %d, want <= 32: %q\n%s", width, line, output.String())
		}
	}
	if compactSummaryLayout(output.String()) != compactSummaryLayout(raw) {
		t.Fatalf("long evidence path changed while wrapping:\n%s", output.String())
	}
}

func compactSummaryLayout(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
}
