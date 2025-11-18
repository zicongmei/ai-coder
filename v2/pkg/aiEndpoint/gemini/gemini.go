package gemini

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/google/generative-ai-go/genai"
	"github.com/zicongmei/ai-coder/v2/pkg/aiEndpoint"
	"github.com/zicongmei/ai-coder/v2/pkg/utils"
	"google.golang.org/api/option"
)

// Client implements the AIEngine interface for the Gemini AI.
type Client struct {
	model *genai.GenerativeModel
	ctx   context.Context // Context for API calls
}

// NewClient initializes a new Gemini AI client.
// It uses an API key from GEMINI_API_KEY environment variable if set,
// otherwise it attempts to use Application Default Credentials (ADC).
// The 'flash' parameter determines which model to use (gemini-pro-flash vs gemini-pro).
func NewClient(modelName string) (aiEndpoint.AIEngine, error) {
	ctx := context.Background()
	var opts []option.ClientOption

	apiKey := GetAPIKey() // Use the auth.go function
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
		glog.V(1).Info("Gemini client initializing with API key.")
	} else {
		glog.V(1).Info("GEMINI_API_KEY not set. Attempting to use Application Default Credentials (ADC).")
	}

	client, err := genai.NewClient(ctx, opts...)
	if err != nil {
		glog.Errorf("Failed to create Gemini client: %v", err)
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	// The underlying genai client should ideally be closed, but the AIEngine interface
	// doesn't expose a Close method. For long-running applications, the client should
	// be managed at a higher level (e.g., in `main` function with `defer client.Close()`).
	glog.V(0).Info("Gemini client successfully created.")

	glog.V(0).Infof("Using %q model.", modelName)

	model := client.GenerativeModel(modelName)

	return &Client{
		model: model,
		ctx:   ctx,
	}, nil
}

// SendPrompt sends a string prompt to the Gemini AI endpoint and returns
// the AI's response as a string.
func (c *Client) SendPrompt(prompt string) (string, error) {
	glog.V(1).Info("Sending prompt to Gemini AI...")
	glog.V(2).Infof("Prompt content (truncated): %q", utils.TruncateString(prompt, 200))

	resp, err := c.model.GenerateContent(c.ctx, genai.Text(prompt))
	if err != nil {
		glog.Errorf("Failed to generate content from Gemini: %v", err)
		return "", fmt.Errorf("failed to generate content from Gemini: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 {
		glog.Warning("Gemini response was empty or contained no candidates.")
		return "", fmt.Errorf("Gemini returned an empty response or no candidates")
	}

	// Concatenate all parts from the first candidate
	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			sb.WriteString(string(text))
		} else {
			glog.Warningf("Received non-text part in Gemini response: %T", part)
		}
	}

	result := sb.String()
	glog.V(1).Infof("Received response from Gemini (length: %d).", len(result))
	glog.V(2).Infof("Full Gemini response (truncated): %q", utils.TruncateString(result, 200))

	return result, nil
}

// CountTokens estimates the number of tokens in the given prompt string using the Gemini model.
func (c *Client) CountTokens(prompt string) (int, error) {
	glog.V(1).Info("Counting tokens for prompt using Gemini model.")
	return CountTokens(c.ctx, c.model, prompt)
}
