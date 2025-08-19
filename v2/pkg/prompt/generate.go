package prompt

import (
	"fmt"
	"strings"
)

const (
	additionalInstructions string = `Return the result in unified diff format. 
Do not include any introductory text, explanations, or other formatting outside the unified diff format.`
)

// GeneratePrompt constructs a complete AI prompt based on user input,
// file contents, and specific instructions for the AI.
//
// The prompt will contain:
// 1. The user input from the argument.
// 2. The full text of the files in the fileContents map, with start/end markers.
// 3. A specific instruction for the AI regarding the output format.
func GeneratePrompt(userInput string, fileContents map[string]string) string {
	var builder strings.Builder

	// 1. Add the user input
	builder.WriteString(userInput)
	builder.WriteString("\n") // Add a newline after user input for separation

	// 2. Add the full text of the files
	// Iterating through the map. The order of files in the prompt will depend on map iteration order.
	for filePath, content := range fileContents {
		builder.WriteString(fmt.Sprintf("\n--- Start of File: %s ---\n", filePath))
		builder.WriteString(content)
		// Ensure the last line of content has a newline if it doesn't already, to prevent
		// the file end marker from being on the same line.
		if !strings.HasSuffix(content, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("--- End of File: %s ---\n", filePath))
	}

	// 3. Add the instruction
	builder.WriteString("\n")                   // Add a newline before the instruction for clarity
	builder.WriteString(additionalInstructions) // Fixed typo: changed addadditionalInstructions to additionalInstructions

	return builder.String()
}
