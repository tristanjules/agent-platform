.PHONY: build build-noaudio build-voice build-pi run test test-all clean \
        fmt vet lint deps \
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
