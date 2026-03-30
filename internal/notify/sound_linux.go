//go:build linux

package notify

import "os/exec"

func playAudio(path string) error {
	cmd := exec.Command("aplay", "-q", path)
	return cmd.Run()
}
