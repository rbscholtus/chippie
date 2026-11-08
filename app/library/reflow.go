package library

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// wrappedLength is the length from which a line counts as hard-wrapped
// prose: many descriptions come from old text files wrapped at 70–85
// columns.
const wrappedLength = 60

var (
	underline = regexp.MustCompile(`^[-=_*~]{3,}$`)
	columns   = regexp.MustCompile(`\S {2,}\S|\t`)
	listItem  = regexp.MustCompile(`^([-*•]|\d+[.)]|"[^"]*"(\s*,\s*"[^"]*")*\s*[-–:])\s`)
)

// Preformatted reports whether a description line has its own layout that
// must be kept: a table row (columns separated by two or more spaces), an
// indented line, or a heading underline. The UI draws such lines unwrapped,
// in a monospaced font.
func Preformatted(line string) bool {
	s := strings.TrimSpace(line)
	if s == "" {
		return false
	}

	return underline.MatchString(s) || columns.MatchString(s) || strings.HasPrefix(line, "  ")
}

// structural lines are never joined to their neighbours.
func structural(line string) bool {
	s := strings.TrimSpace(line)

	return s == "" || Preformatted(line) || listItem.MatchString(s)
}

// Reflow joins the hard-wrapped lines of a description into paragraphs, so
// the UI can wrap them to its own width. A line is joined to the next only
// when it is long (wrapped prose) and neither line is blank, a list item, a
// table row, an indented line or an underline: short lines and layout are
// kept as the author wrote them.
func Reflow(desc string) string {
	lines := strings.Split(strings.ReplaceAll(desc, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		if n := len(out); n > 0 {
			prev := strings.TrimRight(out[n-1], " ")
			if utf8.RuneCountInString(prev) >= wrappedLength && !structural(prev) && !structural(line) {
				out[n-1] = prev + " " + strings.TrimSpace(line)

				continue
			}
		}

		out = append(out, strings.TrimRight(line, " "))
	}

	return strings.Join(out, "\n")
}
