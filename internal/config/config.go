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
	Mesh      MeshConfig      `toml:"mesh"`
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
	Mode       string                `toml:"mode"` // "local", "cloud", "auto"
	Local      LocalInferenceConfig  `toml:"local"`
	Cloud      CloudInferenceConfig  `toml:"cloud"`
	Classifier ClassifierConfig      `toml:"classifier"`
}

// ClassifierConfig configures the intent classifier and training data collector.
// All fields are optional — when Enabled is false the system behaves as before.
type ClassifierConfig struct {
	// Enabled activates intent-based routing and training data collection.
	Enabled bool `toml:"enabled"`
	// ToolModel is the Ollama model used when ShouldUseTool=true and Confidence >= threshold.
	// Falls back to inference.local.model when empty.
	ToolModel string `toml:"tool_model"`
	// ChatModel is the Ollama model used when ShouldUseTool=false and Confidence >= threshold.
	// Falls back to inference.local.model when empty.
	ChatModel string `toml:"chat_model"`
	// ConfidenceThreshold is the minimum confidence for model routing decisions.
	// Below this value the default model is used with tools enabled (conservative fallback).
	// Defaults to 0.7 when zero.
	ConfidenceThreshold float64 `toml:"confidence_threshold"`
	// TrainingDataPath is the file path for JSONL training data. Empty = disabled.
	TrainingDataPath string `toml:"training_data_path"`
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

// MeshConfig configures the LoRa mesh communication subsystem.
type MeshConfig struct {
	Enabled           bool    `toml:"enabled"`
	SerialPort        string  `toml:"serial_port"`        // e.g. /dev/ttyUSB0
	BaudRate          int     `toml:"baud_rate"`          // default 115200
	NodeID            string  `toml:"node_id"`            // This node's Meshtastic node ID, e.g. !abcd1234
	AgentName         string  `toml:"agent_name"`         // Name announced in discovery (defaults to agent.name)
	OwnerName         string  `toml:"owner_name"`         // Human owner name announced in discovery
	PSK               string  `toml:"psk"`                // AES-256 pre-shared key (hex) for encrypted channel
	HeartbeatInterval string  `toml:"heartbeat_interval"` // e.g. "5m"
	DiscoveryTimeout  string  `toml:"discovery_timeout"`  // e.g. "30s"
	PeerTTL           string  `toml:"peer_ttl"`           // e.g. "30m"
	QueueSize         int     `toml:"queue_size"`         // Outbound message queue depth (default 16)
	RateLimitSecs     int     `toml:"rate_limit_secs"`    // Min seconds between sends (default 30)
	IdleTimeoutMins   int     `toml:"idle_timeout_mins"`  // Minutes before user considered idle (default 5)
	MessageRetention  int     `toml:"message_retention"`  // Max stored messages (default 200)
	InteractionLimit  int     `toml:"interaction_limit"`  // Max interactions per peer (default 100)
	PeersFile         string  `toml:"peers_file"`         // default mesh-peers.json
	PeerMemoryFile    string  `toml:"peer_memory_file"`   // default mesh-peer-memory.json
	MessagesFile      string  `toml:"messages_file"`      // default mesh-messages.json
	SoundFile         string  `toml:"sound_file"`         // default assets/sounds/transmission.wav
	// GPS stub configuration.
	GPSLat float64 `toml:"gps_lat"`
	GPSLon float64 `toml:"gps_lon"`
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
			Classifier: ClassifierConfig{
				Enabled:             false,
				ConfidenceThreshold: 0.7,
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
		Mesh: MeshConfig{
			Enabled:           false,
			SerialPort:        "/dev/ttyUSB0",
			BaudRate:          115200,
			HeartbeatInterval: "5m",
			DiscoveryTimeout:  "30s",
			PeerTTL:           "30m",
			QueueSize:         16,
			RateLimitSecs:     30,
			IdleTimeoutMins:   5,
			MessageRetention:  200,
			InteractionLimit:  100,
			PeersFile:         "mesh-peers.json",
			PeerMemoryFile:    "mesh-peer-memory.json",
			MessagesFile:      "mesh-messages.json",
			SoundFile:         "assets/sounds/transmission.wav",
		},
	}
}
