//go:build !notts

package tts

import (
	"strconv"
	"testing"
)

// ---------------------------------------------------------------------------
// splitSentences
// ---------------------------------------------------------------------------

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single sentence",
			input: "Hello there.",
			want:  []string{"Hello there."},
		},
		{
			name:  "two sentences",
			input: "Hello. World.",
			want:  []string{"Hello.", "World."},
		},
		{
			name:  "mixed punctuation",
			input: "Really? Yes! Great.",
			want:  []string{"Really?", "Yes!", "Great."},
		},
		{
			name:  "newline separator",
			input: "Hello\nWorld",
			want:  []string{"Hello", "World"},
		},
		{
			name:  "multiple newlines collapsed",
			input: "Line one\n\nLine two",
			want:  []string{"Line one", "Line two"},
		},
		{
			name:  "surrounding whitespace stripped",
			input: "  Hello.  World.  ",
			want:  []string{"Hello.", "World."},
		},
		{
			name:  "ellipsis treated as single boundary",
			input: "Hmm... interesting.",
			want:  []string{"Hmm...", "interesting."},
		},
		{
			name:  "no punctuation returns whole string",
			input: "Hello world",
			want:  []string{"Hello world"},
		},
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only returns nil",
			input: "   ",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSentences(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitSentences(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitSentences(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitSentences_TypicalLLMResponse(t *testing.T) {
	input := "The universe is vast. It contains billions of galaxies! Are you curious about it?"
	got := splitSentences(input)
	if len(got) != 3 {
		t.Errorf("expected 3 sentences, got %d: %v", len(got), got)
	}
}

// ---------------------------------------------------------------------------
// buildPiperArgs
// ---------------------------------------------------------------------------

func TestBuildPiperArgs_DefaultRate(t *testing.T) {
	voice := VoicePersona{ModelPath: "/models/test.onnx", SpeakingRate: 1.0}
	args := buildPiperArgs(voice)

	wantBase := []string{"--model", "/models/test.onnx", "--output_raw"}
	if len(args) != len(wantBase) {
		t.Fatalf("args = %v, want %v", args, wantBase)
	}
	for i, a := range args {
		if a != wantBase[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, wantBase[i])
		}
	}
}

func TestBuildPiperArgs_SlowRate(t *testing.T) {
	voice := VoicePersona{ModelPath: "/models/test.onnx", SpeakingRate: 0.9}
	args := buildPiperArgs(voice)

	// Expect --length_scale to be present.
	found := false
	for i, a := range args {
		if a == "--length_scale" && i+1 < len(args) {
			found = true
			// 1.0/0.9 = 1.111...
			want := strconv.FormatFloat(1.0/0.9, 'f', 3, 64)
			if args[i+1] != want {
				t.Errorf("--length_scale = %q, want %q", args[i+1], want)
			}
		}
	}
	if !found {
		t.Error("expected --length_scale argument for speaking rate 0.9")
	}
}

func TestBuildPiperArgs_ZeroRate(t *testing.T) {
	// SpeakingRate == 0 means unset; should NOT emit --length_scale.
	voice := VoicePersona{ModelPath: "/models/test.onnx", SpeakingRate: 0}
	args := buildPiperArgs(voice)
	for _, a := range args {
		if a == "--length_scale" {
			t.Error("--length_scale should not be emitted for SpeakingRate == 0")
		}
	}
}

// ---------------------------------------------------------------------------
// rawBytesToInt16
// ---------------------------------------------------------------------------

func TestRawBytesToInt16(t *testing.T) {
	// Encode known int16 values as little-endian bytes.
	// 0x0001 = 1, 0x00FF = 255, 0x8000 = -32768 (min int16)
	data := []byte{
		0x01, 0x00, // 1
		0xFF, 0x00, // 255
		0x00, 0x80, // -32768
	}
	got := rawBytesToInt16(data)
	want := []int16{1, 255, -32768}
	if len(got) != len(want) {
		t.Fatalf("rawBytesToInt16 len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rawBytesToInt16[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestRawBytesToInt16_OddLength(t *testing.T) {
	// Odd trailing byte should be silently discarded.
	data := []byte{0x01, 0x00, 0xFF} // 1 complete sample + 1 orphan byte
	got := rawBytesToInt16(data)
	if len(got) != 1 {
		t.Errorf("expected 1 sample from 3 bytes, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short = %q, want %q", got, "hello")
	}
	got := truncate("hello world", 5)
	if got != "hello…" {
		t.Errorf("truncate long = %q, want %q", got, "hello…")
	}
}
