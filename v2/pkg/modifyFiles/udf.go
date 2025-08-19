package modifyFiles

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// ApplyUnifiedDiff applies a unified diff string to the original content of a file.
// It returns the modified content or an error if the diff cannot be applied.
// This function uses the github.com/sergi/go-diff/diffmatchpatch library.
//
// The unifiedDiff string is expected to be in the standard Git-style unified diff format,
// starting with `--- a/path` and `+++ b/path` headers, followed by hunks.
func ApplyUnifiedDiff(originalContent string, unifiedDiff string) (string, error) {
	glog.V(1).Info("Attempting to apply unified diff.")
	glog.V(2).Infof("Original content length: %d", len(originalContent))
	glog.V(2).Infof("Unified diff length: %d", len(unifiedDiff))

	dmp := diffmatchpatch.New()

	patches, err := dmp.PatchFromText(unifiedDiff)
	if err != nil {
		glog.Errorf("Failed to parse unified diff: %v", err)
		return "", fmt.Errorf("failed to parse unified diff: %w", err)
	}

	if len(patches) == 0 {
		glog.V(1).Info("No patches found in the provided unified diff. Returning original content.")
		return originalContent, nil
	}

	glog.V(2).Infof("Parsed %d patches from the unified diff.", len(patches))

	results, success := dmp.PatchApply(patches, originalContent)

	allPatchesApplied := true
	for i, s := range success {
		if !s {
			allPatchesApplied = false
			glog.Errorf("Patch %d failed to apply successfully (possible text mismatch).", i)
			if i < len(patches) {
				glog.V(3).Infof("Failed patch %d details: %+v", i, patches[i])
			}
		}
	}

	if !allPatchesApplied {
		return "", fmt.Errorf("one or more patches failed to apply due to context mismatch or other issues")
	}

	finalContent := results[0]
	glog.V(1).Infof("Unified diff applied successfully.")

	return string(finalContent), nil
}

// parseUnifiedDiffString parses a multi-file unified diff string into a map
// where keys are file paths and values are the unified diff content for that specific file.
// It expects the diff to follow the standard `--- a/path` and `+++ b/path` headers.
// Each file's diff must start with `--- a/` and be immediately followed by `+++ b/`.
//
// Note: The extracted file paths (map keys) will retain the leading '/' if the path
// in the diff header (e.g., `a/path/to/file.txt`) implies it (i.e., `a/` is trimmed,
// leaving `/path/to/file.txt`). This means `ApplyChangesToFiles` will attempt to
// read these paths as absolute from the filesystem root or as intended by `os.ReadFile`.
func parseUnifiedDiffString(unifiedDiff string) (map[string]string, error) {
	fileDiffs := make(map[string]string)
	lines := strings.Split(unifiedDiff, "\n")

	currentFilePath := ""
	currentFileDiffBuilder := &strings.Builder{}

	// `inDiffBlock` indicates if we are currently parsing lines that belong to a specific file's diff.
	inDiffBlock := false

	for _, line := range lines {
		// Detect the start of a new file diff block
		if strings.HasPrefix(line, "--- a/") {
			// If we were already processing a file, save its collected diff before starting a new one.
			if inDiffBlock && currentFilePath != "" {
				fileDiffs[currentFilePath] = currentFileDiffBuilder.String()
				glog.V(3).Infof("Saved diff for %q (length: %d bytes).", currentFilePath, currentFileDiffBuilder.Len())
			}
			// Reset for the new file
			currentFileDiffBuilder.Reset()
			inDiffBlock = true // Now we are inside a diff block

			// Extract file path from "--- a/path/to/file.txt <timestamp>"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// The path is the second field, e.g., "a/path/to/file.txt"
				// Trim "a" prefix; this leaves a leading "/" if present in the original path.
				// E.g., "a/foo/bar.txt" becomes "/foo/bar.txt". "a//tmp/foo.txt" becomes "//tmp/foo.txt".
				currentFilePath = strings.TrimPrefix(parts[1], "a")
				if currentFilePath == "" || (len(currentFilePath) == 1 && currentFilePath[0] == '/') { // Handle cases like "--- a/" or "--- a//"
					glog.Warningf("Malformed '--- a/' line or empty path encountered: %q. Skipping this potential diff block.", line)
					currentFilePath = "" // Invalidate current file path
					inDiffBlock = false  // Treat as not in a valid diff block until a valid header is found
					continue             // Move to next line
				}
				glog.V(2).Infof("Detected diff for file: %q", currentFilePath)
			} else {
				glog.Warningf("Malformed '--- a/' line encountered: %q. Skipping this potential diff block.", line)
				currentFilePath = "" // Invalidate current file path
				inDiffBlock = false  // Treat as not in a valid diff block until a valid header is found
				continue             // Move to next line
			}
			currentFileDiffBuilder.WriteString(line + "\n") // Include this header in the per-file diff
		} else if strings.HasPrefix(line, "+++ b/") {
			// This line MUST immediately follow a '--- a/' line within a valid diff block
			if inDiffBlock && currentFilePath != "" {
				currentFileDiffBuilder.WriteString(line + "\n") // Include this header in the per-file diff
			} else {
				// This indicates a malformed diff (e.g., +++ b/ without preceding --- a/ or invalid path)
				glog.Warningf("Malformed '+++ b/' line or out of sequence: %q. Skipping.", line)
				// Do not append to builder, and potentially invalidate block if strict.
				// For now, let's keep inDiffBlock true, expecting it to be corrected by next --- a/ or end of string.
			}
		} else if inDiffBlock && currentFilePath != "" {
			// Append all other lines (hunks, context, etc.) if we are inside a valid diff block
			currentFileDiffBuilder.WriteString(line + "\n")
		} else {
			// Lines outside any recognized diff block (e.g., blank lines, introductory text) are ignored.
			glog.V(3).Infof("Ignoring line outside of recognized diff block: %q", line)
		}
	}

	// After the loop, add the last file's diff if any was being processed
	if inDiffBlock && currentFilePath != "" {
		fileDiffs[currentFilePath] = currentFileDiffBuilder.String()
		glog.V(3).Infof("Saved final diff for %q (length: %d bytes).", currentFilePath, currentFileDiffBuilder.Len())
	}

	if len(fileDiffs) == 0 {
		glog.V(1).Info("No individual file diffs successfully extracted from the unified diff string.")
		return nil, nil // Return empty map and no error if no diffs found
	}

	glog.V(1).Infof("Successfully parsed %d individual file diffs from the unified diff string.", len(fileDiffs))
	return fileDiffs, nil
}

// ApplyChangesToFiles takes a single unifiedDiff string, parses it to identify
// changes for individual files, reads the original content of those files from disk,
// applies the diffs, and writes the modified content back to disk.
func ApplyChangesToFiles(unifiedDiff string) error {
	glog.V(1).Info("Starting to apply changes to files on disk from unified diff string.")

	// Parse the single unified diff string into a map of file paths to their individual diff content.
	fileDiffs, err := parseUnifiedDiffString(unifiedDiff)
	if err != nil {
		glog.Errorf("Failed to parse the multi-file unified diff string: %v", err)
		return fmt.Errorf("failed to parse unified diff: %w", err)
	}

	if len(fileDiffs) == 0 {
		glog.V(1).Info("No individual file diffs extracted from the unified diff string. No files will be modified.")
		return nil
	}

	for filePath, diffContent := range fileDiffs {
		glog.V(2).Infof("Processing changes for file: %q", filePath)

		// 1. Read original content from disk
		originalContentBytes, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				glog.Errorf("Original file %q not found on disk. Cannot apply diff.", filePath)
				return fmt.Errorf("file %q not found: %w", filePath, err)
			}
			glog.Errorf("Failed to read original content of file %q: %v", filePath, err)
			return fmt.Errorf("failed to read file %q: %w", filePath, err)
		}
		originalContent := string(originalContentBytes)
		glog.V(3).Infof("Read %d bytes from %q.", len(originalContentBytes), filePath)

		// 2. Apply the diff
		newContent, err := ApplyUnifiedDiff(originalContent, diffContent)
		if err != nil {
			glog.Errorf("Failed to apply diff for file %q: %v", filePath, err)
			return fmt.Errorf("failed to apply diff for file %q: %w", filePath, err)
		}

		// 3. Write the new content back to the file
		glog.V(2).Infof("Writing new content to file: %q (length: %d bytes)", filePath, len(newContent))
		// Use 0644 for file permissions: read/write for owner, read-only for others.
		err = os.WriteFile(filePath, []byte(newContent), 0644)
		if err != nil {
			glog.Errorf("Failed to write modified content to file %q: %v", filePath, err)
			return fmt.Errorf("failed to write modified content to file %q: %w", filePath, err)
		}
		glog.V(1).Infof("Successfully applied changes and updated file: %q", filePath)
	}

	glog.V(1).Info("All specified changes applied to files on disk.")
	return nil
}