package tts_test

import (
	"testing"

	"github.com/tristanj/dusty/internal/config"
	"github.com/tristanj/dusty/internal/tts"
)

func makeTestTTSConfig() config.TTSConfig {
	return config.TTSConfig{
		Engine:       "piper",
		DefaultVoice: "philosopher",
		PiperBin:     "piper",
		Voices: map[string]config.VoiceConfig{
			"philosopher": {
				Model:        "assets/voices/en_US-lessac-medium.onnx",
				SpeakingRate: 0.95,
				Description:  "Warm, contemplative",
			},
			"robot": {
				Model:        "assets/voices/en_US-ryan-medium.onnx",
				SpeakingRate: 1.0,
				Description:  "Crisp delivery",
			},
		},
	}
}

func TestPersonaRegistry_GetKnownVoice(t *testing.T) {
	reg := tts.NewPersonaRegistry(makeTestTTSConfig())

	v, err := reg.Get("philosopher")
	if err != nil {
		t.Fatalf("Get(philosopher) error: %v", err)
	}
	if v.Name != "philosopher" {
		t.Errorf("Name = %q, want %q", v.Name, "philosopher")
	}
	if v.ModelPath != "assets/voices/en_US-lessac-medium.onnx" {
		t.Errorf("ModelPath = %q, unexpected", v.ModelPath)
	}
	if v.SpeakingRate != 0.95 {
		t.Errorf("SpeakingRate = %f, want 0.95", v.SpeakingRate)
	}
}

func TestPersonaRegistry_GetFallsBackToDefault(t *testing.T) {
	reg := tts.NewPersonaRegistry(makeTestTTSConfig())

	// "oracle" is not registered; should fall back to "philosopher".
	v, err := reg.Get("oracle")
	if err != nil {
		t.Fatalf("Get(oracle) should fall back to default, got error: %v", err)
	}
	if v.Name != "philosopher" {
		t.Errorf("fallback name = %q, want %q", v.Name, "philosopher")
	}
}

func TestPersonaRegistry_GetErrorWhenNeitherFound(t *testing.T) {
	cfg := config.TTSConfig{
		DefaultVoice: "missing",
		Voices:       map[string]config.VoiceConfig{},
	}
	reg := tts.NewPersonaRegistry(cfg)

	_, err := reg.Get("also-missing")
	if err == nil {
		t.Error("expected error when neither voice nor default is found")
	}
}

func TestPersonaRegistry_Default(t *testing.T) {
	reg := tts.NewPersonaRegistry(makeTestTTSConfig())

	v, err := reg.Default()
	if err != nil {
		t.Fatalf("Default() error: %v", err)
	}
	if v.Name != "philosopher" {
		t.Errorf("Default().Name = %q, want %q", v.Name, "philosopher")
	}
}

func TestPersonaRegistry_List(t *testing.T) {
	reg := tts.NewPersonaRegistry(makeTestTTSConfig())

	names := reg.List()
	if len(names) != 2 {
		t.Fatalf("List() len = %d, want 2", len(names))
	}
	// List must be sorted.
	if names[0] != "philosopher" || names[1] != "robot" {
		t.Errorf("List() = %v, want [philosopher robot]", names)
	}
}

func TestPersonaRegistry_ZeroSpeakingRateNormalized(t *testing.T) {
	cfg := config.TTSConfig{
		DefaultVoice: "silent",
		Voices: map[string]config.VoiceConfig{
			"silent": {Model: "model.onnx", SpeakingRate: 0},
		},
	}
	reg := tts.NewPersonaRegistry(cfg)

	v, err := reg.Get("silent")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if v.SpeakingRate != 1.0 {
		t.Errorf("zero SpeakingRate should be normalised to 1.0, got %f", v.SpeakingRate)
	}
}
