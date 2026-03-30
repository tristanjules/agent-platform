// Package notify handles incoming mesh transmission notifications:
// audio alerts, haptic feedback, and TUI/REPL interrupt coordination.
package notify

import (
	"fmt"
	"log"
	"os"
)

// SoundPlayer plays a notification sound.
type SoundPlayer interface {
	PlayFile(path string) error
}

// TerminalBellPlayer writes the terminal bell character as a fallback.
type TerminalBellPlayer struct{}

func (t *TerminalBellPlayer) PlayFile(_ string) error {
	fmt.Print("\a")
	return nil
}

// FileSoundPlayer plays a WAV file through the OS audio system.
// Uses the `aplay` command on Linux (standard on Raspberry Pi).
type FileSoundPlayer struct {
	soundPath string
}

// NewFileSoundPlayer creates a player using the given sound file path.
func NewFileSoundPlayer(soundPath string) *FileSoundPlayer {
	return &FileSoundPlayer{soundPath: soundPath}
}

// PlayFile plays the configured sound file, falling back to terminal bell on error.
func (p *FileSoundPlayer) PlayFile(path string) error {
	target := path
	if target == "" {
		target = p.soundPath
	}
	if _, err := os.Stat(target); err != nil {
		log.Printf("[notify] sound file not found (%s), using terminal bell", target)
		fmt.Print("\a")
		return nil
	}
	// Use aplay on Linux (Raspberry Pi), fallback silently on other platforms.
	return playAudio(target)
}
