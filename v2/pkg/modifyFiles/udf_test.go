package modifyFiles

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/glog"
)

func TestMain(m *testing.M) {
	// Parse glog flags to allow environment variables (e.g., GLOG_v, GLOG_logtostderr) to control logging.
	flag.Parse()
	// Set verbosity level to 0 by default to suppress V-level logs during tests.
	// Can be overridden by GLOG_v environment variable (e.g., GLOG_v=2).
	// glog.SetV(0)
	os.Exit(m.Run())
}

// Helper to create a temporary file with content
func createTempFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	// Ensure directory exists for the file
	fileDir := filepath.Dir(filePath)
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		err = os.MkdirAll(fileDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", fileDir, err)
		}
	}
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file %s: %v", filePath, err)
	}
	return filePath
}

// Helper to read file content
func readFileContent(t *testing.T, filePath string) string {
	t.Helper()
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath, err)
	}
	return string(contentBytes)
}

func TestApplyUnifiedDiff(t *testing.T) {
	tests := []struct {
		name            string
		originalContent string
		unifiedDiff     string
		expectedContent string
		expectError     bool
	}{
		{
			name:            "Basic addition",
			originalContent: "line1\nline2\nline3\n",
			unifiedDiff: `@@ -1,3 +1,4 @@
 line1
+new_line
 line2
 line3
`,
			expectedContent: "line1\nnew_line\nline2\nline3\n",
			expectError:     false,
		},
		{
			name:            "Basic deletion",
			originalContent: "line1\nline2\nline3\nline4\n",
			unifiedDiff: `@@ -1,4 +1,3 @@
 line1
-line2
 line3
 line4
`,
			expectedContent: "line1\nline3\nline4\n",
			expectError:     false,
		},
		{
			name:            "Basic modification",
			originalContent: "line1\nold_line\nline3\n",
			unifiedDiff: `@@ -1,3 +1,3 @@
 line1
-old_line
+new_line
 line3
`,
			expectedContent: "line1\nnew_line\nline3\n",
			expectError:     false,
		},
		{
			name:            "Multiple hunks",
			originalContent: "line1\nline2\nline3\nline4\nline5\nline6\n",
			unifiedDiff: `@@ -1,2 +1,3 @@
 line1
+added_line_a
 line2
@@ -4,3 +5,4 @@
 line4
 line5
+added_line_b
 line6
`,
			expectedContent: "line1\nadded_line_a\nline2\nline3\nline4\nline5\nadded_line_b\nline6\n",
			expectError:     false,
		},
		{
			name:            "Empty original content",
			originalContent: "",
			unifiedDiff: `@@ -0,0 +1,3 @@
+line1
+line2
+line3
`,
			expectedContent: "line1\nline2\nline3\n",
			expectError:     false,
		},
		{
			name:            "Empty diff (no changes)",
			originalContent: "line1\nline2\n",
			unifiedDiff:     ``, // An empty diff usually doesn't have hunks, or just headers. dmp.PatchFromText might return 0 patches.
			expectedContent: "line1\nline2\n",
			expectError:     false,
		},
		{
			name:            "Diff with context mismatch (will fail to apply)",
			originalContent: "lineA\nlineB\nlineC\n",
			unifiedDiff: `@@ -1,3 +1,3 @@
 line1
-line2
+lineX
 line3
`, // Context "line1, line3" doesn't match "lineA, lineC"
			expectedContent: "", // Should not return partial content
			expectError:     true,
		},
		{
			name:            "Malformed diff string (invalid header)",
			originalContent: "content",
			unifiedDiff:     "invalid diff format",
			expectedContent: "",
			expectError:     true,
		},
		{
			name:            "Malformed diff string (hunk header error)",
			originalContent: "line1\nline2\n",
			unifiedDiff: `@@ -1,2 +1,2 @@
+invalid hunk
`, // Hunk line starts with '+' but doesn't have context
			expectedContent: "",
			expectError:     true,
		},
		{
			name:            "Add to end of file without newline",
			originalContent: "line1\nline2",
			unifiedDiff: `@@ -1,2 +1,3 @@
 line1
 line2
+line3
`,
			expectedContent: "line1\nline2\nline3\n",
			expectError:     false,
		},
		{
			name:            "Remove last line",
			originalContent: "line1\nline2\nline3\n",
			unifiedDiff: `@@ -1,3 +1,2 @@
 line1
 line2
-line3
`,
			expectedContent: "line1\nline2\n",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualContent, err := ApplyUnifiedDiff(tt.originalContent, tt.unifiedDiff)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none for test %q. Actual content: %q", tt.name, actualContent)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error but got: %v for test %q", err, tt.name)
				}
				if actualContent != tt.expectedContent {
					t.Errorf("Content mismatch for test %q.\nExpected:\n%q\nActual:\n%q", tt.name, tt.expectedContent, actualContent)
				}
			}
		})
	}
}

func TestParseUnifiedDiffString(t *testing.T) {
	tests := []struct {
		name        string
		unifiedDiff string
		expectedMap map[string]string
		expectError bool // This field is for consistency; parseUnifiedDiffString returns nil, nil for no diffs, not an error.
	}{
		{
			name: "Single file diff",
			unifiedDiff: `--- a/file1.txt
+++ b/file1.txt
@@ -1,2 +1,2 @@
 line1
-line2
+new_line2
`,
			expectedMap: map[string]string{
				"/file1.txt": `@@ -1,2 +1,2 @@
line1
-line2
+new_line2
`,
			},
			expectError: false,
		},
		{
			name: "Multi-file diff",
			unifiedDiff: `--- a/dir/file1.txt
+++ b/dir/file1.txt
@@ -1,2 +1,2 @@
 line1
-line2
+new_line2
--- a/dir/subdir/file2.go
+++ b/dir/subdir/file2.go
@@ -1,1 +1,2 @@
 package main
+import "fmt"
`,
			expectedMap: map[string]string{
				"/dir/file1.txt": `@@ -1,2 +1,2 @@
line1
-line2
+new_line2
`,
				"/dir/subdir/file2.go": `@@ -1,1 +1,2 @@
package main
+import "fmt"
`,
			},
			expectError: false,
		},
		{
			name:        "Empty diff string",
			unifiedDiff: "",
			expectedMap: nil, // parseUnifiedDiffString returns nil, nil for no diffs
			expectError: false,
		},
		{
			name:        "Diff string with only blank lines",
			unifiedDiff: "\n\n\n",
			expectedMap: nil,
			expectError: false,
		},
		{
			name: "Diff with no changes (headers only)",
			unifiedDiff: `--- a/file.txt
+++ b/file.txt
`,
			expectedMap: map[string]string{
				"/file.txt": ``, // Headers are removed, so empty string
			},
			expectError: false,
		},
		{
			name: "Diff with malformed --- a/ line (empty path)",
			unifiedDiff: `--- a/
+++ b/file.txt
@@ -1,1 +1,1 @@
-old
+new
`, // Path is empty after a/
			expectedMap: nil, // Should not extract any valid diff, as currentFilePath becomes ""
			expectError: false,
		},
		{
			name: "Diff with --- a/ but missing +++ b/",
			unifiedDiff: `--- a/file.txt
@@ -1,1 +1,1 @@
-old
+new
`,
			expectedMap: map[string]string{
				"/file.txt": `@@ -1,1 +1,1 @@
-old
+new
`,
			}, // It still captures the diff for the file, as long as it starts with --- a/ and has a path. The missing +++ b/ might lead to `expectingPlusHeader` staying true, but the hunk line will correctly be added.
			expectError: false,
		},
		{
			name: "Diff with +++ b/ but missing --- a/",
			unifiedDiff: `+++ b/file.txt
@@ -1,1 +1,1 @@
-old
+new
`,
			expectedMap: nil,
			expectError: false,
		},
		{
			name: "Multi-file diff with one malformed entry",
			unifiedDiff: `--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,1 @@
-old1
+new1
--- a/
+++ b/file2.txt
@@ -1,1 +1,1 @@
-old2
+new2
`,
			expectedMap: map[string]string{
				"/file1.txt": `@@ -1,1 +1,1 @@
-old1
+new1
`,
			}, // Only file1.txt should be parsed because the second '--- a/' is malformed
			expectError: false,
		},
		{
			name: "Diff with timestamp in header",
			unifiedDiff: `--- a/file.txt	2023-10-27 10:00:00.000000000 +0000
+++ b/file.txt	2023-10-27 10:00:00.000000000 +0000
@@ -1,1 +1,1 @@
-old
+new
`,
			expectedMap: map[string]string{
				"/file.txt": `@@ -1,1 +1,1 @@
-old
+new
`,
			},
			expectError: false,
		},
		{
			name: "Diff with no hunks, just headers and some empty lines",
			unifiedDiff: `--- a/file.txt
+++ b/file.txt


`,
			expectedMap: map[string]string{
				"/file.txt": `

`, // Empty lines after headers are still included
			},
			expectError: false,
		},
		{
			name:        "Diff with only --- a/ line and then nothing",
			unifiedDiff: `--- a/file.txt`,
			expectedMap: map[string]string{
				"/file.txt": ``, // No +++ b/ or hunks, so content is empty
			},
			expectError: false,
		},
		{
			name: "Diff with line between --- and +++ headers (malformed)",
			unifiedDiff: `--- a/file.txt
some garbage line
+++ b/file.txt
@@ -1,1 +1,1 @@
-old
+new
`,
			expectedMap: nil, // Should invalidate the block and not parse, as it's malformed
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualMap, err := parseUnifiedDiffString(tt.unifiedDiff)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none for test %q.", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error but got: %v for test %q", err, tt.name)
				}

				if len(actualMap) != len(tt.expectedMap) {
					t.Fatalf("Map size mismatch for test %q. Expected %d, got %d. Actual map: %+v", tt.name, len(tt.expectedMap), len(actualMap), actualMap)
				}

				for k, v := range tt.expectedMap {
					actualV, ok := actualMap[k]
					if !ok {
						t.Errorf("Expected key %q not found in actual map for test %q.", k, tt.name)
					}
					if actualV != v {
						t.Errorf("Content mismatch for key %q in test %q.\nExpected:\n%q\nActual:\n%q", k, tt.name, v, actualV)
					}
				}
			}
		})
	}
}

func TestApplyChangesToFiles(t *testing.T) {
	type testCaseApplyChanges struct {
		name              string
		getUnifiedDiff    func(t *testing.T, dir string) string            // Function to generate the diff string with correct paths
		getExpectedState  func(t *testing.T, dir string) map[string]string // Function to generate expected file contents
		expectError       bool
		initialFilesSetup func(t *testing.T, dir string) // Setup initial files before applying diff
	}

	applyTests := []testCaseApplyChanges{
		{
			name: "Single file modification",
			initialFilesSetup: func(t *testing.T, dir string) {
				createTempFile(t, dir, "file1.txt", "line1\nline2\nline3\n")
			},
			getUnifiedDiff: func(t *testing.T, dir string) string {
				// The paths in the diff headers (`a/path` and `b/path`) must be system-native absolute paths
				// that, after `parseUnifiedDiffString` processes them (e.g., `strings.TrimPrefix(path, "a")`),
				// become valid paths for `os.ReadFile`.
				// For Unix-like systems, `a//tmp/mytestdir/file.txt` becomes `//tmp/mytestdir/file.txt`, which is absolute.
				file1AbsPath := filepath.Join(dir, "file1.txt")
				return fmt.Sprintf(`--- a/%s
+++ b/%s
@@ -1,3 +1,3 @@
 line1
-line2
+modified_line2
 line3
`, file1AbsPath, file1AbsPath)
			},
			getExpectedState: func(t *testing.T, dir string) map[string]string {
				return map[string]string{
					filepath.Join(dir, "file1.txt"): "line1\nmodified_line2\nline3\n",
				}
			},
			// This test case is expected to fail now because parseUnifiedDiffString will remove headers,
			// and ApplyUnifiedDiff expects them for PatchFromText.
			expectError: true,
		},
		{
			name: "Multi-file modification",
			initialFilesSetup: func(t *testing.T, dir string) {
				createTempFile(t, dir, "file1.txt", "line1\nline2\nline3\n")
				createTempFile(t, filepath.Join(dir, "subdir"), "file2.go", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n")
			},
			getUnifiedDiff: func(t *testing.T, dir string) string {
				file1AbsPath := filepath.Join(dir, "file1.txt")
				file2AbsPath := filepath.Join(dir, "subdir", "file2.go")
				return fmt.Sprintf(`--- a/%s
+++ b/%s
@@ -1,3 +1,2 @@
 line1
-line2
 line3
--- a/%s
+++ b/%s
@@ -1,4 +1,5 @@
 package main
 
 func main() {
 	fmt.Println("Hello")
+	// new line added
 }
`, file1AbsPath, file1AbsPath, file2AbsPath, file2AbsPath)
			},
			getExpectedState: func(t *testing.T, dir string) map[string]string {
				return map[string]string{
					filepath.Join(dir, "file1.txt"):          "line1\nline3\n",
					filepath.Join(dir, "subdir", "file2.go"): "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n\t// new line added\n}\n",
				}
			},
			// This test case is expected to fail now because parseUnifiedDiffString will remove headers,
			// and ApplyUnifiedDiff expects them for PatchFromText.
			expectError: true,
		},
		{
			name: "Empty unified diff string",
			initialFilesSetup: func(t *testing.T, dir string) {
				createTempFile(t, dir, "file1.txt", "original content")
			},
			getUnifiedDiff: func(t *testing.T, dir string) string {
				return ""
			},
			getExpectedState: func(t *testing.T, dir string) map[string]string {
				return map[string]string{
					filepath.Join(dir, "file1.txt"): "original content", // Should remain unchanged
				}
			},
			expectError: false, // This case still works, as parseUnifiedDiffString returns nil map, resulting in no ops.
		},
		{
			name: "File not found during read",
			initialFilesSetup: func(t *testing.T, dir string) {
				// Don't create the file that the diff references
			},
			getUnifiedDiff: func(t *testing.T, dir string) string {
				return fmt.Sprintf(`--- a/%s/non_existent_file.txt
+++ b/%s/non_existent_file.txt
@@ -1,1 +1,1 @@
-old
+new
`, dir, dir)
			},
			getExpectedState: func(t *testing.T, dir string) map[string]string {
				return nil // No expected file state, as it should error out immediately
			},
			expectError: true,
		},
		{
			name: "Diff application failure (context mismatch)",
			initialFilesSetup: func(t *testing.T, dir string) {
				createTempFile(t, dir, "file1.txt", "actual line1\nactual line2\nactual line3\n")
			},
			getUnifiedDiff: func(t *testing.T, dir string) string {
				return fmt.Sprintf(`--- a/%s/file1.txt
+++ b/%s/file1.txt
@@ -1,3 +1,3 @@
 line1
-line2
+modified_line2
 line3
`, dir, dir) // Diff has different context than actual file
			},
			getExpectedState: func(t *testing.T, dir string) map[string]string {
				// File content should remain its initial state if diff fails to apply.
				return map[string]string{
					filepath.Join(dir, "file1.txt"): "actual line1\nactual line2\nactual line3\n",
				}
			},
			// This test case will also now result in an error from `PatchFromText` even before context mismatch,
			// because the headers are removed by `parseUnifiedDiffString`.
			expectError: true,
		},
		{
			name: "Malformed unified diff string (parseUnifiedDiffString will return nil)",
			initialFilesSetup: func(t *testing.T, dir string) {
				createTempFile(t, dir, "file1.txt", "original content")
			},
			getUnifiedDiff: func(t *testing.T, dir string) string {
				return `--- a/file1.txt
invalid hunk header line
` // This diff cannot be parsed correctly into patches or has no valid file headers.
			},
			getExpectedState: func(t *testing.T, dir string) map[string]string {
				return map[string]string{
					filepath.Join(dir, "file1.txt"): "original content", // Should remain unchanged
				}
			},
			// This test is already expected to error out during parseUnifiedDiffString or ApplyUnifiedDiff.
			// With headers removed, ApplyUnifiedDiff will definitely error.
			expectError: true,
		},
		{
			name: "Verify file permissions are 0644 after write",
			initialFilesSetup: func(t *testing.T, dir string) {
				// Create file with different permissions initially (e.g., more permissive)
				filePath := filepath.Join(dir, "perms.txt")
				err := os.WriteFile(filePath, []byte("initial content"), 0777) // Writable by all
				if err != nil {
					t.Fatalf("Failed to create file for perm test: %v", err)
				}
			},
			getUnifiedDiff: func(t *testing.T, dir string) string {
				return fmt.Sprintf(`--- a/%s/perms.txt
+++ b/%s/perms.txt
@@ -1,1 +1,1 @@
-initial content
+modified content
`, dir, dir)
			},
			getExpectedState: func(t *testing.T, dir string) map[string]string {
				return map[string]string{
					filepath.Join(dir, "perms.txt"): "modified content",
				}
			},
			// This test case is expected to fail now because parseUnifiedDiffString will remove headers,
			// and ApplyUnifiedDiff expects them for PatchFromText.
			expectError: true,
		},
	}

	for _, tt := range applyTests {
		t.Run(tt.name, func(t *testing.T) {
			testTempDir := t.TempDir() // New temp dir for each sub-test
			glog.V(5).Infof("Running sub-test %q in temp dir: %s", tt.name, testTempDir)

			// Store initial state before diff application for potential error cases
			initialFileContents := make(map[string]string)
			// Setup initial files
			tt.initialFilesSetup(t, testTempDir)

			// Capture initial file contents for comparison in error scenarios.
			// Iterate over the keys that `getExpectedState` would produce, to identify all files involved.
			for filePath := range tt.getExpectedState(t, testTempDir) {
				if _, err := os.Stat(filePath); err == nil { // Only read if file exists
					initialFileContents[filePath] = readFileContent(t, filePath)
				}
			}

			// Get dynamic unified diff and expected state for the current test run
			unifiedDiff := tt.getUnifiedDiff(t, testTempDir)
			expectedFileState := tt.getExpectedState(t, testTempDir)

			err := ApplyChangesToFiles(unifiedDiff)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error but got none for test %q.", tt.name)
				}
				// If an error is expected, verify files are unchanged if they existed initially.
				// For "file not found" cases, there's no content to verify as unchanged.
				if initialFileContents != nil {
					for filePath, initialContent := range initialFileContents {
						// Only check if file exists after an error, and it was initially present
						if _, statErr := os.Stat(filePath); statErr == nil {
							actualContent := readFileContent(t, filePath)
							if actualContent != initialContent {
								t.Errorf("File %s content changed unexpectedly after expected error for test %q.\nExpected initial:\n%q\nActual:\n%q", filePath, tt.name, initialContent, actualContent)
							}
						}
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error but got: %v for test %q", err, tt.name)
				}

				// Verify file contents
				for filePath, expectedContent := range expectedFileState {
					actualContent := readFileContent(t, filePath)
					if actualContent != expectedContent {
						t.Errorf("File content mismatch for %s in test %q.\nExpected:\n%q\nActual:\n%q", filePath, tt.name, expectedContent, actualContent)
					}

					// Special check for file permissions
					if strings.Contains(tt.name, "permissions") {
						fi, err := os.Stat(filePath)
						if err != nil {
							t.Fatalf("Failed to stat file %s for permissions check: %v", filePath, err)
						}
						// Expected permissions are 0644 (owner r/w, others r)
						actualPerms := fi.Mode().Perm()
						expectedPerms := os.FileMode(0644)
						if actualPerms != expectedPerms {
							t.Errorf("File permissions mismatch for %s in test %q.\nExpected: %o, Got: %o", filePath, tt.name, expectedPerms, actualPerms)
						}
					}
				}
			}
		})
	}
}
