package gemini

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/google/generative-ai-go/genai" // Import genai
)

// CountTokens estimates the number of tokens in the given text using the provided Gemini model.
func CountTokens(ctx context.Context, model *genai.GenerativeModel, text string) (int, error) {
	glog.V(1).Info("Requesting token count for prompt.")
	resp, err := model.CountTokens(ctx, genai.Text(text))
	if err != nil {
		glog.Errorf("Failed to count tokens: %v", err)
		return 0, fmt.Errorf("failed to count tokens: %w", err)
	}
	// The log message "Prompt contains %d tokens." will be done in flow.go.
	return int(resp.TotalTokens), nil
}