package tui

import (
	"fmt"
	"strings"

	"github.com/tristanj/dusty/internal/agent"
	"github.com/tristanj/dusty/internal/config"
)

// CommandResult holds the output of a slash command execution.
type CommandResult struct {
	// Output is informational text to show in the conversation pane.
	Output string
	// Quit signals that the app should exit.
	Quit bool
	// SettingsChanged signals that model/persona/mode was modified.
	SettingsChanged bool
}

// VoiceController abstracts the voice loop lifecycle so the TUI can toggle it.
type VoiceController interface {
	Active() bool
	// ToggleVoice returns an error message if toggling fails, or "" on success.
	ToggleVoice() string
}

// CommandHandler processes slash commands entered in the TUI input.
type CommandHandler struct {
	agent *agent.Agent
	cfg   *config.Config
	voice VoiceController
}

// NewCommandHandler creates a command handler.
func NewCommandHandler(a *agent.Agent, cfg *config.Config, voice VoiceController) CommandHandler {
	return CommandHandler{agent: a, cfg: cfg, voice: voice}
}

// Handle processes a slash command string and returns the result.
// input must start with "/".
func (h CommandHandler) Handle(input string) CommandResult {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return CommandResult{}
	}
	cmd := parts[0]

	switch cmd {
	case "/quit", "/exit", "/q":
		if err := h.agent.Close(); err != nil {
			return CommandResult{Output: fmt.Sprintf("Warning saving memory: %v", err), Quit: true}
		}
		return CommandResult{Output: "Goodbye.", Quit: true}

	case "/clear":
		h.agent.ClearMemory()
		return CommandResult{Output: "[memory cleared]", SettingsChanged: true}

	case "/persona":
		if len(parts) < 2 {
			return CommandResult{
				Output: fmt.Sprintf("Available personas: %s\nUsage: /persona <name>",
					strings.Join(agent.ListPersonas(), ", ")),
			}
		}
		h.agent.SetPersona(parts[1])
		h.cfg.Agent.Personality = parts[1]
		return CommandResult{
			Output:          fmt.Sprintf("[persona set to: %s]", parts[1]),
			SettingsChanged: true,
		}

	case "/model":
		if len(parts) < 2 {
			models := h.agent.ListModels()
			var sb strings.Builder
			for _, m := range models {
				local := ""
				if m.Local {
					local = " [local]"
				}
				sb.WriteString(fmt.Sprintf("  %s (%s)%s\n", m.Name, m.Provider, local))
			}
			return CommandResult{Output: strings.TrimRight(sb.String(), "\n")}
		}
		h.cfg.Inference.Local.Model = parts[1]
		return CommandResult{
			Output:          fmt.Sprintf("[model set to: %s — takes effect on next message]", parts[1]),
			SettingsChanged: true,
		}

	case "/mode":
		if len(parts) < 2 {
			return CommandResult{Output: "Usage: /mode <local|cloud|auto>"}
		}
		switch parts[1] {
		case "local":
			h.agent.SetRoutingPreference(agent.PreferLocal)
		case "cloud":
			h.agent.SetRoutingPreference(agent.PreferCloud)
		case "auto":
			h.agent.SetRoutingPreference(agent.PreferAuto)
		default:
			return CommandResult{Output: fmt.Sprintf("Unknown mode: %s (use local, cloud, or auto)", parts[1])}
		}
		h.cfg.Inference.Mode = parts[1]
		return CommandResult{
			Output:          fmt.Sprintf("[routing mode set to: %s]", parts[1]),
			SettingsChanged: true,
		}

	case "/status":
		return CommandResult{
			Output: fmt.Sprintf("State   : %s\nMemory  : %d messages\nModel   : %s [%s]\nPersona : %s",
				h.agent.State(),
				h.agent.MemoryLen(),
				h.cfg.Inference.Local.Model,
				h.cfg.Inference.Mode,
				h.cfg.Agent.Personality,
			),
		}

	case "/voice":
		sub := ""
		if len(parts) > 1 {
			sub = parts[1]
		}
		switch sub {
		case "start", "stop":
			msg := h.voice.ToggleVoice()
			if msg != "" {
				return CommandResult{Output: msg, SettingsChanged: true}
			}
			if h.voice.Active() {
				return CommandResult{Output: "[voice mode started — speak to DUSTY]", SettingsChanged: true}
			}
			return CommandResult{Output: "[voice mode stopped]", SettingsChanged: true}
		default:
			status := "inactive"
			if h.voice.Active() {
				status = "active"
			}
			return CommandResult{Output: fmt.Sprintf("Voice: %s\nUsage: /voice start|stop", status)}
		}

	case "/send":
		tool := findAgentTool("mesh_send")
		if tool == nil {
			return CommandResult{Output: "[mesh not enabled]"}
		}
		if len(parts) < 3 {
			return CommandResult{Output: "Usage: /send <target> <message>"}
		}
		target := parts[1]
		message := strings.Join(parts[2:], " ")
		result, err := tool.Execute(map[string]any{"target": target, "message": message})
		if err != nil {
			return CommandResult{Output: fmt.Sprintf("[mesh error] %v", err)}
		}
		return CommandResult{Output: fmt.Sprintf("[mesh] %s", result)}

	case "/inbox":
		tool := findAgentTool("mesh_inbox")
		if tool == nil {
			return CommandResult{Output: "[mesh not enabled]"}
		}
		args := map[string]any{"action": "unread"}
		if len(parts) >= 2 {
			args["action"] = parts[1]
		}
		for _, kv := range parts[2:] {
			if idx := strings.IndexByte(kv, '='); idx > 0 {
				args[kv[:idx]] = kv[idx+1:]
			}
		}
		result, err := tool.Execute(args)
		if err != nil {
			return CommandResult{Output: fmt.Sprintf("[mesh error] %v", err)}
		}
		return CommandResult{Output: result}

	case "/help":
		lines := []string{
			"Commands:",
			"  /clear              — Clear conversation memory",
			"  /persona <name>     — Switch persona (philosopher, companion, minimal)",
			"  /model [name]       — List or switch local model",
			"  /mode <mode>        — Switch routing: local, cloud, auto",
			"  /status             — Show agent status",
			"  /voice start|stop   — Toggle voice mode",
		}
		if findAgentTool("mesh_send") != nil {
			lines = append(lines,
				"  /send <target> <msg> — Send mesh transmission",
				"  /inbox [action]      — Mesh inbox (unread, history, peers, peer_facts, replay)",
			)
		}
		lines = append(lines,
			"  /quit               — Exit (saves memory)",
			"",
			"Keyboard shortcuts:",
			"  F1                  — Toggle voice mode",
			"  F2                  — Settings panel",
			"  Ctrl+L              — Clear memory",
			"  Ctrl+C              — Quit",
		)
		return CommandResult{Output: strings.Join(lines, "\n")}

	default:
		return CommandResult{Output: fmt.Sprintf("Unknown command: %s (try /help)", cmd)}
	}
}

// findAgentTool looks up a named tool from agent.MeshTools.
func findAgentTool(name string) agent.Tool {
	for _, t := range agent.MeshTools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}
