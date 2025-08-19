package aiEndpoint

// AIEngine defines the interface for interacting with an AI endpoint.
// Implementations of this interface will handle the specific communication
// details (e.g., HTTP requests, authentication) for different AI models
// or services.
type AIEngine interface {
	// SendPrompt sends a string prompt to the AI endpoint and returns
	// the AI's response as a string.
	// It should also return an error if the communication or AI processing fails.
	SendPrompt(prompt string) (string, error)
}