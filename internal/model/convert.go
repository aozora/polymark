package model

import (
	"strings"
	"unicode"
)

// Unsupported reports Markdown constructs this milestone will not import
// into the rich-text editor. Pipe tables are deferred: MyGo's parser turns
// their cells into tab-separated paragraphs, and exporting that text does
// not restore the table. HTML tables are the same kind of loss.
func Unsupported(markdown string) []string {
	var reasons []string
	if hasPipeTable(markdown) {
		reasons = append(reasons, "pipe table")
	}
	if strings.Contains(strings.ToLower(markdown), "<table") {
		reasons = append(reasons, "HTML table")
	}
	return reasons
}

func hasPipeTable(markdown string) bool {
	lines := strings.Split(markdown, "\n")
	for i := 0; i < len(lines)-1; i++ {
		if isTableRow(lines[i]) && isTableSeparator(lines[i+1]) {
			return true
		}
	}
	return false
}

func isTableRow(line string) bool {
	s := strings.TrimSpace(line)
	if !strings.Contains(s, "|") {
		return false
	}
	n := strings.Count(s, "|")
	return n >= 2
}

func isTableSeparator(line string) bool {
	s := strings.TrimSpace(line)
	if s == "" || !strings.Contains(s, "|") {
		return false
	}
	dashes := 0
	for _, r := range s {
		switch r {
		case '|', ':', '-', ' ':
			if r == '-' {
				dashes++
			}
		default:
			if !unicode.IsSpace(r) {
				return false
			}
		}
	}
	return dashes >= 3
}
