// Package classifier provides intent classification for agent messages.
// It determines whether a user message requires tool use before the message
// reaches the LLM, enabling intent-based model routing and tool injection.
//
// The Classifier interface is designed to accommodate rule-based, ML-based,
// and fine-tuned model implementations without contract changes. The
// RuleClassifier shipped here is bootstrap scaffolding — it handles DUSTY's
// 3-tool schema reliably and collects the training data needed to replace
// itself with a fine-tuned model.
package classifier

import "context"

// ClassifyResult holds the output of a classification call.
// ShouldUseTool is the only field the agent must act on; Confidence and
// SuggestedTools are consumed by the router for model selection and are
// available to future fine-tuned implementations that can predict specific tools.
type ClassifyResult struct {
	// ShouldUseTool is true when the classifier believes the message requires
	// at least one tool call. False means respond with plain text only.
	ShouldUseTool bool

	// Confidence is a value in [0.0, 1.0] indicating how certain the classifier
	// is. Values below the configured threshold cause the agent to fall back to
	// the default model with tools enabled (conservative).
	Confidence float64

	// SuggestedTools is an optional list of tool names the classifier believes
	// are relevant to this message. Empty means "let the model decide."
	// Rule-based classifiers populate this; fine-tuned models will too.
	SuggestedTools []string
}

// Classifier categorises a user message before it reaches the LLM.
// Implementations must be safe for concurrent use.
type Classifier interface {
	Classify(ctx context.Context, message string) ClassifyResult
}
