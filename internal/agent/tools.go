package agent

import (
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
)

// Tool defines an action the agent can invoke during conversation.
type Tool interface {
	Name() string
	Description() string
	// Params returns the parameter schema for this tool.
	// Returns nil for tools that take no parameters.
	Params() map[string]*schema.ParameterInfo
	Execute(args map[string]any) (string, error)
}

// ToolInfoAdapter converts a Tool into an Eino schema.ToolInfo for passing to the model.
func ToolInfoAdapter(t Tool) *schema.ToolInfo {
	params := t.Params()
	info := &schema.ToolInfo{
		Name: t.Name(),
		Desc: t.Description(),
	}
	if params != nil {
		info.ParamsOneOf = schema.NewParamsOneOfByParams(params)
	}
	return info
}

// TimeTool reports the current time. A simple tool for testing.
type TimeTool struct{}

func (t TimeTool) Name() string        { return "current_time" }
func (t TimeTool) Description() string { return "Returns the current date and time." }
func (t TimeTool) Params() map[string]*schema.ParameterInfo {
	return nil
}
func (t TimeTool) Execute(_ map[string]any) (string, error) {
	return fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC1123)), nil
}

// MeshTools is an optional set of mesh communication tools. Non-nil when
// mesh is enabled in config. Injected by the mesh subsystem at startup.
var MeshTools []Tool

// MeshSystemPromptAddendum is appended to the persona system prompt when the
// mesh subsystem is active. It instructs the agent on terse message composition
// and inbox management.
const MeshSystemPromptAddendum = `

--- Mesh Communication ---
You are connected to a LoRa mesh radio network and can exchange short messages with other DUSTY agents nearby.

Key constraints:
- Messages must be ≤ 180 characters (LoRa duty-cycle limit). Be terse — think SMS, not email.
- Summarise intent: "Meet at deep playa, 10pm?" not a paragraph.
- Use mesh_send(target, message) to send a message. Target can be a peer agent name or node ID (e.g. "!abcd1234").
- Use mesh_inbox(action) to check messages. Actions: "unread", "history peer=<name>", "replay id=<id>", "peer_facts peer=<name>", "peers".
- Check mesh_inbox("unread") at the start of a session or when the user asks about communications.
- When composing a reply, acknowledge the sender by name if known.`

// DefaultTools returns the set of tools available to the agent.
// Mesh tools are appended when the mesh subsystem is enabled.
func DefaultTools() []Tool {
	tools := []Tool{
		TimeTool{},
	}
	tools = append(tools, MeshTools...)
	return tools
}
