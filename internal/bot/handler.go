package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/eliseohh/zettelcornelbot/internal/index"
	"github.com/eliseohh/zettelcornelbot/internal/markdown"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	api *tele.Bot
	db  *index.DB
	cfg Config
}

type Config struct {
	Token    string
	RootDir  string
	InboxDir string
}

func New(cfg Config, db *index.DB) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	bot := &Bot{api: b, db: db, cfg: cfg}
	bot.register()
	return bot, nil
}

func (b *Bot) Start() {
	fmt.Printf("Bot started: %s\n", b.api.Me.Username)
	b.api.Start()
}

func (b *Bot) register() {
	// Root Commands
	b.api.Handle("/note", b.handleNote)
	b.api.Handle("/cue", b.handleCue)

	// Legacy/Utility (kept for status check)
	b.api.Handle("/status", b.handleStatus)

	// Catch-all for Text to Strict Reject
	b.api.Handle(tele.OnText, func(c tele.Context) error {
		// Checks...
		return c.Send("⛔ Error: Texto libre prohibido. Use comandos atómicos.")
	})

	// AI
	b.registerAI()
}

// /note router
func (b *Bot) handleNote(c tele.Context) error {
	payload := c.Message().Payload // "create Title...", "validate ID", "link A B"
	args := strings.Fields(payload)

	if len(args) < 1 {
		return c.Send("Usage: /note [create|validate|link] ...")
	}

	action := strings.ToLower(args[0])
	switch action {
	case "create":
		// /note create <Title...>
		if len(args) < 2 {
			return c.Send("Usage: /note create <Title>")
		}
		title := strings.Join(args[1:], " ")
		return b.noteCreate(c, title)

	case "validate":
		// /note validate <ID>
		if len(args) < 2 {
			return c.Send("Usage: /note validate <ID>")
		}
		return b.noteValidate(c, args[1])

	case "link":
		// /note link <ID1> <ID2>
		if len(args) < 3 {
			return c.Send("Usage: /note link <SourceID> <TargetID>")
		}
		return b.noteLink(c, args[1], args[2])

	default:
		return c.Send(fmt.Sprintf("Unknown action: %s", action))
	}
}

// /cue router
func (b *Bot) handleCue(c tele.Context) error {
	payload := c.Message().Payload
	// "add ID Question?"
	// Need to split carefully. ID is mostly 1 word (derived from filename? usually no spaces in filename or kebab-case).
	// Assuming ID doesn't contain spaces.

	parts := strings.SplitN(payload, " ", 3) // "add", "ID", "Rest..."
	if len(parts) < 3 {
		return c.Send("Usage: /cue add <ID> <Question?>")
	}

	action := strings.ToLower(parts[0])
	id := parts[1]
	question := parts[2]

	if action != "add" {
		return c.Send("Usage: /cue add ...")
	}

	return b.cueAdd(c, id, question)
}

// -- Implementations --

func (b *Bot) noteCreate(c tele.Context, title string) error {
	// Generate Filename: YYYYMMDD-kebab-title.md
	dateStr := time.Now().Format("20060102")
	kebab := toKebab(title)
	filename := fmt.Sprintf("%s-%s.md", dateStr, kebab)
	path := filepath.Join(b.cfg.RootDir, filename)

	// Ensure atomic: Check if exists
	if _, err := os.Stat(path); err == nil {
		return c.Send("⛔ Error: Note already exists.")
	}

	content := fmt.Sprintf(`# %s
Fecha: %s
Tipo: idea

## Notas


## Cues


## Resumen


## Enlaces

`, title, time.Now().Format("2006-01-02"))

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return c.Send(fmt.Sprintf("FS Error: %v", err))
	}

	return c.Send(fmt.Sprintf("✅ Created: `%s`", filename))
}

func (b *Bot) noteValidate(c tele.Context, id string) error {
	path, err := b.resolvePath(id)
	if err != nil {
		return c.Send(fmt.Sprintf("🔍 Not Found: %s", id))
	}

	_, err = markdown.ParseFile(path)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Invalid: %v", err))
	}
	return c.Send(fmt.Sprintf("✅ Valid: `%s`", id))
}

func (b *Bot) noteLink(c tele.Context, srcID, tgtID string) error {
	srcPath, err := b.resolvePath(srcID)
	if err != nil {
		return c.Send(fmt.Sprintf("🔍 Not Found Source: %s", srcID))
	}

	// Atomic Append to "## Enlaces" section
	// Strategy: Read file, find line "## Enlaces", append "- [[tgtID]]" after it.
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	foundSection := false
	inserted := false

	for _, line := range lines {
		newLines = append(newLines, line)
		if strings.TrimSpace(line) == "## Enlaces" {
			foundSection = true
			if !inserted {
				newLines = append(newLines, fmt.Sprintf("- [[%s]]", tgtID))
				inserted = true
			}
		}
	}

	if !foundSection {
		// Section missing? Strict format says it must exist.
		// If missing, append it? Or error?
		// Verification check says validation enforces it. So it should exist if valid.
		return c.Send("⛔ Error: missing '## Enlaces' section in source.")
	}

	if err := os.WriteFile(srcPath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return nil // err
	}

	return c.Send(fmt.Sprintf("🔗 Linked: %s -> %s", srcID, tgtID))
}

func (b *Bot) cueAdd(c tele.Context, id, question string) error {
	// Validation first
	if !strings.HasSuffix(strings.TrimSpace(question), "?") {
		return c.Send("⛔ Error: Cue must end with '?'")
	}

	path, err := b.resolvePath(id)
	if err != nil {
		return c.Send(fmt.Sprintf("🔍 Not Found: %s", id))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	found := false
	inserted := false

	for _, line := range lines {
		newLines = append(newLines, line)
		if strings.TrimSpace(line) == "## Cues" {
			found = true
			if !inserted {
				newLines = append(newLines, fmt.Sprintf("- %s", question))
				inserted = true
			}
		}
	}

	if !found {
		return c.Send("⛔ Error: missing '## Cues' section.")
	}

	if err := os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return c.Send("Write Error")
	}

	return c.Send("✅ Cue Added")
}

// Helpers

func (b *Bot) handleStatus(c tele.Context) error {
	// Keep status purely for DB stats
	var count int
	b.db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&count)
	return c.Send(fmt.Sprintf("Nodes: %d", count))
}

func (b *Bot) resolvePath(id string) (string, error) {
	// ID is filename without extension?
	// Try implicit path
	path := filepath.Join(b.cfg.RootDir, id+".md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Try explicit DB lookup if we tracked ID vs Path?
	// Our indexer derives ID from filename.
	// So strict mapping: ID == Basename.
	return "", fmt.Errorf("not found")
}

func toKebab(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile("[^a-z0-9]+")
	return strings.Trim(reg.ReplaceAllString(s, "-"), "-")
}
