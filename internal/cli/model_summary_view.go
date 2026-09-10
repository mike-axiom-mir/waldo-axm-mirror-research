// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// runModelSummaryView keeps the existing model summary as the source of truth
// and adapts only its human realization to the available terminal width.
// Structured JSON bypasses this layer completely.
func runModelSummaryView(context Context, args []string, stdout, stderr io.Writer) error {
	if context.JSON {
		return runModelSummary(context, args, stdout, stderr)
	}

	var canonical bytes.Buffer
	if err := runModelSummary(context, args, &canonical, stderr); err != nil {
		return err
	}
	return writeModelSummaryView(stdout, canonical.String(), modelListOutputWidth())
}

func writeModelSummaryView(output io.Writer, summary string, width int) error {
	width = clampModelListWidth(width)
	if modelSummaryFits(summary, width) {
		_, err := io.WriteString(output, summary)
		return err
	}

	trailingNewline := strings.HasSuffix(summary, "\n")
	body := summary
	if trailingNewline {
		body = strings.TrimSuffix(body, "\n")
	}

	var rendered bytes.Buffer
	for _, line := range strings.Split(body, "\n") {
		if err := writeModelSummaryLine(&rendered, line, width); err != nil {
			return err
		}
	}
	value := rendered.String()
	if !trailingNewline {
		value = strings.TrimSuffix(value, "\n")
	}
	_, err := io.WriteString(output, value)
	return err
}

func modelSummaryFits(summary string, width int) bool {
	for _, line := range strings.Split(strings.TrimSuffix(summary, "\n"), "\n") {
		if utf8.RuneCountInString(line) > width {
			return false
		}
	}
	return true
}

func writeModelSummaryLine(output io.Writer, line string, width int) error {
	if utf8.RuneCountInString(line) <= width {
		_, err := fmt.Fprintln(output, line)
		return err
	}

	prefix, value := modelSummaryField(line)
	prefixWidth := utf8.RuneCountInString(prefix)
	if prefixWidth >= width {
		prefix = ""
		value = line
		prefixWidth = 0
	}
	continuation := strings.Repeat(" ", prefixWidth)
	parts := wrapModelSummaryValue(value, max(1, width-prefixWidth))
	for index, part := range parts {
		linePrefix := continuation
		if index == 0 {
			linePrefix = prefix
		}
		if _, err := fmt.Fprintf(output, "%s%s\n", linePrefix, part); err != nil {
			return err
		}
	}
	return nil
}

func modelSummaryField(line string) (string, string) {
	if strings.HasPrefix(line, "  - ") {
		return "  - ", strings.TrimSpace(strings.TrimPrefix(line, "  - "))
	}
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return "", line
	}
	end := colon + 1
	for end < len(line) && line[end] == ' ' {
		end++
	}
	if end >= len(line) {
		return "", line
	}
	return line[:end], line[end:]
}

func wrapModelSummaryValue(value string, width int) []string {
	normalized := strings.Join(strings.Fields(value), " ")
	if normalized == "" {
		return []string{""}
	}

	remaining := []rune(normalized)
	parts := make([]string, 0, 2)
	for len(remaining) > width {
		take, skip := modelSummaryBreak(remaining, width)
		parts = append(parts, string(remaining[:take]))
		remaining = remaining[take+skip:]
		for len(remaining) > 0 && unicode.IsSpace(remaining[0]) {
			remaining = remaining[1:]
		}
	}
	parts = append(parts, string(remaining))
	return parts
}

func modelSummaryBreak(value []rune, width int) (take, skip int) {
	limit := min(width, len(value))
	for index := limit; index > 0; index-- {
		if unicode.IsSpace(value[index-1]) {
			if index == 1 {
				break
			}
			return index - 1, 1
		}
	}
	for index := limit; index > 0; index-- {
		switch value[index-1] {
		case '/', '\\', '-', '—':
			return index, 0
		}
	}
	return limit, 0
}
