package display

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang/glog"
)

// SaveAndOpenDiffAsMarkdown saves the provided AI response (expected to be a unified diff)
// to a Markdown file in /tmp and attempts to open it in the default web browser.
// The diff content is wrapped in a Markdown code block with 'diff' highlighting.
func SaveAndOpenDiffAsMarkdown(aiResponse string) error {
	glog.V(1).Info("Preparing to save AI response as Markdown and open in browser.")

	// Generate a unique filename using a timestamp
	timestamp := time.Now().Format("20060102_150405") // YYYYMMDD_HHMMSS
	fileName := fmt.Sprintf("ai_response_diff_%s.md", timestamp)
	filePath := filepath.Join(os.TempDir(), fileName)

	// Format the content as a Markdown code block for diff highlighting
	markdownContent := fmt.Sprintf("```diff\n%s\n```\n", aiResponse)

	// Write the content to the file
	err := os.WriteFile(filePath, []byte(markdownContent), 0644)
	if err != nil {
		glog.Errorf("Failed to save AI response to Markdown file %q: %v", filePath, err)
		return fmt.Errorf("failed to save AI response: %w", err)
	}
	glog.V(0).Infof("AI response (unified diff) saved to %q", filePath)

	// Determine the command to open the file based on the operating system
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", filePath)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", filePath)
	case "windows": // Windows
		// Use "start" command with "/c" to run it in a new shell and then exit
		cmd = exec.Command("cmd", "/c", "start", filePath)
	default:
		glog.Warningf("Unsupported operating system for opening file in browser: %s. Please open %q manually.", runtime.GOOS, filePath)
		return nil // Not considered a critical error, so return nil
	}

	glog.V(1).Infof("Attempting to open %q in browser using command: %s", filePath, cmd.String())

	// Use Start() to open the file asynchronously, so the main program doesn't wait for the browser to close.
	err = cmd.Start()
	if err != nil {
		glog.Errorf("Failed to open file %q in browser: %v", filePath, err)
		return fmt.Errorf("failed to open file in browser: %w", err)
	}

	glog.V(0).Info("AI response file opened in browser (if supported and successful).")
	return nil
}