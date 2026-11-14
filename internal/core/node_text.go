package core

import (
	"math"
	"strings"
	"unicode"
)

const nodeTitleLineHeight = 24.0

// Conservative advances for the 16px font used by NodeMarkup. Measuring CJK
// and ASCII separately avoids treating a long Chinese title like narrow Latin
// text; long unbroken identifiers must also be allowed to wrap.
func titleAdvance(r rune) float64 {
	switch {
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), r == '\u200d':
		return 0
	case r == ' ' || r == '\t':
		return 6
	case strings.ContainsRune("ilI.,:;!|'`", r):
		return 6.4
	case strings.ContainsRune("MWmw@%", r):
		return 17
	case r < 128:
		return 10.4
	default:
		return 16.8
	}
}

func titleMeasuredWidth(text string) float64 {
	maximum, line := 0.0, 0.0
	for _, r := range text {
		if r == '\n' {
			maximum = math.Max(maximum, line)
			line = 0
		} else if r != '\r' {
			line += titleAdvance(r)
		}
	}
	return math.Max(maximum, line)
}

func wrapNodeTitle(text string, available float64) []string {
	lines := []string{}
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		runes := []rune(paragraph)
		if len(runes) == 0 {
			lines = append(lines, "")
			continue
		}
		for start := 0; start < len(runes); {
			end, lastSpace, width := start, -1, 0.0
			for end < len(runes) {
				advance := titleAdvance(runes[end])
				if end > start && width+advance > available {
					break
				}
				width += advance
				if unicode.IsSpace(runes[end]) {
					lastSpace = end
				}
				end++
			}
			next := end
			if end < len(runes) && lastSpace > start {
				end, next = lastSpace, lastSpace+1
				for next < len(runes) && unicode.IsSpace(runes[next]) {
					next++
				}
			}
			lines = append(lines, string(runes[start:end]))
			start = next
		}
	}
	return lines
}

func boxTitleSize(title string, database bool) (float64, float64) {
	width := math.Max(220, math.Min(360, math.Ceil(titleMeasuredWidth(title)+36)))
	lines := wrapNodeTitle(title, width-36)
	padding := 24.0
	if database {
		// The cap ends at y=29.5; the lower side ends at height-16.
		padding = 50
	}
	return width, math.Max(74, float64(len(lines))*nodeTitleLineHeight+padding)
}

func nodeTitleCenter(n Node) float64 {
	if n.Shape == "database" {
		return (30 + n.Height - 16) / 2
	}
	return n.Height / 2
}
