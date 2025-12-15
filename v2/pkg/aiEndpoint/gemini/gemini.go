package gemini

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/zicongmei/ai-coder/v2/pkg/aiEndpoint"
	"github.com/zicongmei/ai-coder/v2/pkg/utils"
	"google.golang.org/genai"
)

// Client implements the AIEngine interface for the Gemini AI.
type Client struct {
	client             *genai.Client
	modelName          string
	ctx                context.Context // Context for API calls
	enableGoogleSearch bool
}

// NewClient initializes a new Gemini AI client.
// It uses an API key from GEMINI_API_KEY environment variable if set,
// otherwise it attempts to use Application Default Credentials (ADC).
// The 'flash' parameter determines which model to use (gemini-pro-flash vs gemini-pro).
func NewClient(modelName string, enableGoogleSearch bool) (aiEndpoint.AIEngine, error) {
	ctx := context.Background()

	cfg := &genai.ClientConfig{
		HTTPOptions: genai.HTTPOptions{APIVersion: "v1beta"},
	}

	apiKey := GetAPIKey() // Use the auth.go function
	if apiKey != "" {
		cfg.APIKey = apiKey
		glog.V(1).Info("Gemini client initializing with API key.")
	} else {
		glog.V(1).Info("GEMINI_API_KEY not set. Attempting to use Application Default Credentials (ADC).")
	}

	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		glog.Errorf("Failed to create Gemini client: %v", err)
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	// The underlying genai client should ideally be closed, but the AIEngine interface
	// doesn't expose a Close method. For long-running applications, the client should
	// be managed at a higher level (e.g., in `main` function with `defer client.Close()`).
	glog.V(0).Info("Gemini client successfully created.")

	glog.V(0).Infof("Using %q model.", modelName)
	if enableGoogleSearch {
		glog.V(0).Info("Google Search tool enabled.")
	}

	return &Client{
		client:             client,
		modelName:          modelName,
		ctx:                ctx,
		enableGoogleSearch: enableGoogleSearch,
	}, nil
}

// SendPrompt sends a string prompt to the Gemini AI endpoint and returns
// the AI's response as a string.
func (c *Client) SendPrompt(prompt string) (string, error) {
	glog.V(1).Info("Sending prompt to Gemini AI...")
	glog.V(2).Infof("Prompt content (truncated): %q", utils.TruncateString(prompt, 200))

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
			},
			Role: "user",
		},
	}

	var config *genai.GenerateContentConfig
	if c.enableGoogleSearch {
		config = &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{GoogleSearch: &genai.GoogleSearch{}},
			},
		}
	}

	resp, err := c.client.Models.GenerateContent(c.ctx, c.modelName, contents, config)
	if err != nil {
		glog.Errorf("Failed to generate content from Gemini: %v, response: %v", err, resp.Text())
		return "", fmt.Errorf("failed to generate content from Gemini: %w", err)
	}

	result := resp.Text()
	if result == "" {
		glog.Warning("Gemini response was empty.")
	}

	glog.V(1).Infof("Received response from Gemini (length: %d).", len(result))
	glog.V(2).Infof("Full Gemini response (truncated): %q", utils.TruncateString(result, 200))

	return result, nil
}

// CountTokens estimates the number of tokens in the given prompt string using the Gemini model.
func (c *Client) CountTokens(prompt string) (int, error) {
	glog.V(1).Info("Counting tokens for prompt using Gemini model.")
	return CountTokens(c.ctx, c.client, c.modelName, prompt)
}