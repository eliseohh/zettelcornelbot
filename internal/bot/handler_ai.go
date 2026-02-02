package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/eliseohenriquez/zettelcornelbot/internal/neural"
	tele "gopkg.in/telebot.v3"
)

// Add Neural Client to Bot struct in handler.go (Separate update needed)
// For now, extending Bot via strict file separation might require modifying the struct definition.
// We will assume Bot struct is updated or we pass it here.

func (b *Bot) registerAI() {
	b.api.Handle("/ai", b.handleAI)
}

func (b *Bot) handleAI(c tele.Context) error {
	payload := c.Message().Payload
	args := strings.Fields(payload)
	if len(args) < 1 {
		return c.Send("Usage: /ai [summarize|cues|draft] ...")
	}

	action := strings.ToLower(args[0])

	// Check if Ollama is enabled/configured?
	// We can init a client on the fly or better store in Bot.
	// Assuming simple instantiation for now or centralized.
	ai := neural.NewClient(os.Getenv("OLLAMA_URL"), os.Getenv("OLLAMA_MODEL"))

	switch action {
	case "summarize":
		if len(args) < 2 {
			return c.Send("Usage: /ai summarize <ID>")
		}
		id := args[1]
		return b.aiSummarize(c, ai, id)

	case "cues":
		if len(args) < 2 {
			return c.Send("Usage: /ai cues <ID>")
		}
		id := args[1]
		return b.aiCues(c, ai, id)

	case "draft":
		if len(args) < 2 {
			return c.Send("Usage: /ai draft <Topic...>")
		}
		topic := strings.Join(args[1:], " ")
		return b.aiDraft(c, ai, topic)

	default:
		return c.Send("Unknown AI command. Permitted: summarize, cues, draft")
	}
}

func (b *Bot) aiSummarize(c tele.Context, ai *neural.Client, id string) error {
	path, err := b.resolvePath(id) // Reusing existing helper
	if err != nil {
		return c.Send(fmt.Sprintf("🔍 Not Found: %s", id))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return c.Send("Read Error")
	}

	c.Send("🧠 Thinking...")
	summary, err := ai.Summarize(string(content))
	if err != nil {
		return c.Send(fmt.Sprintf("AI Error: %v", err))
	}

	return c.Send(fmt.Sprintf("📝 **Summary Suggestion**:\n\n%s", summary), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

func (b *Bot) aiCues(c tele.Context, ai *neural.Client, id string) error {
	path, err := b.resolvePath(id)
	if err != nil {
		return c.Send(fmt.Sprintf("🔍 Not Found: %s", id))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return c.Send("Read Error")
	}

	c.Send("🧠 Thinking...")
	cues, err := ai.SuggestCues(string(content))
	if err != nil {
		return c.Send(fmt.Sprintf("AI Error: %v", err))
	}

	return c.Send(fmt.Sprintf("❓ **Cue Suggestions**:\n\n%s\n\n_Use /cue add <id> <text> to apply_", cues), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

func (b *Bot) aiDraft(c tele.Context, ai *neural.Client, topic string) error {
	c.Send("🧠 Drafting...")
	draft, err := ai.Draft(topic)
	if err != nil {
		return c.Send(fmt.Sprintf("AI Error: %v", err))
	}

	// Send as code block for easy copy
	return c.Send(fmt.Sprintf("📄 **Draft Generated**:\n```markdown\n%s\n```\n_Copy and use /note create to start._", draft), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}
