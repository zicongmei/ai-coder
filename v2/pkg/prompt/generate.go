package prompt

import (
	"fmt"
	"strings"

	"github.com/golang/glog" // Import glog
	"github.com/zicongmei/ai-coder/v2/pkg/utils"
)

var (
	additionalInstructionsFullText string = `

Do not include any introductory text, explanations, or other formatting outside of these BEGIN/END blocks. 
Always return full text. Never return diff.
Ensure the ABSOLUTE file paths in the BEGIN/END markers match the requested files: 
`
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
		builder.WriteString(utils.BeginMarkerPrefix + filePath + utils.BeginMarkerSuffix)
		builder.WriteString(content)
		// // Ensure the last line of content has a newline if it doesn't already, to prevent
		// // the file end marker from being on the same line.
		// if !strings.HasSuffix(content, "\n") {
		// 	builder.WriteString("\n")
		// }
		builder.WriteString(utils.EndMarkerPrefix + filePath + utils.EndMarkerSuffix)
	}

	// 3. Add the instruction based on the requested output format
	if inplace {
		glog.V(3).Info("Appending additional instructions for AI output format.")
		builder.WriteString("\nIMPORTANT: Respond ONLY with the complete, modified content for each file, formatted exactly as follows, using the ABSOLUTE file paths provided:\n")
		allPaths := []string{}
		for filePath, _ := range fileContents {
			builder.WriteString(utils.BeginMarkerPrefix + filePath + utils.BeginMarkerSuffix)
			builder.WriteString(fmt.Sprintf("{content for %s}", filePath))
			builder.WriteString(utils.EndMarkerPrefix + filePath + utils.EndMarkerSuffix)
			allPaths = append(allPaths, filePath)
		}
		builder.WriteString("\n") // Add a newline before the instruction for clarity
		builder.WriteString(additionalInstructionsFullText)
		builder.WriteString(strings.Join(allPaths, ", "))

	}

	finalPrompt := builder.String()
	glog.V(1).Infof("Prompt generation complete. Final prompt length: %d bytes.", len(finalPrompt))
	// Log the full generated prompt only at a very high verbosity level, as it can be very large.
	glog.V(4).Infof("Full generated prompt content: %q", finalPrompt)

	return finalPrompt
}