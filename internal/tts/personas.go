// Package tts — VoicePersonaRegistry maps config-defined voice names to
// VoicePersona structs, with default-voice fallback.
package tts

import (
	"fmt"
	"sort"

	"github.com/tristanj/dusty/internal/config"
)

// PersonaRegistry maps persona names to VoicePersona structs and provides
// fallback to a default voice when a requested name is not found.
type PersonaRegistry struct {
	voices      map[string]VoicePersona
	defaultName string
}

// NewPersonaRegistry constructs a PersonaRegistry from the TTS config section.
// An empty or zero SpeakingRate in the config is normalised to 1.0.
func NewPersonaRegistry(cfg config.TTSConfig) *PersonaRegistry {
	voices := make(map[string]VoicePersona, len(cfg.Voices))
	for name, vc := range cfg.Voices {
		rate := vc.SpeakingRate
		if rate <= 0 {
			rate = 1.0
		}
		voices[name] = VoicePersona{
			Name:         name,
			ModelPath:    vc.Model,
			SpeakingRate: rate,
		}
	}
	return &PersonaRegistry{
		voices:      voices,
		defaultName: cfg.DefaultVoice,
	}
}

// Get returns the VoicePersona for the given name.
// If name is not registered, it falls back to the default voice.
// Returns an error only when neither the requested name nor the default voice
// is registered.
func (r *PersonaRegistry) Get(name string) (VoicePersona, error) {
	if v, ok := r.voices[name]; ok {
		return v, nil
	}
	// Fallback to default.
	if name != r.defaultName {
		if v, ok := r.voices[r.defaultName]; ok {
			return v, nil
		}
	}
	return VoicePersona{}, fmt.Errorf("tts: voice %q not found and no default voice configured", name)
}

// Default returns the default VoicePersona as specified by TTSConfig.DefaultVoice.
func (r *PersonaRegistry) Default() (VoicePersona, error) {
	return r.Get(r.defaultName)
}

// List returns all registered voice names in alphabetical order.
func (r *PersonaRegistry) List() []string {
	names := make([]string, 0, len(r.voices))
	for name := range r.voices {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
