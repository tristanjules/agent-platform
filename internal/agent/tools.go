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

// noopTool is a minimal Tool implementation that always returns "ok".
// Used by the eval runner to provide placeholder tools without real infrastructure.
type noopTool struct{ name string }

func (n *noopTool) Name() string                              { return n.name }
func (n *noopTool) Description() string                       { return n.name + " (noop)" }
func (n *noopTool) Params() map[string]*schema.ParameterInfo { return nil }
func (n *noopTool) Execute(_ map[string]any) (string, error) { return "ok", nil }

// NewNoopTool returns a Tool that accepts any arguments and always returns "ok".
func NewNoopTool(name string) Tool { return &noopTool{name: name} }

// evalStub is a noop Tool with a proper description and parameter schema,
// used by the eval runner to give the model enough context to call the tool.
type evalStub struct {
	name   string
	desc   string
	params map[string]*schema.ParameterInfo
}

func (e *evalStub) Name() string                              { return e.name }
func (e *evalStub) Description() string                       { return e.desc }
func (e *evalStub) Params() map[string]*schema.ParameterInfo { return e.params }
func (e *evalStub) Execute(_ map[string]any) (string, error)  { return "ok", nil }

// EvalToolStub returns a properly-described stub Tool for use in the eval harness.
// For known tool names (mesh_send, mesh_inbox, current_time) it returns a stub
// with the real schema. Unknown names fall back to NewNoopTool.
func EvalToolStub(name string) Tool {
	switch name {
	case "mesh_send":
		return &evalStub{
			name: "mesh_send",
			desc: "Send a text message to another DUSTY node on the LoRa mesh network.",
			params: map[string]*schema.ParameterInfo{
				"target":  {Type: schema.String, Desc: "Node ID or callsign of the recipient (e.g. DUSTY-B)", Required: true},
				"message": {Type: schema.String, Desc: "Text content of the message (≤180 chars)", Required: true},
			},
		}
	case "mesh_inbox":
		return &evalStub{
			name: "mesh_inbox",
			desc: "Check the inbox for messages received over the LoRa mesh network.",
			params: map[string]*schema.ParameterInfo{
				"action": {Type: schema.String, Desc: `Inbox action: "unread", "history peer=<name>", "peers"`, Required: false},
			},
		}
	case "current_time":
		return TimeTool{}
	default:
		return NewNoopTool(name)
	}
}

// MeshTools is an optional set of mesh communication tools. Non-nil when
// mesh is enabled in config. Injected by the mesh subsystem at startup.
var MeshTools []Tool

// MeshSystemPromptAddendum is appended to the persona system prompt when the
// mesh subsystem is active. It instructs the agent on terse message composition
// and inbox management, with few-shot examples for reliable tool calling.
const MeshSystemPromptAddendum = `

--- Mesh Communication ---
You are connected to a LoRa mesh radio network and can exchange short messages with other DUSTY agents nearby.

Key constraints:
- Messages must be ≤ 180 characters (LoRa duty-cycle limit). Be terse — think SMS, not email.
- Summarise intent: "Meet at deep playa, 10pm?" not a paragraph.
- Use mesh_send(target, message) to send a message. Target can be a peer agent name or node ID (e.g. "!abcd1234"). Node names are case-insensitive: "dusty-b", "DUSTY-B", and "Dusty-B" all refer to the same node.
- Use mesh_inbox(action) to check messages. Actions: "unread", "history peer=<name>", "replay id=<id>", "peer_facts peer=<name>", "peers".
- Check mesh_inbox("unread") at the start of a session or when the user asks about communications.
- When composing a reply, acknowledge the sender by name if known.
- There is NO broadcast tool. If asked to message everyone or all agents, explain this limitation and offer to send messages individually to known peers.

Examples of correct tool use:

User: "Send DUSTY-B the message hello"
→ call mesh_send(target="DUSTY-B", message="hello")

User: "tell dusty-b I'll be at the temple at sunset"
→ call mesh_send(target="dusty-b", message="I'll be at the temple at sunset")

User: "Can you message DUSTY-C and let them know I'm okay?"
→ call mesh_send(target="DUSTY-C", message="Your operator is okay")

User: "Do I have any unread messages?" / "What's in my inbox?" / "Any messages from DUSTY-B?"
→ call mesh_inbox(action="unread")

User: "Show me my message history with DUSTY-B"
→ call mesh_inbox(action="history peer=DUSTY-B")

User: "What time is it?"
→ call current_time()

Do NOT call any tool for questions about philosophy, feelings, the burn, general knowledge, or anything that does not require mesh communication or time lookup. For those, respond with plain text only.`

// DefaultTools returns the set of tools available to the agent.
// Mesh tools are appended when the mesh subsystem is enabled.
func DefaultTools() []Tool {
	tools := []Tool{
		TimeTool{},
	}
	tools = append(tools, MeshTools...)
	return tools
}
