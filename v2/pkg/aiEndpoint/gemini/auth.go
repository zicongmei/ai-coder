package gemini

import (
	"os"

	"github.com/golang/glog"
)

// GetAPIKey retrieves the Gemini API key.
// It checks the GEMINI_API_KEY environment variable.
// If set, it returns the value. Otherwise, it returns an empty string,
// indicating that Application Default Credentials (ADC) should be used.
func GetAPIKey() string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		glog.V(1).Info("Using API key from GEMINI_API_KEY environment variable.")
		return apiKey
	}
	glog.V(1).Info("GEMINI_API_KEY not set. Attempting to use Application Default Credentials (ADC).")
	return "" // Empty string signals to use ADC
}