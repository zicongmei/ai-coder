package modifyFiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zicongmei/ai-coder/v2/pkg/modifyFiles" // Import the package under test
)

// setupTestDir creates a temporary directory and populates it with initial files for testing.
// It returns the path to the temporary directory and a cleanup function.
func setupTestDir(t *testing.T, initialFiles map[string]string) (string, func()) {
	tempDir, err := os.MkdirTemp("", "modifyfiles_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	for name, content := range initialFiles {
		filePath := filepath.Join(tempDir, name)
		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			os.RemoveAll(tempDir) // Clean up on error
			t.Fatalf("Failed to create parent dirs for %s: %v", filePath, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			os.RemoveAll(tempDir) // Clean up on error
			t.Fatalf("Failed to write file %s: %v", filePath, err)
		}
	}

	return tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

// readFileContent reads the content of a file relative to the test directory.
func readFileContent(t *testing.T, dir, filename string) string {
	content, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	return string(content)
}

func TestApplyChangesToFiles(t *testing.T) {
	tests := []struct {
		name          string
		initialFiles  map[string]string // File name -> initial content
		unifiedDiff   string
		expectedFiles map[string]string // File name -> expected content (nil if expected to be deleted)
		expectError   bool
	}{
		{
			name: "Modify existing file",
			initialFiles: map[string]string{
				"file1.txt": "line 1\nline 2\nline 3\n",
			},
			unifiedDiff: `--- a/file1.txt
+++ b/file1.txt
@@ -1,3 +1,4 @@
 line 1
+new line 1.5
 line 2
 line 3
`,
			expectedFiles: map[string]string{
				"file1.txt": "line 1\nnew line 1.5\nline 2\nline 3\n",
			},
			expectError: false,
		},
		{
			name: "Diff for non-existent file (not an add)",
			initialFiles: map[string]string{
				"some_other.txt": "content",
			},
			unifiedDiff: `--- a/non_existent.txt
+++ b/non_existent.txt
@@ -1,3 +1,4 @@
 line 1
+new line 1.5
 line 2
 line 3
`,
			expectedFiles: map[string]string{
				"some_other.txt": "content", // Should remain untouched
			},
			expectError: true, // Should error because non_existent.txt is not present and it's not a /dev/null diff
		},
		{
			name: "Malformed diff",
			initialFiles: map[string]string{
				"file.txt": "original",
			},
			unifiedDiff: `--- a/non_existent.txt
+++ b/non_existent.txt
@@ -1,3 +1,4 @@
 line 1
+new line 1.5
 line 2
 line 3`,
			expectedFiles: map[string]string{
				"file.txt": "original", // Should remain untouched
			},
			expectError: true,
		},
		{
			name: "Diff wrapped in backticks",
			initialFiles: map[string]string{
				"wrapped.txt": "initial line\n",
			},
			unifiedDiff: "```diff\n--- a/wrapped.txt\n+++ b/wrapped.txt\n@@ -1 +1,2 @@\n initial line\n+second line\n```",
			expectedFiles: map[string]string{
				"wrapped.txt": "initial line\nsecond line",
			},
			expectError: false,
		},
		{
			name: "Empty diff wrapped in backticks",
			initialFiles: map[string]string{
				"emptywrap.txt": "initial line\n",
			},
			unifiedDiff: "```\n```", // After sanitize, this is just an empty string
			expectedFiles: map[string]string{
				"emptywrap.txt": "initial line\n",
			},
			expectError: false, // Empty string diff should parse to no files, thus no changes, no error.
		},
		{
			name: "Diff with empty content after sanitization (code fence with no diff)",
			initialFiles: map[string]string{
				"file.txt": "original",
			},
			unifiedDiff: "```diff\n```", // After sanitize, this is just an empty string
			expectedFiles: map[string]string{
				"file.txt": "original",
			},
			expectError: false, // Empty string diff should not cause an error, just no changes.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, cleanup := setupTestDir(t, tt.initialFiles)
			defer cleanup()

			// Change current working directory to the tempDir so os.Open/WriteFile work relative to it
			originalCwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get original working directory: %v", err)
			}
			err = os.Chdir(dir)
			if err != nil {
				t.Fatalf("Failed to change to test directory %s: %v", dir, err)
			}
			defer os.Chdir(originalCwd) // Restore original working directory

			err = modifyFiles.ApplyChangesToFiles(tt.unifiedDiff)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none")
				}
				// For error cases, we stop further checks as file state might be undefined or partial.
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file contents and existence based on expectedFiles
			for filename, expectedContent := range tt.expectedFiles {
				fullPath := filepath.Join(dir, filename)
				_, statErr := os.Stat(fullPath)

				if expectedContent == "" { // File expected to be deleted
					if !os.IsNotExist(statErr) {
						t.Errorf("Expected file %s to be deleted, but it still exists (err: %v)", filename, statErr)
					}
				} else { // File expected to exist with specific content
					if os.IsNotExist(statErr) {
						t.Errorf("Expected file %s to exist, but it was not found", filename)
						continue
					}
					actualContent := readFileContent(t, dir, filename)
					if actualContent != expectedContent {
						t.Errorf("File %s content mismatch:\nExpected:\n%q\nActual:\n%q", filename, expectedContent, actualContent)
					}
				}
			}

			// Verify no *unexpected* files were created or old ones left behind.
			// Get all files that actually exist in the temp directory.
			actualExistingFiles := make(map[string]struct{})
			err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() && path != dir { // Don't add the root tempDir itself, but allow subdirectories
					return nil
				}
				if !info.IsDir() { // Only consider files
					relPath, err := filepath.Rel(dir, path)
					if err != nil {
						return err
					}
					actualExistingFiles[relPath] = struct{}{}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("Failed to walk test directory to verify final state: %v", err)
			}

			// Compare actual existing files with expected existing files
			expectedExistingFiles := make(map[string]struct{})
			for filename, content := range tt.expectedFiles {
				if content != "" { // Only add files that are expected to exist (not nil/deleted)
					expectedExistingFiles[filename] = struct{}{}
				}
			}

			if len(actualExistingFiles) != len(expectedExistingFiles) {
				t.Errorf("Mismatch in number of files. Expected %d files, Got %d files.\nExpected: %v\nActual:   %v",
					len(expectedExistingFiles), len(actualExistingFiles), expectedExistingFiles, actualExistingFiles)
			}

			for f := range actualExistingFiles {
				if _, ok := expectedExistingFiles[f]; !ok {
					t.Errorf("Unexpected file %q found in test directory", f)
				}
			}
			for f := range expectedExistingFiles {
				if _, ok := actualExistingFiles[f]; !ok {
					t.Errorf("Expected file %q not found in test directory", f)
				}
			}
		})
	}
}
