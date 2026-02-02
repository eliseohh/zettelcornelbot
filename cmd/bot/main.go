package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/eliseohh/zettelcornelbot/internal/bot"
	"github.com/eliseohh/zettelcornelbot/internal/index"
)

func main() {
	fmt.Println("ZettelCornelBot: Sistema Cognitivo Local")

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		fmt.Println("⚠ No TELEGRAM_TOKEN found. Bot will not start.")
	}

	rootDir := os.Getenv("ZETTEL_ROOT")
	if rootDir == "" {
		rootDir = "."
	}

	// 1. Initialize DB
	dbPath := "./zettel.db"
	db, err := index.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	defer db.Close()

	// 2. Apply Schema
	schema, err := index.ReadSchemaFile("internal/index/schema.sql")
	if err != nil {
		log.Fatalf("Cannot load schema: %v", err)
	}
	if err := db.InitSchema(schema); err != nil {
		log.Fatalf("Schema init failed: %v", err)
	}

	// 3. Initial Sync
	fmt.Printf("Syncing %s...\n", rootDir)
	idx := index.NewIndexer(db)
	if err := idx.Sync(rootDir); err != nil {
		log.Printf("⚠ Initial sync failed: %v", err)
	}

	// 4. Start Sync Loop (Every 5 min)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			if err := idx.Sync(rootDir); err != nil {
				log.Printf("Sync error: %v", err)
			}
		}
	}()

	// 5. Start Bot
	if token != "" {
		cfg := bot.Config{
			Token:    token,
			RootDir:  rootDir,
			InboxDir: rootDir,
		}

		b, err := bot.New(cfg, db)
		if err != nil {
			log.Fatalf("Bot init failed: %v", err)
		}

		fmt.Println("🤖 Bot Online. Listening...")
		b.Start()
	} else {
		// Just run as indexer/watcher if no bot
		fmt.Println("Running in Indexer-Only mode.")
		select {}
	}
}
