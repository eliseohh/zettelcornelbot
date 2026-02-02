package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/eliseohenriquez/zettelcornelbot/internal/markdown"
)

func main() {
	// Case 1: Valid
	validContent := `# Title Valid
Fecha: 2024-02-02
Tipo: idea

## Notas
Content.

## Cues
- What is life?
- Is this valid?

## Resumen
Short summary.

## Enlaces
- [[link]]
`
	testParse("valid", validContent, true)

	// Case 2: Invalid Cue
	invalidCue := `# Title
Fecha: 2024-02-02
Tipo: idea

## Cues
- Missing question mark
`
	testParse("invalid_cue", invalidCue, false)

	// Case 3: Title too long
	longTitle := "# " + strings.Repeat("A", 121) + "\nFecha: 2024\n"
	testParse("long_title", longTitle, false)

	fmt.Println("✔ ALL Constraints Tests Passed")
}

func testParse(name, content string, expectSuccess bool) {
	filename := "test_" + name + ".md"
	os.WriteFile(filename, []byte(content), 0644)
	defer os.Remove(filename)

	_, err := markdown.ParseFile(filename)
	if expectSuccess && err != nil {
		fmt.Printf("❌ %s failed unexpectedly: %v\n", name, err)
		os.Exit(1)
	}
	if !expectSuccess && err == nil {
		fmt.Printf("❌ %s succeeded unexpectedly (expected failure)\n", name)
		os.Exit(1)
	}
	if !expectSuccess {
		fmt.Printf("✔ %s rejected as expected: %v\n", name, err)
	}
}
