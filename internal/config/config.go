// Package config defines the full configuration schema for DUSTY.
// All sections are defined upfront; unused sections (STT, TTS, display)
// will be populated as later phases are implemented.
package config

// Config is the top-level configuration for DUSTY.
type Config struct {
	Agent     AgentConfig     `toml:"agent"`
	Inference InferenceConfig `toml:"inference"`
	Audio     AudioConfig     `toml:"audio"`
	STT       STTConfig       `toml:"stt"`
	TTS       TTSConfig       `toml:"tts"`
	Display   DisplayConfig   `toml:"display"`
	Memory    MemoryConfig    `toml:"memory"`
}

// AgentConfig controls the agent's identity and behavior.
type AgentConfig struct {
	Name        string `toml:"name"`
	Personality string `toml:"personality"`
	WakeWord    string `toml:"wake_word"`
	MaxHistory  int    `toml:"max_history"`
}

// InferenceConfig controls model routing.
type InferenceConfig struct {
	Mode  string              `toml:"mode"` // "local", "cloud", "auto"
	Local LocalInferenceConfig  `toml:"local"`
	Cloud CloudInferenceConfig  `toml:"cloud"`
}

// LocalInferenceConfig configures the local (Ollama) inference backend.
type LocalInferenceConfig struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
	Endpoint string `toml:"endpoint"`
}

// CloudInferenceConfig configures the cloud inference backend.
type CloudInferenceConfig struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
	// API key is loaded from environment variables, not stored in config.
}

// AudioConfig controls microphone capture and speaker playback (Phase 2).
type AudioConfig struct {
	SampleRate    int     `toml:"sample_rate"`     // samples per second; default 16000
	Channels      int     `toml:"channels"`        // 1 = mono; default 1
	InputDevice   string  `toml:"input_device"`    // "" = system default
	OutputDevice  string  `toml:"output_device"`   // "" = system default
	VADThreshold  float32 `toml:"vad_threshold"`   // RMS energy threshold 0–1; default 0.02
	SilenceMs     int     `toml:"silence_ms"`      // ms of silence to end an utterance; default 800
	MaxSegmentSec int     `toml:"max_segment_sec"` // hard cap on utterance length; default 30
}

// STTConfig configures speech-to-text (Phase 2).
type STTConfig struct {
	Engine   string `toml:"engine"`
	Model    string `toml:"model"`
	Language string `toml:"language"`
}

// TTSConfig configures text-to-speech (Phase 2).
type TTSConfig struct {
	Engine       string                 `toml:"engine"`
	DefaultVoice string                 `toml:"default_voice"`
	PiperBin     string                 `toml:"piper_bin"` // path to piper binary; default "piper"
	Voices       map[string]VoiceConfig `toml:"voices"`
}

// VoiceConfig defines a single TTS voice persona.
type VoiceConfig struct {
	Model        string  `toml:"model"`
	SpeakingRate float64 `toml:"speaking_rate"`
	Description  string  `toml:"description"`
}

// DisplayConfig configures the display system (Phase 3-4).
type DisplayConfig struct {
	Mode   string       `toml:"mode"`  // "tui", "visual", "both"
	Theme  string       `toml:"theme"` // "phosphor-green", "amber", "blue", "custom"
	Visual VisualConfig `toml:"visual"`
}

// VisualConfig configures the visual/graphics display mode (Phase 4).
type VisualConfig struct {
	Resolution [2]int `toml:"resolution"`
	Palette    string `toml:"palette"`
	CRTShader  bool   `toml:"crt_shader"`
}

// MemoryConfig controls conversation memory persistence.
type MemoryConfig struct {
	Persist bool   `toml:"persist"`
	Path    string `toml:"path"`
}

// Defaults returns a Config populated with sensible default values.
func Defaults() Config {
	return Config{
		Agent: AgentConfig{
			Name:        "DUSTY",
			Personality: "philosopher",
			WakeWord:    "hey dusty",
			MaxHistory:  50,
		},
		Inference: InferenceConfig{
			Mode: "local",
			Local: LocalInferenceConfig{
				Provider: "ollama",
				Model:    "gemma3:1b",
				Endpoint: "http://localhost:11434",
			},
			Cloud: CloudInferenceConfig{
				Provider: "anthropic",
				Model:    "claude-sonnet-4-20250514",
			},
		},
		STT: STTConfig{
			Engine:   "whisper",
			Model:    "base",
			Language: "en",
		},
		Audio: AudioConfig{
			SampleRate:    16000,
			Channels:      1,
			VADThreshold:  0.02,
			SilenceMs:     800,
			MaxSegmentSec: 30,
		},
		TTS: TTSConfig{
			Engine:       "piper",
			DefaultVoice: "philosopher",
			PiperBin:     "piper",
		},
		Display: DisplayConfig{
			Mode:  "tui",
			Theme: "phosphor-green",
			Visual: VisualConfig{
				Resolution: [2]int{256, 256},
				Palette:    "16-retro",
				CRTShader:  true,
			},
		},
		Memory: MemoryConfig{
			Persist: true,
			Path:    "dusty.memory.json",
		},
	}
}
