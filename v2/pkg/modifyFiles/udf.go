package modifyFiles

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/golang/glog"
)

// ApplyChangesToFiles takes a single unifiedDiff string, parses it to identify
// changes for individual files, reads the original content of those files from disk,
// applies the diffs, and writes the modified content back to disk.
func ApplyChangesToFiles(unifiedDiff string) error {
	unifiedDiff = sanitizeResponse(unifiedDiff)
	reader := strings.NewReader(unifiedDiff)
	files, _, err := gitdiff.Parse(reader)
	if err != nil {
		return fmt.Errorf("failed to parse git diff: %v", err)
	}

	for _, file := range files {
		// Access fields directly on the struct.
		var output bytes.Buffer
		filePath := file.OldName
		if strings.HasPrefix(filePath, "a/") || strings.HasPrefix(filePath, "b/") {
			filePath = filePath[2:]
		}

		glog.V(2).Infof("File changed: %s", filePath)
		targetFile, _ := os.Open(filePath)
		defer targetFile.Close()
		if err := gitdiff.Apply(&output, targetFile, file); err != nil {
			return fmt.Errorf("failed to apply git diff: %v", err)
		}
		err = os.WriteFile(filePath, output.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("failed to write file %q: %v", filePath, err)
		}
	}
	return nil
}

func sanitizeResponse(response string) string {
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		response = strings.Join(lines[1:len(lines)-1], "\n")
	}
	return response
}
