#!/bin/bash
set -e

echo "🔍 Starting System Audit..."

echo "1. Checking Compliance (Static Analysis)..."
go run cmd/compliance/main.go

echo "2. Checking Dependencies..."
go mod tidy

echo "2. Testing Parser..."
go run cmd/test_parser/main.go

echo "3. Testing Indexer..."
go get github.com/mattn/go-sqlite3 # ensure driver
go run cmd/test_indexer/main.go

echo "4. Testing Bot Ops..."
go run cmd/test_ops/main.go
go test -v ./internal/bot/...

echo "5. Testing AI Permissions..."
go run cmd/test_ai/main.go

echo "6. Building Binary..."
go build -o zettelbot cmd/bot/main.go

echo "✅ System Verified. Ready for deployment."
