package voice_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/stt"
	"github.com/tristanj/dusty/internal/tts"
	"github.com/tristanj/dusty/internal/voice"
	"log/slog"
	"os"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockCapture feeds pre-built frames then closes.
type mockCapture struct {
	frames [][]float32
}

func (m *mockCapture) Start(_ context.Context) (<-chan []float32, error) {
	ch := make(chan []float32, len(m.frames))
	for _, f := range m.frames {
		ch <- f
	}
	close(ch)
	return ch, nil
}

func (m *mockCapture) Close() error { return nil }

// mockPlayback records received chunks and notifies via gotAudio on the first.
type mockPlayback struct {
	mu       sync.Mutex
	received [][]int16
	gotAudio chan struct{}
	once     sync.Once
}

func newMockPlayback() *mockPlayback {
	return &mockPlayback{gotAudio: make(chan struct{})}
}

func (m *mockPlayback) Play(ctx context.Context, audio <-chan []int16) error {
	for {
		select {
		case chunk, ok := <-audio:
			if !ok {
				return nil
			}
			m.mu.Lock()
			m.received = append(m.received, chunk)
			m.mu.Unlock()
			m.once.Do(func() { close(m.gotAudio) })
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *mockPlayback) Close() error { return nil }

func (m *mockPlayback) totalSamples() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, c := range m.received {
		n += len(c)
	}
	return n
}

// mockSTT returns a fixed transcription for any audio.
type mockSTT struct {
	result string
	called int
	mu     sync.Mutex
}

func (m *mockSTT) Transcribe(_ context.Context, _ []float32) (stt.STTResult, error) {
	m.mu.Lock()
	m.called++
	m.mu.Unlock()
	return stt.STTResult{Text: m.result, Language: "en", Duration: time.Second}, nil
}

func (m *mockSTT) Close() error { return nil }

// mockChatter records calls and returns a fixed token sequence.
type mockChatter struct {
	mu       sync.Mutex
	received []string
	tokens   []string
	called   chan struct{}
	once     sync.Once
}

func newMockChatter(tokens []string) *mockChatter {
	return &mockChatter{tokens: tokens, called: make(chan struct{})}
}

func (m *mockChatter) Chat(_ context.Context, text string) (<-chan string, error) {
	m.mu.Lock()
	m.received = append(m.received, text)
	m.mu.Unlock()
	m.once.Do(func() { close(m.called) })

	ch := make(chan string, len(m.tokens))
	for _, t := range m.tokens {
		ch <- t
	}
	close(ch)
	return ch, nil
}

// mockTTS returns a fixed audio chunk for any sentence.
type mockTTS struct {
	mu          sync.Mutex
	synthesized []string
	chunk       []int16
}

func (m *mockTTS) Synthesize(_ context.Context, text string, _ tts.VoicePersona) (<-chan []int16, error) {
	m.mu.Lock()
	m.synthesized = append(m.synthesized, text)
	m.mu.Unlock()

	ch := make(chan []int16, 1)
	ch <- m.chunk
	close(ch)
	return ch, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func speechFrames(count, size int) [][]float32 {
	frames := make([][]float32, count)
	for i := range frames {
		f := make([]float32, size)
		for j := range f {
			f[j] = 0.5 // above any threshold
		}
		frames[i] = f
	}
	return frames
}

func silenceFrames(count, size int) [][]float32 {
	frames := make([][]float32, count)
	for i := range frames {
		frames[i] = make([]float32, size)
	}
	return frames
}

func testConfig(silenceMs int) config.Config {
	cfg := config.Defaults()
	cfg.Audio.SampleRate = 16000
	cfg.Audio.VADThreshold = 0.02
	cfg.Audio.SilenceMs = silenceMs
	cfg.Audio.MaxSegmentSec = 30
	return cfg
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestVoiceLoop_SpeechFlowsToPlayback verifies the complete pipeline:
// speech frames → VAD → STT → Agent → TTS → Playback.
func TestVoiceLoop_SpeechFlowsToPlayback(t *testing.T) {
	// SilenceMs=320 → silenceTimeoutFrames = 320/32 = 10 frames.
	// 20 speech frames + 15 silence frames guarantees VAD emits.
	frames := append(speechFrames(20, 512), silenceFrames(15, 512)...)

	cap := &mockCapture{frames: frames}
	play := newMockPlayback()
	mockStt := &mockSTT{result: "hello dusty"}
	mockAgent := newMockChatter([]string{"Hello", " there", "!", " How", " are", " you", "?"})
	mockTtsEng := &mockTTS{chunk: []int16{1, 2, 3, 4}}

	vl := voice.New(voice.Config{
		AppConfig: testConfig(320),
		Agent:     mockAgent,
		Capture:   cap,
		Playback:  play,
		STT:       mockStt,
		TTS:       mockTtsEng,
		Voice:     tts.VoicePersona{Name: "test", ModelPath: "/model.onnx", SpeakingRate: 1.0},
		Log:       discardLogger(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go vl.Run(ctx) //nolint:errcheck

	// Wait for playback to receive audio, then cancel the loop.
	select {
	case <-play.gotAudio:
		cancel()
	case <-ctx.Done():
		t.Fatal("timed out — audio never reached playback")
	}

	// Agent must have received the transcribed text.
	mockAgent.mu.Lock()
	agentCalls := mockAgent.received
	mockAgent.mu.Unlock()

	if len(agentCalls) == 0 {
		t.Fatal("agent was never called")
	}
	if agentCalls[0] != "hello dusty" {
		t.Errorf("agent received %q, want %q", agentCalls[0], "hello dusty")
	}

	// TTS must have been invoked with at least one sentence.
	mockTtsEng.mu.Lock()
	synthesized := mockTtsEng.synthesized
	mockTtsEng.mu.Unlock()

	if len(synthesized) == 0 {
		t.Fatal("TTS was never called")
	}
	// Playback must contain audio samples.
	if play.totalSamples() == 0 {
		t.Error("playback received no samples")
	}
}

// TestVoiceLoop_SilenceNeverReachesAgent verifies that pure silence never
// triggers STT or Agent calls.
func TestVoiceLoop_SilenceNeverReachesAgent(t *testing.T) {
	frames := silenceFrames(50, 512)

	cap := &mockCapture{frames: frames}
	play := newMockPlayback()
	mockStt := &mockSTT{result: "should never be called"}
	mockAgent := newMockChatter(nil)
	mockTtsEng := &mockTTS{}

	vl := voice.New(voice.Config{
		AppConfig: testConfig(320),
		Agent:     mockAgent,
		Capture:   cap,
		Playback:  play,
		STT:       mockStt,
		TTS:       mockTtsEng,
		Voice:     tts.VoicePersona{},
		Log:       discardLogger(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go vl.Run(ctx) //nolint:errcheck

	// Wait for the loop to finish processing the silence.
	<-ctx.Done()

	if mockStt.called > 0 {
		t.Errorf("STT should not be called for pure silence, was called %d times", mockStt.called)
	}
	mockAgent.mu.Lock()
	calls := len(mockAgent.received)
	mockAgent.mu.Unlock()
	if calls > 0 {
		t.Errorf("agent should not be called for pure silence, was called %d times", calls)
	}
}

// TestVoiceLoop_ContextCancellation verifies the loop stops cleanly when ctx
// is cancelled without panicking or goroutine leaks.
func TestVoiceLoop_ContextCancellation(t *testing.T) {
	// Endless stream: use a large number of speech frames.
	frames := speechFrames(200, 512)

	cap := &mockCapture{frames: frames}
	play := newMockPlayback()

	vl := voice.New(voice.Config{
		AppConfig: testConfig(800),
		Agent:     newMockChatter([]string{"hi"}),
		Capture:   cap,
		Playback:  play,
		STT:       &mockSTT{result: "hi"},
		TTS:       &mockTTS{chunk: []int16{1}},
		Voice:     tts.VoicePersona{},
		Log:       discardLogger(),
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		vl.Run(ctx) //nolint:errcheck
	}()

	// Let it run briefly, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Run returned — success
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after ctx cancellation")
	}
}

// TestStreamTokensToTTS_SentenceBoundaries verifies that the agent goroutine's
// sentence splitting logic works by testing the exported firstSentenceEnd helper
// indirectly through a full pipeline run with controlled tokens.
func TestVoiceLoop_SentenceStreaming(t *testing.T) {
	// The agent returns tokens that form two sentences.
	// We verify TTS is called twice (once per sentence).
	frames := append(speechFrames(20, 512), silenceFrames(15, 512)...)

	cap := &mockCapture{frames: frames}
	play := newMockPlayback()
	mockStt := &mockSTT{result: "test"}
	// Two complete sentences in the token stream.
	mockAgent := newMockChatter([]string{"Hello", " world", ".", " Goodbye", " world", "."})
	mockTtsEng := &mockTTS{chunk: []int16{1}}

	vl := voice.New(voice.Config{
		AppConfig: testConfig(320),
		Agent:     mockAgent,
		Capture:   cap,
		Playback:  play,
		STT:       mockStt,
		TTS:       mockTtsEng,
		Voice:     tts.VoicePersona{},
		Log:       discardLogger(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go vl.Run(ctx) //nolint:errcheck

	// Wait for playback to receive anything.
	select {
	case <-play.gotAudio:
		// Give the pipeline a moment to finish the second sentence too.
		time.Sleep(100 * time.Millisecond)
		cancel()
	case <-ctx.Done():
		t.Fatal("timed out — no audio produced")
	}

	mockTtsEng.mu.Lock()
	n := len(mockTtsEng.synthesized)
	mockTtsEng.mu.Unlock()

	if n < 2 {
		t.Errorf("expected TTS called at least 2 times for 2 sentences, got %d", n)
	}
}
