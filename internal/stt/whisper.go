//go:build !nostt

// whisper.go -- whisper.cpp STT engine via CGo.
//
// Prerequisites before building (run once from the repo root):
//
//	make build-whisper           # builds extern/whisper.cpp/build/...
//	make download-whisper-model  # downloads assets/models/ggml-base.en.bin
//	make build-voice             # compiles with CGo enabled
//
// NOTE: go analysis tools (gopls, go vet) will report "whisper.h not found"
// until make build-whisper has been run. This is expected.
package stt

/*
#cgo CFLAGS: -I${SRCDIR}/../../extern/whisper.cpp/include
#cgo CFLAGS: -I${SRCDIR}/../../extern/whisper.cpp/ggml/include

#cgo LDFLAGS: -L${SRCDIR}/../../extern/whisper.cpp/build/src              -lwhisper
#cgo LDFLAGS: -L${SRCDIR}/../../extern/whisper.cpp/build/ggml/src         -lggml
#cgo LDFLAGS: -L${SRCDIR}/../../extern/whisper.cpp/build/ggml/src         -lggml-base
#cgo LDFLAGS: -L${SRCDIR}/../../extern/whisper.cpp/build/ggml/src         -lggml-cpu
#cgo LDFLAGS: -L${SRCDIR}/../../extern/whisper.cpp/build/ggml/src/ggml-blas  -lggml-blas
#cgo LDFLAGS: -L${SRCDIR}/../../extern/whisper.cpp/build/ggml/src/ggml-metal -lggml-metal

#cgo darwin LDFLAGS: -framework Accelerate -framework Foundation -framework Metal -framework MetalKit
#cgo linux  LDFLAGS: -ldl -lpthread

#cgo LDFLAGS: -lstdc++ -lm

#include "whisper.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/tristanj/dusty/internal/config"
)

type whisperEngine struct {
	wctx *C.struct_whisper_context
	cfg  config.STTConfig
}

// newWhisperEngine loads a GGML model and returns a ready-to-use Engine.
// cfg.Model may be a short name ("base", "tiny") or an explicit file path.
func newWhisperEngine(cfg config.STTConfig) (Engine, error) {
	path := resolveModelPath(cfg.Model)

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// Use the non-deprecated with_params variant.
	cparams := C.whisper_context_default_params()
	wctx := C.whisper_init_from_file_with_params(cPath, cparams)
	if wctx == nil {
		return nil, fmt.Errorf("whisper: failed to load model %q\n"+
			"  -> run: make download-whisper-model", path)
	}
	return &whisperEngine{wctx: wctx, cfg: cfg}, nil
}

// Transcribe runs whisper.cpp full-segment transcription on the provided PCM
// samples (16 kHz mono float32).
func (w *whisperEngine) Transcribe(ctx context.Context, pcm []float32) (STTResult, error) {
	if err := ctx.Err(); err != nil {
		return STTResult{}, err
	}
	if len(pcm) == 0 {
		return STTResult{}, nil
	}

	start := time.Now()

	params := C.whisper_full_default_params(C.WHISPER_SAMPLING_GREEDY)
	// Suppress whisper's built-in stderr logging to keep the REPL output clean.
	params.print_progress = C.bool(false)
	params.print_realtime = C.bool(false)
	params.print_timestamps = C.bool(false)
	params.print_special = C.bool(false)
	params.translate = C.bool(false)
	params.single_segment = C.bool(false)

	lang := w.cfg.Language
	if lang == "" {
		lang = "en"
	}
	cLang := C.CString(lang)
	defer C.free(unsafe.Pointer(cLang))
	params.language = cLang

	ret := C.whisper_full(
		w.wctx,
		params,
		(*C.float)(unsafe.Pointer(&pcm[0])),
		C.int(len(pcm)),
	)
	if ret != 0 {
		return STTResult{}, fmt.Errorf("whisper: transcription failed (code %d)", int(ret))
	}

	var sb strings.Builder
	n := int(C.whisper_full_n_segments(w.wctx))
	for i := 0; i < n; i++ {
		sb.WriteString(C.GoString(C.whisper_full_get_segment_text(w.wctx, C.int(i))))
	}

	return STTResult{
		Text:     strings.TrimSpace(sb.String()),
		Language: lang,
		Duration: time.Since(start),
	}, nil
}

// Close frees the whisper context and unloads the model.
func (w *whisperEngine) Close() error {
	if w.wctx != nil {
		C.whisper_free(w.wctx)
		w.wctx = nil
	}
	return nil
}

// resolveModelPath maps a short name like "base" or "tiny" to the GGML bin
// path downloaded by `make download-whisper-model`.
// Strings already containing a path separator are returned unchanged.
func resolveModelPath(model string) string {
	if strings.ContainsAny(model, "/\\") {
		return model
	}
	name := model
	if name == "" {
		name = "base"
	}
	return fmt.Sprintf("assets/models/ggml-%s.en.bin", name)
}
