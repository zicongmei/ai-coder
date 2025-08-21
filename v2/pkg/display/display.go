package display

import (
	"fmt"
	"html" // Import html package to escape content for display in browser
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

// SaveAndOpenAIResponseAsHTML saves the provided AI response (raw text, no specific format assumed)
// to an HTML file in /tmp and attempts to open it in the default web browser.
// The content is HTML-escaped and wrapped in <pre> tags for literal display,
// ensuring whitespace and newlines are preserved.
func SaveAndOpenAIResponseAsHTML(aiResponse string) error {
	glog.V(1).Info("Preparing to save raw AI response as HTML and open in browser.")

	// Generate a unique filename using a timestamp
	timestamp := time.Now().Format("20060102_150405") // YYYYMMDD_HHMMSS
	fileName := fmt.Sprintf("ai_raw_response_%s.html", timestamp)
	filePath := filepath.Join(os.TempDir(), fileName)

	// HTML escape the AI response to ensure it's displayed literally and doesn't break HTML structure.
	escapedResponse := html.EscapeString(aiResponse)

	// Format the content as a basic HTML page.
	// Using <pre> tags to preserve whitespace, newlines, and fixed-width font.
	// Basic CSS is included for better readability.
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>AI Coder Raw Response</title>
    <style>
        body { font-family: monospace; white-space: pre-wrap; word-wrap: break-word; background-color: #f0f0f0; padding: 20px; }
        pre { background-color: #ffffff; border: 1px solid #ccc; padding: 15px; overflow-x: auto; }
        h1 { color: #333; }
        .timestamp { font-size: 0.9em; color: #777; margin-bottom: 10px; }
    </style>
</head>
<body>
    <h1>AI Coder Raw Response</h1>
    <div class="timestamp">Generated: %s</div>
    <p>%s</p>
</body>
</html>`, time.Now().Format("January 2, 2006 at 3:04 PM MST"), escapedResponse)

	// Write the content to the file
	err := os.WriteFile(filePath, []byte(htmlContent), 0644)
	if err != nil {
		glog.Errorf("Failed to save raw AI response to HTML file %q: %v", filePath, err)
		return fmt.Errorf("failed to save AI response: %w", err)
	}
	glog.V(0).Infof("Raw AI response saved to %q", filePath)

	// Determine the command to open the file based on the operating system
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", filePath)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", filePath)
	case "windows": // Windows
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
