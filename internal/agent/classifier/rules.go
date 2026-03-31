package classifier

import (
	"context"
	"regexp"
	"strings"
)

// toolPattern describes a single tool-intent detection rule.
type toolPattern struct {
	tool     string         // tool name this pattern maps to
	strong   *regexp.Regexp // high-confidence match (>= 0.9)
	weak     *regexp.Regexp // medium-confidence match (0.6)
}

// conversationPattern matches messages that are clearly not tool-related.
type conversationPattern struct {
	re *regexp.Regexp
}

// ClassifierOpts customises the RuleClassifier. Pass nil to use DUSTY defaults.
type ClassifierOpts struct {
	// ToolPatterns replaces the default tool detection patterns.
	// Each entry must have at least a strong or weak pattern.
	ToolPatterns []toolPattern
	// ConversationPatterns replaces the default no-tool patterns.
	ConversationPatterns []conversationPattern
}

// RuleClassifier is a keyword/pattern-based Classifier.
// It is bootstrap scaffolding: accurate for DUSTY's 3-tool schema, designed
// to be replaced by a fine-tuned model once training data is collected.
// Safe for concurrent use.
type RuleClassifier struct {
	tools    []toolPattern
	convos   []conversationPattern
}

// nodeNameRE matches DUSTY-style node references (case-insensitive).
// Covers: dusty-b, DUSTY-C, dusty_b, and bare !hex node IDs.
var nodeNameRE = regexp.MustCompile(`(?i)(dusty[-_][a-z0-9]+|![0-9a-f]{4,8})`)

// defaultToolPatterns returns the built-in rules covering mesh_send, mesh_inbox,
// and current_time.
func defaultToolPatterns() []toolPattern {
	return []toolPattern{
		{
			tool: "mesh_send",
			// Strong: explicit send/tell/text/ping verbs, OR "message <node>" (message used as
			// a verb targeting a specific node). The latter excludes "message history" because
			// "history" doesn't match the node name pattern.
			strong: regexp.MustCompile(`(?i)\b(send|tell|msg|text|ping|notify|contact|reach)\b|(?i)\bmessage\s+` + nodeNameRE.String()),
			// Weak: bare node name with no send verb (e.g. "DUSTY-B, can you hear me?").
			weak: nodeNameRE,
		},
		{
			tool: "mesh_inbox",
			// Strong: inbox-related keywords. "message history" scores 0.97 to outrank
			// the mesh_send node+verb match (0.95) when history is clearly requested.
			strong: regexp.MustCompile(`(?i)\b(inbox|unread|heard from|any word|reply|replies|message history|transmissions?|history)\b`),
			// Weak: standalone "messages" or questions about peers.
			weak: regexp.MustCompile(`(?i)\b(peers?|nodes?|who'?s? (on|out there|nearby|around|online|connected)|messages?)\b`),
		},
		{
			tool: "current_time",
			// Strong: explicit time request.
			strong: regexp.MustCompile(`(?i)\b(what time|current time|what'?s the time|time is it|timestamp)\b`),
		},
	}
}

// defaultConversationPatterns returns patterns that indicate a message is
// clearly conversational (no tool needed).
func defaultConversationPatterns() []conversationPattern {
	return []conversationPattern{
		// Philosophical / existential questions.
		{re: regexp.MustCompile(`(?i)\b(meaning of|purpose of|reason for|why do we|what is life|what are we|consciousness|existence|philosophy|philosophical|spiritual|metaphysical)\b`)},
		// Emotional / reflective statements.
		{re: regexp.MustCompile(`(?i)\b(i feel|i'm feeling|feeling (lost|found|connected|disconnected|happy|sad|anxious|grateful|overwhelmed|inspired)|how are you|how do you feel)\b`)},
		// General knowledge questions with no tool relevance.
		{re: regexp.MustCompile(`(?i)\b(tell me about|explain|what is|who is|what was|history of|how does|why is|when did|where did|define|describe)\b`)},
		// Burn-specific conversational topics.
		{re: regexp.MustCompile(`(?i)\b(burning man|the burn|the playa|radical (self[-\s]reliance|inclusion|gifting|decommodification|participation|expression)|ten principles|temple|effigy|default world)\b`)},
	}
}

// NewRuleClassifier creates a RuleClassifier. Pass nil opts for DUSTY defaults.
func NewRuleClassifier(opts *ClassifierOpts) *RuleClassifier {
	r := &RuleClassifier{}
	if opts != nil && len(opts.ToolPatterns) > 0 {
		r.tools = opts.ToolPatterns
	} else {
		r.tools = defaultToolPatterns()
	}
	if opts != nil && len(opts.ConversationPatterns) > 0 {
		r.convos = opts.ConversationPatterns
	} else {
		r.convos = defaultConversationPatterns()
	}
	return r
}

// Classify implements Classifier.
func (r *RuleClassifier) Classify(_ context.Context, message string) ClassifyResult {
	msg := strings.TrimSpace(message)

	// Score each tool pattern.
	type match struct {
		tool       string
		confidence float64
	}
	var best match

	for _, tp := range r.tools {
		var conf float64
		var suggested string

		if tp.strong != nil && tp.strong.MatchString(msg) {
			// For mesh_send: strong verb alone is not enough — also require a node name
			// for high confidence. Without a node name it's probably about inbox or general messaging.
			if tp.tool == "mesh_send" {
				if nodeNameRE.MatchString(msg) {
					conf = 0.95
				} else {
					// "send" without a node name is likely inbox-related or ambiguous.
					conf = 0.55
				}
			} else {
				conf = 0.92
			}
			suggested = tp.tool
		} else if tp.weak != nil && tp.weak.MatchString(msg) {
			if tp.tool == "mesh_send" && nodeNameRE.MatchString(msg) {
				// Node name present but no send verb — probably a message intent.
				conf = 0.65
			} else {
				conf = 0.55
			}
			suggested = tp.tool
		}

		if conf > best.confidence {
			best = match{tool: suggested, confidence: conf}
		}
	}

	// If we have a confident tool match, return it.
	if best.confidence >= 0.6 {
		return ClassifyResult{
			ShouldUseTool:  true,
			Confidence:     best.confidence,
			SuggestedTools: []string{best.tool},
		}
	}

	// Check for clear conversation patterns — high-confidence no-tool.
	for _, cp := range r.convos {
		if cp.re.MatchString(msg) {
			return ClassifyResult{
				ShouldUseTool: false,
				Confidence:    0.92,
			}
		}
	}

	// Unrecognised: low confidence, let the model decide with tools available.
	return ClassifyResult{
		ShouldUseTool: false,
		Confidence:    0.40,
	}
}
