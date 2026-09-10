// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/openwaldo/waldo/internal/model"
	"golang.org/x/term"
)

const (
	minimumModelListWidth = 32
	defaultModelListWidth = 120
	maximumModelListWidth = 240
)

var modelListOutputWidth = func() int {
	if columns, err := strconv.Atoi(strings.TrimSpace(os.Getenv("COLUMNS"))); err == nil {
		return clampModelListWidth(columns)
	}
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return clampModelListWidth(width)
	}
	return defaultModelListWidth
}

func clampModelListWidth(width int) int {
	if width < minimumModelListWidth {
		return minimumModelListWidth
	}
	if width > maximumModelListWidth {
		return maximumModelListWidth
	}
	return width
}

func modelListState(item model.Listing) string {
	if item.State == "" {
		return "untrained"
	}
	return item.State
}

func writeModelList(output io.Writer, models []model.Listing) error {
	width := modelListOutputWidth()
	if modelListTableWidth(models) <= width {
		return writeModelListTable(output, models)
	}
	return writeModelListCards(output, models, width)
}

func writeEmptyModelList(output io.Writer, patterns []string) error {
	width := modelListOutputWidth()
	if len(patterns) == 0 {
		if err := writeModelListField(output, "", "No local models found.", width); err != nil {
			return err
		}
		return writeModelListField(output, "NEXT  ", "waldo model init <name> --preset 10m", width)
	}
	if err := writeModelListField(output, "NO MATCH  ", strings.Join(patterns, ", "), width); err != nil {
		return err
	}
	return writeModelListField(output, "NEXT      ", "waldo model list", width)
}

func modelListTableWidth(models []model.Listing) int {
	widths := []int{len("NAME"), len("STATE"), len("PARAMETERS"), len("RUNS"), len("UPDATED (UTC)")}
	for _, item := range models {
		fields := []string{item.Name, modelListState(item), humanCount(int64(item.Parameters)), humanInteger(int64(item.Runs)), item.Updated}
		for index, field := range fields {
			widths[index] = max(widths[index], utf8.RuneCountInString(field))
		}
	}
	total := 2 * (len(widths) - 1)
	for _, width := range widths {
		total += width
	}
	return total
}

func writeModelListTable(output io.Writer, models []model.Listing) error {
	table := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "NAME\tSTATE\tPARAMETERS\tRUNS\tUPDATED (UTC)")
	for _, item := range models {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", item.Name, modelListState(item), humanCount(int64(item.Parameters)), humanInteger(int64(item.Runs)), item.Updated)
	}
	return table.Flush()
}

func writeModelListCards(output io.Writer, models []model.Listing, width int) error {
	fmt.Fprintf(output, "MODELS  %s\n\n", humanInteger(int64(len(models))))
	for index, item := range models {
		if err := writeModelListField(output, fmt.Sprintf("%d. ", index+1), item.Name, width); err != nil {
			return err
		}
		for _, field := range []struct{ label, value string }{
			{"STATE", modelListState(item)},
			{"PARAMETERS", humanCount(int64(item.Parameters))},
			{"RUNS", humanInteger(int64(item.Runs))},
			{"UPDATED", item.Updated},
		} {
			if err := writeModelListField(output, fmt.Sprintf("   %-11s", field.label), field.value, width); err != nil {
				return err
			}
		}
		if index < len(models)-1 {
			fmt.Fprintln(output)
		}
	}
	fmt.Fprintln(output, "\nNEXT  waldo model summary <name>")
	return nil
}

func writeModelListField(output io.Writer, prefix, value string, width int) error {
	prefixWidth := utf8.RuneCountInString(prefix)
	continuation := strings.Repeat(" ", prefixWidth)
	remaining := []rune(value)
	first := true
	for len(remaining) > 0 || first {
		linePrefix := continuation
		if first {
			linePrefix = prefix
		}
		available := max(1, width-utf8.RuneCountInString(linePrefix))
		take := min(available, len(remaining))
		if _, err := fmt.Fprintf(output, "%s%s\n", linePrefix, string(remaining[:take])); err != nil {
			return err
		}
		remaining = remaining[take:]
		first = false
	}
	return nil
}
