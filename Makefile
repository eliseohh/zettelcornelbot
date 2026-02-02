.PHONY: all check test compliance services build run clean help

BINARY_NAME=zettelbot
OLLAMA_URL=http://localhost:11434

help:
	@echo "ZettelCornelBot Management"
	@echo "=========================="
	@echo "make run       : Full start (Services -> Checks -> Build -> Run)"
	@echo "make check     : Run all tests and compliance suites"
	@echo "make services  : Check external dependencies (Ollama, Env)"
	@echo "make clean     : Remove artifacts"

all: check build

# --- 1. Service Verification ---
services:
	@echo "🔍 Checking Services..."
	@# 1. Telegram Token
	@if [ -z "$(TELEGRAM_TOKEN)" ]; then \
		echo "⚠️  WARNING: TELEGRAM_TOKEN not set. Bot will run in Index-Only mode."; \
	else \
		echo "✅ Telegram Token found."; \
	fi
	@# 2. Ollama Connectivity
	@if curl -s --head  --request GET $(OLLAMA_URL) | grep "200 OK" > /dev/null; then \
		echo "✅ Ollama is RUNNING at $(OLLAMA_URL)"; \
	else \
		echo "⚠️  WARNING: Ollama not reachable at $(OLLAMA_URL). AI features will fail."; \
	fi

# --- 2. Code Compliance & Integrity ---
compliance:
	@echo "🛡️  Running Compliance Suite..."
	@go run cmd/compliance/main.go

# --- 3. Testing ---
test: compliance
	@echo "🧪 Running Test Suite..."
	@go test -v ./internal/...
	@# Legacy verification scripts
	@go run cmd/test_parser/main.go
	@go run cmd/test_indexer/main.go
	@go run cmd/test_ops/main.go
	@go run cmd/test_ai/main.go

# --- 4. Build ---
build: test services
	@echo "🔨 Building Binary..."
	@go build -o $(BINARY_NAME) cmd/bot/main.go

# --- 5. Execution ---
run: build
	@echo "🚀 Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

clean:
	@echo "🧹 Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f test_*.md
	@rm -rf test_ops
	@rm -rf internal/index/test_vault
