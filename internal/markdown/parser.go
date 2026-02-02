package markdown

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Note struct {
	Path  string
	Title string
	Date  string
	Type  string

	Links []string
}

var (
	reDate = regexp.MustCompile(`^Fecha:\s*(.+)`)
	reType = regexp.MustCompile(`^Tipo:\s*(.+)`)
	// Global link regex
	reLinkGlobal = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
)

const (
	MaxTotalChars   = 4000
	MaxTitleChars   = 120
	MaxNotasChars   = 2800
	MaxResumenChars = 500
	MaxCuesCount    = 7
	MaxCueLen       = 120
)

func ParseFile(path string) (*Note, error) {
	// 1. Read entire file to check global limit efficiently first?
	// Or stream it. Streaming is better but we need total count.
	// Given 4000 chars limit, reading to memory is trivial.
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	totalLen := utf8.RuneCount(contentBytes)
	if totalLen > MaxTotalChars {
		return nil, fmt.Errorf("validation error: total length %d exceeds limit %d", totalLen, MaxTotalChars)
	}

	note := &Note{Path: path}

	// Section Buffers
	var (
		currSection string
		bufNotas    strings.Builder
		bufResumen  strings.Builder
		cues        []string
	)

	scanner := bufio.NewScanner(strings.NewReader(string(contentBytes)))
	lineNum := 0

	for scanner.Scan() {
		lineWithSpace := scanner.Text() // Preserve logic? Scan strips newline usually.
		// utf8.RuneCountInString(line) for checks.
		line := strings.TrimSpace(lineWithSpace)
		lineNum++

		if line == "" {
			continue
		}

		// 1. Title (Must be Line 1 technically, or first non-empty)
		if note.Title == "" {
			if strings.HasPrefix(line, "# ") {
				title := strings.TrimPrefix(line, "# ")
				if utf8.RuneCountInString(title) > MaxTitleChars {
					return nil, fmt.Errorf("validation error: title length %d exceeds limit %d", utf8.RuneCountInString(title), MaxTitleChars)
				}
				note.Title = title
				continue
			}
			// Strict: If first line is not title?
			// Spec says "Línea 1, H1".
			if lineNum == 1 {
				return nil, fmt.Errorf("validation error: first line must be H1 Title")
			}
		}

		// 2. Metadata (Header section)
		if currSection == "" {
			if matches := reDate.FindStringSubmatch(line); len(matches) > 1 {
				note.Date = strings.TrimSpace(matches[1])
				continue
			}
			if matches := reType.FindStringSubmatch(line); len(matches) > 1 {
				note.Type = strings.TrimSpace(matches[1])
				continue
			}
		}

		// 3. Section Switching
		if strings.HasPrefix(line, "## ") {
			currSection = strings.TrimPrefix(line, "## ")
			continue
		}

		// 4. Content Capture & Specific Validation
		switch currSection {
		case "Notas":
			bufNotas.WriteString(lineWithSpace + "\n")
		case "Resumen":
			bufResumen.WriteString(lineWithSpace + "\n")
		case "Cues":
			if strings.HasPrefix(line, "- ") {
				cueText := strings.TrimPrefix(line, "- ")
				cues = append(cues, cueText)
			}
		case "Enlaces":
			// Just extractor logic below
		}

		// 5. Global Link Extraction
		matches := reLinkGlobal.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				target := strings.Split(m[1], "|")[0]
				note.Links = append(note.Links, strings.TrimSpace(target))
			}
		}
	}

	// Post-Scan Validation
	if utf8.RuneCountInString(bufNotas.String()) > MaxNotasChars {
		return nil, fmt.Errorf("validation error: 'Notas' section exceeds %d chars", MaxNotasChars)
	}
	if utf8.RuneCountInString(bufResumen.String()) > MaxResumenChars {
		return nil, fmt.Errorf("validation error: 'Resumen' section exceeds %d chars", MaxResumenChars)
	}

	// Cues Usage Validation
	if len(cues) > MaxCuesCount {
		return nil, fmt.Errorf("validation error: too many cues (%d > %d)", len(cues), MaxCuesCount)
	}
	for i, c := range cues {
		if utf8.RuneCountInString(c) > MaxCueLen {
			return nil, fmt.Errorf("validation error: cue %d length exceeds %d", i+1, MaxCueLen)
		}
		if !strings.HasSuffix(strings.TrimSpace(c), "?") {
			return nil, fmt.Errorf("validation error: cue '%s' must end with '?'", c)
		}
	}

	if note.Title == "" {
		return nil, fmt.Errorf("missing title")
	}

	return note, nil
}
