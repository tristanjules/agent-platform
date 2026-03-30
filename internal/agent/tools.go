package agent

import (
	"fmt"
	"time"
)

// Tool defines an action the agent can invoke during conversation.
// This is a placeholder interface for Phase 1 — full tool use
// integration with Eino's tool calling comes in later phases.
type Tool interface {
	Name() string
	Description() string
	Execute(args map[string]any) (string, error)
}

// TimeTool reports the current time. A simple tool for testing.
type TimeTool struct{}

func (t TimeTool) Name() string        { return "current_time" }
func (t TimeTool) Description() string  { return "Returns the current date and time." }
func (t TimeTool) Execute(_ map[string]any) (string, error) {
	return fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC1123)), nil
}

// DefaultTools returns the set of tools available to the agent.
func DefaultTools() []Tool {
	return []Tool{
		TimeTool{},
	}
}
