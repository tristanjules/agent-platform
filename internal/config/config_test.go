package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanj/dusty/internal/config"
)

func TestDefaults(t *testing.T) {
	cfg := config.Defaults()

	if cfg.Agent.Name != "DUSTY" {
		t.Errorf("expected name DUSTY, got %s", cfg.Agent.Name)
	}
	if cfg.Inference.Mode != "local" {
		t.Errorf("expected mode local, got %s", cfg.Inference.Mode)
	}
	if cfg.Agent.MaxHistory != 50 {
		t.Errorf("expected max_history 50, got %d", cfg.Agent.MaxHistory)
	}

	// Phase 2: Audio defaults.
	if cfg.Audio.SampleRate != 16000 {
		t.Errorf("expected sample_rate 16000, got %d", cfg.Audio.SampleRate)
	}
	if cfg.Audio.Channels != 1 {
		t.Errorf("expected channels 1, got %d", cfg.Audio.Channels)
	}
	if cfg.Audio.VADThreshold != 0.02 {
		t.Errorf("expected vad_threshold 0.02, got %f", cfg.Audio.VADThreshold)
	}
	if cfg.Audio.SilenceMs != 800 {
		t.Errorf("expected silence_ms 800, got %d", cfg.Audio.SilenceMs)
	}
	if cfg.Audio.MaxSegmentSec != 30 {
		t.Errorf("expected max_segment_sec 30, got %d", cfg.Audio.MaxSegmentSec)
	}
	if cfg.TTS.PiperBin != "piper" {
		t.Errorf("expected piper_bin 'piper', got %q", cfg.TTS.PiperBin)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error loading nonexistent file")
	}
}

func TestLoadAndOverride(t *testing.T) {
	toml := `
[agent]
name = "TEST"
max_history = 10

[inference]
mode = "cloud"

[inference.local]
model = "llama3.2:1b"
endpoint = "http://localhost:11434"

[memory]
persist = false
path = ""
`
	tmp := filepath.Join(t.TempDir(), "test.toml")
	if err := os.WriteFile(tmp, []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Agent.Name != "TEST" {
		t.Errorf("expected name TEST, got %s", cfg.Agent.Name)
	}
	if cfg.Agent.MaxHistory != 10 {
		t.Errorf("expected max_history 10, got %d", cfg.Agent.MaxHistory)
	}
	if cfg.Inference.Mode != "cloud" {
		t.Errorf("expected mode cloud, got %s", cfg.Inference.Mode)
	}
	if cfg.Inference.Local.Model != "llama3.2:1b" {
		t.Errorf("expected model llama3.2:1b, got %s", cfg.Inference.Local.Model)
	}
	// Defaults for unset fields should remain.
	if cfg.Agent.Personality != "philosopher" {
		t.Errorf("expected default personality philosopher, got %s", cfg.Agent.Personality)
	}
}

func TestOllamaHostEnvOverride(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "http://pi.local:11434")

	toml := `
[inference.local]
endpoint = "http://localhost:11434"
`
	tmp := filepath.Join(t.TempDir(), "test.toml")
	if err := os.WriteFile(tmp, []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Inference.Local.Endpoint != "http://pi.local:11434" {
		t.Errorf("expected endpoint from env, got %s", cfg.Inference.Local.Endpoint)
	}
}
