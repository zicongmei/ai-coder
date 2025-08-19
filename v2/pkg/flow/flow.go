package flow

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time" // Import the time package for timestamps

	"github.com/golang/glog"
	"github.com/zicongmei/ai-coder/v2/pkg/aiEndpoint/gemini" // Assuming Gemini is the chosen AI engine
	"github.com/zicongmei/ai-coder/v2/pkg/modifyFiles"
	"github.com/zicongmei/ai-coder/v2/pkg/prompt"
	"github.com/zicongmei/ai-coder/v2/pkg/utils" // For TruncateString
)

// Run executes the main AI coding flow.
// It creates a prompt, sends it to the AI, and then either modifies files in-place
// or prints the AI's response (unified diff) to stdout.
func Run(fileListPath, userInputPrompt string, flashMode, inplace bool) error {
	glog.V(0).Info("Starting AI coding flow.")
	glog.V(1).Infof("File List Path: %q", fileListPath)
	glog.V(1).Infof("User Prompt (truncated): %q", utils.TruncateString(userInputPrompt, 100))
	glog.V(1).Infof("Flash Mode: %t, In-place: %t", flashMode, inplace)

	// 1. Read files and their contents
	fileContents, err := readFiles(fileListPath)
	if err != nil {
		glog.Errorf("Failed to read files from list %q: %v", fileListPath, err)
		return fmt.Errorf("failed to read files: %w", err)
	}
	glog.V(1).Infof("Successfully read %d files for prompt generation.", len(fileContents))

	// 2. Create the prompt
	fullPrompt := prompt.GeneratePrompt(userInputPrompt, fileContents, inplace)
	glog.V(1).Infof("Prompt generated. Total length: %d bytes.", len(fullPrompt))
	glog.V(2).Infof("Full generated prompt (truncated): %q", utils.TruncateString(fullPrompt, 500))

	// Generate dynamic file names based on current timestamp
	timestamp := time.Now().Format("20060102_150405") // YYYYMMDD_HHMMSS
	promptDumpFileName := fmt.Sprintf("ai_prompt_%s.txt", timestamp)
	rawOutputDumpFileName := fmt.Sprintf("ai_raw_output_%s.txt", timestamp)

	promptDumpPath := filepath.Join(os.TempDir(), promptDumpFileName)
	rawOutputDumpPath := filepath.Join(os.TempDir(), rawOutputDumpFileName)

	// Save the generated prompt to a file in /tmp
	err = os.WriteFile(promptDumpPath, []byte(fullPrompt), 0644)
	if err != nil {
		glog.Errorf("Failed to save generated prompt to %q: %v", promptDumpPath, err)
		// Do not return error, proceed with AI call as saving is a secondary feature.
	} else {
		glog.V(0).Infof("Generated AI prompt saved to %q", promptDumpPath)
	}

	// 3. Send the prompt to the AI endpoint
	aiEngine, err := gemini.NewClient(flashMode) // Assuming gemini is the only AI engine for now
	if err != nil {
		glog.Errorf("Failed to initialize AI engine: %v", err)
		return fmt.Errorf("failed to initialize AI engine: %w", err)
	}

	aiResponse, err := aiEngine.SendPrompt(fullPrompt)
	if err != nil {
		glog.Errorf("Failed to get response from AI: %v", err)
		return fmt.Errorf("failed to get AI response: %w", err)
	}
	glog.V(1).Infof("AI responded. Response length: %d bytes.", len(aiResponse))
	glog.V(2).Infof("Full AI response (truncated): %q", utils.TruncateString(aiResponse, 500))

	// Save the raw AI output to a file in /tmp
	err = os.WriteFile(rawOutputDumpPath, []byte(aiResponse), 0644)
	if err != nil {
		glog.Errorf("Failed to save raw AI output to %q: %v", rawOutputDumpPath, err)
		// Do not return error, proceed with modification/display as saving is a secondary feature.
	} else {
		glog.V(0).Infof("Raw AI output saved to %q", rawOutputDumpPath)
	}

	// 4. Modify files or show response
	if inplace {
		glog.V(0).Info("In-place modification requested. Applying changes to files.")
		err = modifyFiles.ApplyChangesToFiles(aiResponse)
		if err != nil {
			glog.Errorf("Failed to apply changes to files in-place: %v", err)
			return fmt.Errorf("failed to apply changes: %w", err)
		}
		glog.V(0).Info("Files modified successfully in-place.")
	} else {
		glog.V(0).Info("In-place modification not requested. Displaying AI response (unified diff):")
		fmt.Println(aiResponse)
		glog.V(0).Info("AI response displayed.")
	}

	glog.V(0).Info("AI coding flow completed.")
	return nil
}

// readFiles reads the file paths from the given file list path
// and then reads the content of each file, returning a map of file paths to their content.
func readFiles(fileListPath string) (map[string]string, error) {
	glog.V(1).Infof("Reading file list from: %q", fileListPath)
	filePaths := []string{}

	// Open the file list file
	file, err := os.Open(fileListPath)
	if err != nil {
		glog.Errorf("Failed to open file list %q: %v", fileListPath, err)
		return nil, fmt.Errorf("failed to open file list: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" { // Ignore empty lines
			filePaths = append(filePaths, line)
		}
	}

	if err := scanner.Err(); err != nil {
		glog.Errorf("Error reading file list %q: %v", fileListPath, err)
		return nil, fmt.Errorf("error reading file list: %w", err)
	}
	glog.V(1).Infof("Found %d files in the file list.", len(filePaths))

	// Read content of each file
	fileContents := make(map[string]string)
	for _, path := range filePaths {
		glog.V(2).Infof("Reading content of file: %q", path)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			// Log the error but continue if possible, or decide to fail fast.
			// For now, fail fast as missing files are critical for prompt generation.
			glog.Errorf("Failed to read content of file %q: %v", path, err)
			return nil, fmt.Errorf("failed to read file %q: %w", path, err)
		}
		fileContents[path] = string(contentBytes)
		glog.V(3).Infof("Read %d bytes from %q.", len(contentBytes), path)
	}

	return fileContents, nil
}
