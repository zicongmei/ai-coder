package gemini

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"google.golang.org/genai"
)

// CountTokens estimates the number of tokens in the given text using the provided Gemini model.
func CountTokens(ctx context.Context, client *genai.Client, modelName string, text string) (int, error) {
	glog.V(1).Info("Requesting token count for prompt.")

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: text},
			},
			Role: "user",
		},
	}

	resp, err := client.Models.CountTokens(ctx, modelName, contents, nil)
	if err != nil {
		glog.Errorf("Failed to count tokens: %v", err)
		return 0, fmt.Errorf("failed to count tokens: %w", err)
	}
	// The log message "Prompt contains %d tokens." will be done in flow.go.
	return int(resp.TotalTokens), nil
}