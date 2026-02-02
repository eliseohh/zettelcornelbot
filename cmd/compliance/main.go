package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	errors := 0
	fmt.Println("🛡️  Iniciando Protocolo de Cumplimiento (Compliance Suite)...")

	// 1. Scan for Forbidden Terms (Stack & Tone)
	forbidden := map[string][]string{
		"Stack Violation": {"github.com/aws/aws-sdk-go", "obsidian", "notion", "firebase", "mongo"},
		"Tone Violation":  {"coach", "empatía", "empathy", "siento", "lo siento", "feel"},
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip binaries and git
		if strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".db") || strings.HasSuffix(path, ".exe") || path == "zettelbot" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sContent := strings.ToLower(string(content))

		for category, terms := range forbidden {
			for _, term := range terms {
				if strings.Contains(sContent, term) {
					// Exception for this file itself (compliance tool) and COMPLIANCE.md
					if strings.Contains(path, "cmd/compliance") || strings.Contains(path, "COMPLIANCE.md") {
						continue
					}
					fmt.Printf("❌ [Violation] %s found in %s ('%s')\n", category, path, term)
					errors++
				}
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	// 2. Verify Bot Strictness
	handlerPath := "internal/bot/handler.go"
	if content, err := os.ReadFile(handlerPath); err == nil {
		sContent := string(content)
		if !strings.Contains(sContent, "Texto libre prohibido") {
			fmt.Printf("❌ [Bot] Handler does not seem to strictly reject free text.\n")
			errors++
		}
		if !strings.Contains(sContent, "/note") || !strings.Contains(sContent, "/cue") {
			fmt.Printf("❌ [Bot] Missing required atomic commands.\n")
			errors++
		}
	} else {
		fmt.Printf("❌ [Bot] handler.go not found.\n")
		errors++
	}

	// 3. Verify Parser Limits (Source of Truth)
	parserPath := "internal/markdown/parser.go"
	if content, err := os.ReadFile(parserPath); err == nil {
		sContent := string(content)
		checks := map[string]string{
			"MaxTotalChars   = 4000": "Total Limit 4000",
			"MaxTitleChars   = 120":  "Title Limit 120",
			"MaxNotasChars   = 2800": "Notas Limit 2800",
			"MaxResumenChars = 500":  "Resumen Limit 500",
			"MaxCuesCount    = 7":    "Cues Count 7",
			"MaxCueLen       = 120":  "Cue Len 120",
		}
		for sig, name := range checks {
			if !strings.Contains(sContent, sig) {
				fmt.Printf("❌ [Spec] %s constraint NOT found in parser.go\n", name)
				errors++
			}
		}
	} else {
		fmt.Printf("❌ [Spec] parser.go not found.\n")
		errors++
	}

	if errors > 0 {
		fmt.Printf("\n🚫 SE ENCONTRARON %d VIOLACIONES DE CUMPLIMIENTO.\n", errors)
		os.Exit(1)
	}

	fmt.Println("✅ CUMPLIMIENTO VERIFICADO: 100% (Binary Pass)")
}
