package gemini

import (
	"testing"
)

// TestCountTokens_Integration performs real API calls to Gemini to count tokens.
// This is an integration test, not a pure unit test, as it depends on network
// connectivity and a valid GEMINI_API_KEY environment variable.
func TestCountTokens_Integration(t *testing.T) {
	// Check for API key early and skip if not set, providing a clear message.
	if GetAPIKey() == "" {
		t.Skip("GEMINI_API_KEY not set. Skipping integration test for token counting. Please set the environment variable to run this test.")
	}

	// Use NewClient to create the AI client with gemini-2.5-flash model.
	// Passing 'true' for 'flash' parameter selects "gemini-2.5-flash".
	aiEngine, err := NewClient("gemini-2.5-flash")
	if err != nil {
		t.Fatalf("Failed to create Gemini client using NewClient for test: %v", err)
	}

	tests := []struct {
		name          string
		prompt        string
		wantMinTokens int // Minimum expected tokens, as exact count can vary with real API
		wantErr       bool
	}{
		{
			name:          "Short English Prompt",
			prompt:        "Hello world!",
			wantMinTokens: 2, // "Hello" and "world!" should be at least 2 tokens
			wantErr:       false,
		},
		{
			name:          "Longer English Prompt",
			prompt:        "This is a longer sentence designed to test the token counting for a more substantial piece of text. It includes several words and punctuation marks, which should result in a higher token count.",
			wantMinTokens: 20, // Expect a reasonable count for this length
			wantErr:       false,
		},
		{
			name:          "Prompt with Numbers and Symbols",
			prompt:        "Code example: func add(a, b int) int { return a + b } // Version 1.0! 😎",
			wantMinTokens: 15, // Should tokenize code and symbols
			wantErr:       false,
		},
		{
			name:          "Empty Prompt",
			prompt:        "",
			wantMinTokens: 1,
			wantErr:       false,
		},
		{
			name:          "Prompt with Unicode Characters",
			prompt:        "你好，世界！👋 This includes some non-ASCII characters.",
			wantMinTokens: 10, // Unicode can affect tokenization
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the CountTokens method on the AIEngine interface returned by NewClient
			gotTokens, err := aiEngine.CountTokens(tt.prompt)

			if (err != nil) != tt.wantErr {
				t.Errorf("CountTokens() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr { // Only check token count if no error is expected
				if gotTokens < tt.wantMinTokens {
					t.Errorf("CountTokens() gotTokens = %v, expected at least %v for prompt: %q", gotTokens, tt.wantMinTokens, tt.prompt)
				}
			}
		})
	}
}

// TestDummy ensures 'go test' finds at least one test even if the integration test is skipped.
// This is a common pattern for tests that might be conditionally skipped.
func TestDummy(t *testing.T) {
	t.Log("This dummy test ensures 'go test' finds a test case if the integration test is skipped.")
}