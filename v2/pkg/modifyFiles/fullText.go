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
// The AI response is expected to be formatted with explicit BEGIN_OF_FILE and END_OF_FILE
// markers, matching the constants defined in prompt/generate.go.
// Example format:
// --- BEGIN_OF_FILE: /path/to/file1 ---
// {content for /path/to/file1}
// --- END_OF_FILE: /path/to/file1 ---
func ApplyFullTextChangesToFiles(fullTextResponse string) error {
	fullTextResponse = cleanAIMarkdown(fullTextResponse) // Use common markdown cleaner

	// Trim leading/trailing whitespace (including newlines) from the entire response.
	// This helps in handling potential preamble/postamble from the LLM that isn't part of the structured file content.
	fullTextResponse = strings.TrimSpace(fullTextResponse)

	fullTextPath := "/tmp/fullTextChanges.txt"
	err := os.WriteFile(fullTextPath, []byte(fullTextResponse), 0644)
	if err != nil {
		glog.Errorf("Failed to write full text response to %s: %v", fullTextPath, err)
		return fmt.Errorf("failed to write %s: %w", fullTextPath, err)
	}
	glog.V(2).Infof("Full text response written to %s", fullTextPath)

	remainingResponse := fullTextResponse
	foundAnyFile := false

	for {
		// Find the start of the next file block
		beginIndex := strings.Index(remainingResponse, utils.BeginMarkerPrefix)
		if beginIndex == -1 {
			break // No more begin markers found
		}

		// The path starts immediately after `beginMarkerPrefix`
		pathStartInRemaining := beginIndex + len(utils.BeginMarkerPrefix)

		// The path ends before `beginMarkerSuffix`
		pathEndInSegment := strings.Index(remainingResponse[pathStartInRemaining:], utils.BeginMarkerSuffix)
		if pathEndInSegment == -1 {
			glog.Warningf("Malformed BEGIN_OF_FILE marker: missing suffix %q near %q. Skipping remaining response.",
				utils.BeginMarkerSuffix, utils.TruncateString(remainingResponse[beginIndex:], 100))
			break // Malformed marker, cannot parse further
		}

		filePath := strings.TrimSpace(remainingResponse[pathStartInRemaining : pathStartInRemaining+pathEndInSegment])

		// Content starts immediately after the full begin marker
		contentStartIndex := pathStartInRemaining + pathEndInSegment + len(utils.BeginMarkerSuffix)

		// Construct the full end marker string for this specific file
		fullEndMarker := fmt.Sprintf("%s%s%s", utils.EndMarkerPrefix, filePath, utils.EndMarkerSuffix)

		// Search for the end marker in the portion of the response *after* the content started
		endIndexInContentSegment := strings.Index(remainingResponse[contentStartIndex:], fullEndMarker)

		// Robustness: try matching the end marker without the final trailing newline, as LLMs can sometimes omit it.
		// This must be a distinct marker string to ensure correct length calculation later.
		if endIndexInContentSegment == -1 {
			fullEndMarkerNoTrailingNewline := fmt.Sprintf("%s%s ---", utils.EndMarkerPrefix, filePath)
			endIndexInContentSegment = strings.Index(remainingResponse[contentStartIndex:], fullEndMarkerNoTrailingNewline)
			// If found, update `fullEndMarker` so its length is correct for advancing `remainingResponse`
			if endIndexInContentSegment != -1 {
				fullEndMarker = fullEndMarkerNoTrailingNewline
			}
		}

		if endIndexInContentSegment == -1 {
			glog.Warningf("Malformed or missing END_OF_FILE marker for %q. Expected %q or %q near %q. Skipping this file and remainder.",
				filePath,
				fmt.Sprintf("%s%s%s", utils.EndMarkerPrefix, filePath, utils.EndMarkerSuffix),
				fmt.Sprintf("%s%s ---", utils.EndMarkerPrefix, filePath),
				utils.TruncateString(remainingResponse[contentStartIndex:], 100))
			break // Cannot find end marker, break from loop
		}

		// Extract the file content
		fileContent := remainingResponse[contentStartIndex : contentStartIndex+endIndexInContentSegment]

		// The prompt generator adds newlines around content (e.g., `\n---BEGIN---\ncontent\n---END---\n`).
		// `os.WriteFile` will write exactly the extracted content. No `TrimSpace` here to preserve
		// legitimate leading/trailing blank lines or newlines within the actual file content.

		glog.V(2).Infof("Attempting to write %d bytes to file: %q", len(fileContent), filePath)
		glog.V(3).Infof("File content for %q (truncated): %q", filePath, utils.TruncateString(fileContent, 200))

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			glog.Warningf("File %q specified in AI response does not exist on disk. Creating it.", filePath)
			// For new files, 0644 permission is fine.
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
		foundAnyFile = true

		// Advance `remainingResponse` past the current file's block for the next iteration
		remainingResponse = remainingResponse[contentStartIndex+endIndexInContentSegment+len(fullEndMarker):]
	}

	if !foundAnyFile {
		glog.Warning("AI response for full text changes did not contain any correctly formatted file blocks.")
		// Consider if a hard error is necessary here depending on expected behavior.
		// For now, a warning is kept to allow partial success in case of malformed output.
		return fmt.Errorf("no valid file blocks found in AI response")
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