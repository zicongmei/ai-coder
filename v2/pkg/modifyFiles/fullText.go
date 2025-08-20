package modifyFiles

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/zicongmei/ai-coder/v2/pkg/utils"
)

// ApplyFullTextChangesToFiles parses the AI response containing full text of modified files
// and writes the content to the respective files on disk.
// The AI response is expected to be in the format:
// == Begin of /path/to/file ==
// file content
// == End of /path/to/file ==
func ApplyFullTextChangesToFiles(fullTextResponse string) error {
	fullTextResponse = cleanAIMarkdown(fullTextResponse) // Use common markdown cleaner

	// Ensure the response is trimmed to avoid issues with leading/trailing newlines
	fullTextResponse = strings.TrimSpace(fullTextResponse)

	fullTextPath := "/tmp/fullTextChanges.txt"
	err := os.WriteFile(fullTextPath, []byte(fullTextResponse), 0644)
	if err != nil {
		glog.Errorf("Failed to write full text response to %s: %v", fullTextPath, err)
		return fmt.Errorf("failed to write %s: %v", fullTextPath, err)
	}
	glog.V(2).Infof("Full text response written to %s", fullTextPath)

	// Split the response by the file begin marker
	// This approach assumes that a file's content doesn't contain the begin marker itself.
	// We'll split by "== Begin of " and then process each block.
	// The first split might yield an empty string if the response starts with a marker.
	blocks := strings.Split(fullTextResponse, "== Begin of ")
	if len(blocks) <= 1 { // Need at least one "== Begin of " to have a file block
		if len(blocks) == 1 && strings.Contains(blocks[0], "== End of ") {
			// This could be a single file response without the initial split creating an empty block.
			// Proceed, but log a warning if it doesn't match the expected multi-block pattern.
			glog.Warningf("AI response for full text changes might be malformed or contain only one file without initial '== Begin of' split artifact.")
		} else {
			glog.Errorf("AI response for full text changes did not contain expected file markers or was empty after sanitization.")
			return fmt.Errorf("invalid full text response format: no file begin markers found")
		}
	}

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue // Skip empty blocks from initial split (e.g., before the first "== Begin of ")
		}

		// Each block should start with '/path/to/file ==\n' and end with '== End of /path/to/file =='
		// Find the first " ==" to get the file path.
		pathEndIndex := strings.Index(block, " ==\n")
		if pathEndIndex == -1 {
			glog.Warningf("Skipping malformed block: %s (no path end marker ' ==\\n')", utils.TruncateString(block, 200))
			continue
		}

		filePath := block[:pathEndIndex]
		contentStart := pathEndIndex + len(" ==\n")

		// Find the end marker
		endMarker := fmt.Sprintf("== End of %s ==\n", filePath) // Add newline for robustness
		contentEndIndex := strings.LastIndex(block, endMarker)

		if contentEndIndex == -1 {
			// Try without newline at the end of endMarker, in case LLM omits it
			endMarker = fmt.Sprintf("== End of %s ==", filePath)
			contentEndIndex = strings.LastIndex(block, endMarker)
			if contentEndIndex == -1 {
				glog.Warningf("Skipping malformed block for file %q: no end marker found. Block: %s", filePath, utils.TruncateString(block, 200))
				continue
			}
		}

		// Extract the actual file content
		fileContent := block[contentStart:contentEndIndex]
		// Trim any leading/trailing newlines that might be artifacts of the markers
		fileContent = strings.TrimSpace(fileContent)

		glog.V(2).Infof("Attempting to write %d bytes to file: %q", len(fileContent), filePath)

		// Ensure the file exists before writing. If not, maybe it's a new file?
		// For now, assume it's modifying existing files as per problem statement's context.
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			glog.Warningf("File %q specified in AI response does not exist on disk. Creating it.", filePath)
			// Decide if creating new files is desired. For now, let's allow it with a warning.
		} else if err != nil {
			glog.Errorf("Error checking file %q before writing: %v", filePath, err)
			return fmt.Errorf("error checking file %q: %w", filePath, err)
		}

		err = os.WriteFile(filePath, []byte(fileContent), 0644)
		if err != nil {
			glog.Errorf("Failed to write content to file %q: %v", filePath, err)
			return fmt.Errorf("failed to write content to file %q: %w", filePath, err)
		}
		glog.V(0).Infof("Successfully updated file: %q", filePath)
	}

	return nil
}

// cleanAIMarkdown removes markdown code block fences (```) from the beginning and end of a string.
// It's a defensive function in case the LLM includes them despite instructions.
func cleanAIMarkdown(response string) string {
	// Trim leading/trailing whitespace first
	response = strings.TrimSpace(response)

	lines := strings.Split(response, "\n")
	if len(lines) < 2 { // Not enough lines for a multi-line markdown block
		return response
	}

	// Check if the first line starts with ``` and the last line starts with ```
	// and there's more than just the fences.
	if strings.HasPrefix(lines[0], "```") && strings.HasPrefix(lines[len(lines)-1], "```") {
		// Attempt to remove the first and last line (the fences)
		// and join the rest.
		processedResponse := strings.Join(lines[1:len(lines)-1], "\n")
		return strings.TrimSpace(processedResponse) // Trim again in case content also has leading/trailing newlines
	}
	return response
}
