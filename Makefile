.PHONY: build build-noaudio build-voice build-pi run test test-all test-integration \
        eval eval-compare \
        clean fmt vet lint deps \
        build-whisper download-whisper-model download-piper

BINARY_NAME  = dusty
BUILD_DIR    = bin
GO           = go
WHISPER_DIR  = extern/whisper.cpp
ASSETS_DIR   = assets/models
PIPER_DIR    = bin

# Default config
CONFIG ?= configs/default.toml

# ─── Text REPL (default, no CGo, no hardware) ────────────────────────────────
# Suitable for CI, local development without mic/speaker, and Pi 5 text mode.
build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -tags "noaudio nostt notts" \
		-o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dusty

build-noaudio: build   # alias for clarity

run: build
	$(BUILD_DIR)/$(BINARY_NAME) --config $(CONFIG)

# ─── Full voice build (CGo, requires malgo + libwhisper.a) ───────────────────
# Run `make build-whisper download-whisper-model download-piper` first.
build-voice:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build \
		-o $(BUILD_DIR)/$(BINARY_NAME)-voice ./cmd/dusty

# ─── Pi 5 cross-compile (ARM64 Linux, requires aarch64-linux-gnu-gcc) ────────
# Install toolchain: brew install aarch64-elf-gcc  (macOS)
#                    sudo apt install gcc-aarch64-linux-gnu  (Linux)
build-pi:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 \
		$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/dusty

# ─── Tests ───────────────────────────────────────────────────────────────────
# Default: stub build tags so tests run everywhere without hardware.
test:
	$(GO) test -tags "noaudio nostt notts" ./... -timeout 60s

# Run all tests including any that require hardware (skip in CI).
test-all:
	$(GO) test ./... -timeout 120s

# Run integration tests against a real Ollama instance.
# Requires: ollama serve (running locally).
# Override model: DUSTY_TEST_MODEL=qwen2.5:1.5b make test-integration
test-integration:
	$(GO) test -tags integration -timeout 120s ./internal/agent/...

# ─── Tool Calling Eval ───────────────────────────────────────────────────────
# Measures tool calling success rate for a given model config.
# Requires: ollama serve (running locally with the target model pulled).
#
# Single model eval:
#   make eval CONFIG=configs/tool-test-qwen.toml
#
# Run against all three presets in sequence:
#   make eval-compare
eval:
	$(GO) run ./cmd/eval/ \
		--config $(CONFIG) \
		--cases testdata/eval/tool-calling-cases.yaml \
		--runs 5

# Run eval against all three tool-test presets and print a comparison.
eval-compare:
	@echo "=== Running eval for all three model presets ==="
	@echo ""
	@echo "--- gemma3:1b (baseline) ---"
	-$(GO) run ./cmd/eval/ \
		--config configs/tool-test-gemma.toml \
		--cases testdata/eval/tool-calling-cases.yaml \
		--runs 5 --out eval-report-gemma.json
	@echo ""
	@echo "--- qwen2.5:1.5b ---"
	-$(GO) run ./cmd/eval/ \
		--config configs/tool-test-qwen.toml \
		--cases testdata/eval/tool-calling-cases.yaml \
		--runs 5 --out eval-report-qwen.json
	@echo ""
	@echo "--- llama3.2:1b ---"
	-$(GO) run ./cmd/eval/ \
		--config configs/tool-test-llama.toml \
		--cases testdata/eval/tool-calling-cases.yaml \
		--runs 5 --out eval-report-llama.json
	@echo ""
	@echo "Reports written to eval-report-*.json"

# ─── whisper.cpp build ───────────────────────────────────────────────────────
# Clones and builds libwhisper.a from source. Run once before build-voice.
#
# Requires cmake. Install with:
#   macOS : brew install cmake
#   Linux : sudo apt install cmake   (or sudo dnf install cmake)
build-whisper:
	@command -v cmake >/dev/null 2>&1 || { \
		echo ""; \
		echo "  ERROR: cmake not found."; \
		echo ""; \
		echo "  Install it first:"; \
		echo "    macOS : brew install cmake"; \
		echo "    Linux : sudo apt install cmake"; \
		echo ""; \
		exit 1; \
	}
	@if [ ! -d "$(WHISPER_DIR)" ]; then \
		git clone --depth 1 https://github.com/ggml-org/whisper.cpp $(WHISPER_DIR); \
	fi
	cmake -S $(WHISPER_DIR) -B $(WHISPER_DIR)/build \
		-DBUILD_SHARED_LIBS=OFF \
		-DWHISPER_BUILD_EXAMPLES=OFF \
		-DWHISPER_BUILD_TESTS=OFF
	cmake --build $(WHISPER_DIR)/build --config Release -j4

# Download the Whisper base.en GGML model (~142 MB).
download-whisper-model:
	@mkdir -p $(ASSETS_DIR)
	curl -L \
		https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin \
		-o $(ASSETS_DIR)/ggml-base.en.bin

# Download the Piper TTS ARM64 binary and extract to bin/.
download-piper:
	@mkdir -p $(PIPER_DIR)
	curl -L \
		https://github.com/rhasspy/piper/releases/latest/download/piper_linux_aarch64.tar.gz \
		| tar -xz -C $(PIPER_DIR)/

# ─── Development helpers ─────────────────────────────────────────────────────
clean:
	rm -rf $(BUILD_DIR)

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet -tags "noaudio nostt notts" ./...

lint: fmt vet

deps:
	$(GO) mod tidy
	$(GO) mod download
