//go:build !notts

// Package tts — Piper TTS implementation via os/exec.
// No CGo required; uses the precompiled Piper ARM64 binary from rhasspy.
package tts

import (
	"context"
	"encoding/binary"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/tristanj/dusty/internal/config"
)

type piperEngine struct {
	piperBin string
	cfg      config.TTSConfig
}

func newPiperEngine(cfg config.TTSConfig) (Engine, error) {
	bin := cfg.PiperBin
	if bin == "" {
		bin = "piper"
	}
	// Resolve to absolute path; error clearly if the binary is not found.
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf(
			"piper binary %q not found: %w\n"+
				"  → run: make download-piper  (ARM64)\n"+
				"  → or install piper and add it to PATH", bin, err)
	}
	return &piperEngine{piperBin: resolved, cfg: cfg}, nil
}

// Synthesize converts text to raw PCM audio using the Piper TTS binary.
//
// Text is split into individual sentences; each sentence is synthesized by a
// separate Piper invocation. Audio chunks are streamed to the returned channel
// as synthesis proceeds — the first sentence starts playing while the LLM is
// still generating the rest.
//
// The caller must drain the returned channel.
func (e *piperEngine) Synthesize(ctx context.Context, text string, voice VoicePersona) (<-chan []int16, error) {
	ch := make(chan []int16, 8)
	go func() {
		defer close(ch)
		for _, s := range splitSentences(text) {
			if err := e.synthesizeSentence(ctx, s, voice, ch); err != nil {
				return
			}
			// Check for cancellation between sentences.
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	return ch, nil
}

// synthesizeSentence invokes Piper for a single sentence and sends the
// resulting int16 PCM samples to ch.
func (e *piperEngine) synthesizeSentence(ctx context.Context, sentence string, voice VoicePersona, ch chan<- []int16) error {
	args := buildPiperArgs(voice)
	cmd := exec.CommandContext(ctx, e.piperBin, args...)
	cmd.Stdin = strings.NewReader(sentence)

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("piper: %q: %w", truncate(sentence, 40), err)
	}
	if len(out) < 2 {
		return nil // no audio produced (e.g. whitespace-only input)
	}

	samples := rawBytesToInt16(out)
	select {
	case ch <- samples:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// buildPiperArgs constructs the argument list for a Piper invocation.
func buildPiperArgs(voice VoicePersona) []string {
	args := []string{"--model", voice.ModelPath, "--output_raw"}
	if voice.SpeakingRate > 0 && voice.SpeakingRate != 1.0 {
		// Piper's --length_scale is the inverse of speaking_rate:
		//   higher length_scale = longer phoneme duration = slower speech.
		scale := strconv.FormatFloat(1.0/voice.SpeakingRate, 'f', 3, 64)
		args = append(args, "--length_scale", scale)
	}
	return args
}

// rawBytesToInt16 reinterprets a little-endian raw PCM byte slice as int16 samples.
// An odd trailing byte is silently discarded.
func rawBytesToInt16(data []byte) []int16 {
	n := len(data) &^ 1 // round down to even
	samples := make([]int16, n/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(data[2*i:]))
	}
	return samples
}

// sentenceRe matches sentence-ending punctuation runs or newline runs.
var sentenceRe = regexp.MustCompile(`[.!?]+|\n+`)

// splitSentences splits text on sentence boundaries (.!?\n).
// Each returned string includes its trailing punctuation and is trimmed of
// leading/trailing whitespace.
func splitSentences(text string) []string {
	var out []string
	rest := strings.TrimSpace(text)
	for rest != "" {
		loc := sentenceRe.FindStringIndex(rest)
		if loc == nil {
			// No more sentence-ending punctuation; emit the remainder as-is.
			if s := strings.TrimSpace(rest); s != "" {
				out = append(out, s)
			}
			break
		}
		s := strings.TrimSpace(rest[:loc[1]])
		if s != "" {
			out = append(out, s)
		}
		rest = strings.TrimSpace(rest[loc[1]:])
	}
	return out
}

// truncate shortens s to at most n bytes, appending "…" if trimmed.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
