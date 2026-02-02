package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eliseohh/zettelcornelbot/internal/index"
	tele "gopkg.in/telebot.v3"
)

// MockContext definition for internal use
type MockContext struct {
	tele.Context
	PayloadVal string
	SentMsg    interface{}
}

func (m *MockContext) Message() *tele.Message {
	return &tele.Message{Payload: m.PayloadVal}
}
func (m *MockContext) Send(what interface{}, opts ...interface{}) error {
	m.SentMsg = what
	return nil
}

func TestBotHandlers(t *testing.T) {
	// Setup FS
	tmpDir, _ := os.MkdirTemp("", "bot_test")
	defer os.RemoveAll(tmpDir)

	// Setup DB
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := index.NewDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create Bot Instance
	cfg := Config{RootDir: tmpDir}
	b := &Bot{db: db, cfg: cfg}

	// Test 1: Note Create
	t.Run("Note Create Success", func(t *testing.T) {
		ctx := &MockContext{PayloadVal: "create Test Note"}
		if err := b.handleNote(ctx); err != nil {
			t.Fatal(err)
		}

		msg := ctx.SentMsg.(string)
		if !strings.Contains(msg, "✅ Created") {
			t.Errorf("Expected success msg, got: %s", msg)
		}

		// Verify file exists
		date := time.Now().Format("20060102")
		expectedPath := filepath.Join(tmpDir, date+"-test-note.md")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Error("File not created")
		}
	})

	// Test 1.1: Note Create with Folder (Libro)
	t.Run("Note Create Folder", func(t *testing.T) {
		ctx := &MockContext{PayloadVal: "create libro My Book"}
		if err := b.handleNote(ctx); err != nil {
			t.Fatal(err)
		}

		msg := ctx.SentMsg.(string)
		if !strings.Contains(msg, "libro/") {
			t.Errorf("Expected folder path, got: %s", msg)
		}

		// Verify file exists in subfolder
		date := time.Now().Format("20060102")
		expectedPath := filepath.Join(tmpDir, "libro", date+"-my-book.md")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Error("File not created in subfolder")
		}
	})

	// Test 2: Cue Add Strictness
	t.Run("Cue Add Invalid", func(t *testing.T) {
		// Needs an existing note ID first. from prev test: date-test-note
		date := time.Now().Format("20060102")
		id := date + "-test-note"

		ctx := &MockContext{PayloadVal: "add " + id + " Invalid Cue"}
		b.handleCue(ctx)

		msg := ctx.SentMsg.(string)
		if !strings.Contains(msg, "must end with '?'") {
			t.Errorf("Strict cues check failed, got: %s", msg)
		}
	})

	t.Run("Cue Add Valid", func(t *testing.T) {
		date := time.Now().Format("20060102")
		id := date + "-test-note"

		ctx := &MockContext{PayloadVal: "add " + id + " Valid Cue?"}
		if err := b.handleCue(ctx); err != nil {
			t.Error(err)
		}

		msg := ctx.SentMsg.(string)
		if !strings.Contains(msg, "✅ Cue Added") {
			t.Errorf("Expected success, got: %s", msg)
		}
	})

	// Test 3: Note Link
	t.Run("Note Link", func(t *testing.T) {
		date := time.Now().Format("20060102")
		src := date + "-test-note"
		tgt := "some-other-id"

		ctx := &MockContext{PayloadVal: "link " + src + " " + tgt}
		if err := b.handleNote(ctx); err != nil {
			t.Error(err)
			// Test 4: Status Tree
			t.Run("Status Tree", func(t *testing.T) {
				ctx := &MockContext{PayloadVal: ""}
				if err := b.handleStatus(ctx); err != nil {
					t.Error(err)
				}

				msg := ctx.SentMsg.(string)
				if !strings.Contains(msg, "🌳 **Zettelkasten Status**") {
					t.Errorf("Expected tree header, got: %s", msg)
				}
			})
		}

		msg := ctx.SentMsg.(string)
		if !strings.Contains(msg, "🔗 Linked") {
			t.Errorf("Expected linked msg, got: %s", msg)
		}
	})
}
