//go:build !linux

package notify

import (
	"fmt"
)

// On non-Linux platforms, fall back to terminal bell.
func playAudio(_ string) error {
	fmt.Print("\a")
	return nil
}
