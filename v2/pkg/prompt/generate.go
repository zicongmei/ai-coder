package prompt

import (
	"fmt"
	"strings"

	"github.com/golang/glog" // Import glog
	"github.com/zicongmei/ai-coder/v2/pkg/utils"
)

const (
	additionalInstructions string = `
* Important:	
1. Return the result in unified diff format. 
2. Return only the unified diff, nothing else. Ensure the diff is clean in apply ready.
3. Do not include any introductory, explanations, or other text other than the unified diff format.`
)

// GeneratePrompt constructs a complete AI prompt based on user input,
// file contents, and specific instructions for the AI.
//
// The prompt will contain:
// 1. The user input from the argument.
// 2. The full text of the files in the fileContents map, with start/end markers.
// 3. A specific instruction for the AI regarding the output format.
func GeneratePrompt(userInput string, fileContents map[string]string, inplace bool) string {
	glog.V(1).Info("Starting prompt generation process.")
	glog.V(2).Infof("Received user input for prompt (truncated): %q", utils.TruncateString(userInput, 100))
	glog.V(2).Infof("Number of files provided for prompt generation: %d", len(fileContents))

	var builder strings.Builder

	// 1. Add the user input
	glog.V(3).Info("Appending user input to the prompt.")
	builder.WriteString(userInput)
	builder.WriteString("\n") // Add a newline after user input for separation

	// 2. Add the full text of the files
	// Iterating through the map. The order of files in the prompt will depend on map iteration order.
	for filePath, content := range fileContents {
		glog.V(2).Infof("Adding file %q (length: %d characters) to the prompt.", filePath, len(content))
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
	if inplace {
		glog.V(3).Info("Appending additional instructions for AI output format.")
		builder.WriteString("\n") // Add a newline before the instruction for clarity
		builder.WriteString(additionalInstructions)
	}

	finalPrompt := builder.String()
	glog.V(1).Infof("Prompt generation complete. Final prompt length: %d bytes.", len(finalPrompt))
	// Log the full generated prompt only at a very high verbosity level, as it can be very large.
	glog.V(4).Infof("Full generated prompt content: %q", finalPrompt)

	return finalPrompt
}
