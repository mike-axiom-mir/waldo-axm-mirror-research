// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	waldoai "github.com/openwaldo/waldo/internal/ai"
	"github.com/openwaldo/waldo/internal/model"
	"github.com/openwaldo/waldo/internal/training"
)

func TestAdvisorCheckpointMonitorUsesEvidenceCapturedAtObservation(t *testing.T) {
	root := t.TempDir()
	const name = "checkpoint-snapshot"
	compose := advisorTestCompose()
	if _, err := (model.Builder{Root: root}).Initialize(name, compose.Architecture); err != nil {
		t.Fatal(err)
	}

	transcript := &advisorTranscript{root: root, name: name, session: "test-session"}
	var output bytes.Buffer
	var warnings bytes.Buffer
	monitor := &advisorCheckpointMonitor{
		ctx: context.Background(), root: root, name: name,
		transcript: transcript, output: &output, warnings: &warnings,
		events: make(chan model.Progress, 1), done: make(chan struct{}),
	}

	previousAsk := modelAdvisorAsk
	modelAdvisorAsk = func(context.Context, waldoai.Selection, string) (string, error) {
		return `{"reply":"checkpoint snapshot survived later model changes"}`, nil
	}
	t.Cleanup(func() { modelAdvisorAsk = previousAsk })

	event := model.Progress{
		Phase: "training", RunID: "run-1", State: model.RunRunning,
		Training: &training.Event{Kind: "checkpoint", Step: 1, Tokens: 64, Message: "checkpoint persisted"},
	}
	monitor.Observe(event)

	// The build is allowed to advance as soon as Observe returns. Make that
	// advancement destructive so a monitor that rereads model state later has
	// no checkpoint-time evidence left to inspect.
	if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
		t.Fatal(err)
	}
	close(monitor.events)
	monitor.run()

	if !strings.Contains(output.String(), "checkpoint snapshot survived later model changes") {
		t.Fatalf("monitor output = %q, warnings = %q", output.String(), warnings.String())
	}
}
