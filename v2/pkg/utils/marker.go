package utils

// Define the exact markers based on prompt/generate.go's fileHeader and fileFooter
const BeginMarkerPrefix = "--- BEGIN_OF_FILE:"
const BeginMarkerSuffix = " ---\n"            // From prompt/generate.go: `\n--- BEGIN_OF_FILE: %s ---\n`
const EndMarkerPrefix = "\n--- END_OF_FILE: " // From prompt/generate.go: `\n--- END_OF_FILE: %s ---\n`
const EndMarkerSuffix = " ---\n"              // From prompt/generate.go: `\n--- END_OF_FILE: %s ---\n`
