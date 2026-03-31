package classifier

import (
	"context"
	"regexp"
	"testing"
)

func TestRuleClassifier_GoldenEvalCases(t *testing.T) {
	c := NewRuleClassifier(nil)
	ctx := context.Background()

	tests := []struct {
		name           string
		message        string
		wantTool       bool
		wantMinConf    float64
		wantToolHint   string // empty = don't check SuggestedTools
	}{
		// --- Golden eval cases ---
		{
			name:         "explicit send to named node",
			message:      "Send DUSTY-B the message hello",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "mesh_send",
		},
		{
			name:         "informal send with lowercase node",
			message:      "Tell dusty-b I'll be at the temple at sunset",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "mesh_send",
		},
		{
			name:         "message verb with named node",
			message:      "Can you message DUSTY-C?",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "mesh_send",
		},
		{
			name:        "broadcast-style request",
			message:     "Let the other agents know I'm heading to the temple",
			wantTool:    false,  // no node name, no clear tool verb — ambiguous/low-conf
			wantMinConf: 0.0,
		},
		{
			name:         "check unread messages",
			message:      "Do I have any unread messages?",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "mesh_inbox",
		},
		{
			name:         "what is in inbox",
			message:      "What's in my inbox?",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "mesh_inbox",
		},
		{
			name:         "message history with node",
			message:      "Show me my message history with DUSTY-B",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "mesh_inbox",
		},
		{
			name:         "current time query",
			message:      "What time is it?",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "current_time",
		},
		{
			name:        "philosophical no-tool question",
			message:     "What is the meaning of the burn?",
			wantTool:    false,
			wantMinConf: 0.9,
		},

		// --- Edge cases ---
		{
			name:         "lowercase node name only",
			message:      "tell dusty-b I'll be late",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "mesh_send",
		},
		{
			name:        "general philosophy",
			message:     "What is the meaning of life?",
			wantTool:    false,
			wantMinConf: 0.9,
		},
		{
			name:        "emotional statement",
			message:     "I feel deeply connected to the playa",
			wantTool:    false,
			wantMinConf: 0.9,
		},
		{
			name:         "current time alternate phrasing",
			message:      "What's the current time?",
			wantTool:     true,
			wantMinConf:  0.9,
			wantToolHint: "current_time",
		},
		{
			name:        "custom options nil uses defaults",
			message:     "Send DUSTY-B hello",
			wantTool:    true,
			wantMinConf: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(ctx, tt.message)

			if result.ShouldUseTool != tt.wantTool {
				t.Errorf("ShouldUseTool = %v, want %v (message: %q)", result.ShouldUseTool, tt.wantTool, tt.message)
			}
			if result.Confidence < tt.wantMinConf {
				t.Errorf("Confidence = %.2f, want >= %.2f (message: %q)", result.Confidence, tt.wantMinConf, tt.message)
			}
			if tt.wantToolHint != "" {
				found := false
				for _, s := range result.SuggestedTools {
					if s == tt.wantToolHint {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SuggestedTools = %v, want %q present (message: %q)", result.SuggestedTools, tt.wantToolHint, tt.message)
				}
			}
		})
	}
}

func TestRuleClassifier_LowConfidenceAmbiguous(t *testing.T) {
	c := NewRuleClassifier(nil)
	ctx := context.Background()

	ambiguous := []string{
		"Can you help coordinate with the others?",
		"We should probably check in with the rest of the crew",
	}

	for _, msg := range ambiguous {
		t.Run(msg, func(t *testing.T) {
			result := c.Classify(ctx, msg)
			if result.Confidence >= 0.7 {
				t.Errorf("expected low confidence for ambiguous input %q, got %.2f", msg, result.Confidence)
			}
		})
	}
}

func TestRuleClassifier_CustomOpts(t *testing.T) {
	custom := &ClassifierOpts{
		ToolPatterns: []toolPattern{
			{
				tool:   "custom_tool",
				strong: mustRegexp(`(?i)\bcustom\b`),
			},
		},
	}
	c := NewRuleClassifier(custom)
	ctx := context.Background()

	result := c.Classify(ctx, "do the custom thing")
	if !result.ShouldUseTool {
		t.Error("expected custom tool pattern to match")
	}
	if len(result.SuggestedTools) == 0 || result.SuggestedTools[0] != "custom_tool" {
		t.Errorf("expected SuggestedTools=[custom_tool], got %v", result.SuggestedTools)
	}
}

func mustRegexp(s string) *regexp.Regexp {
	return regexp.MustCompile(s)
}
