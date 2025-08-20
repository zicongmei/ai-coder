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
	udfPath := "/tmp/unifiedDiff.txt"
	err := os.WriteFile(udfPath, []byte(unifiedDiff), 0644)
	if err != nil {
		glog.Errorf("Failed to write unified diff to %s: %v", udfPath, err)
		return fmt.Errorf("failed to write %s: %v", udfPath, err)
	}
	glog.V(2).Infof("Unified diff written to %s", udfPath)
	reader := strings.NewReader(unifiedDiff)
	files, _, err := gitdiff.Parse(reader)
	if err != nil {
		glog.Errorf("Failed to parse git diff: %v", err)
		return fmt.Errorf("failed to parse git diff: %v", err)
	}

	for _, file := range files {
		// Access fields directly on the struct.
		var output bytes.Buffer
		filePath := file.OldName
		if strings.HasPrefix(filePath, "a/") || strings.HasPrefix(filePath, "b/") {
			filePath = filePath[2:]
		}

		targetFile, _ := os.Open(filePath)
		defer targetFile.Close()
		glog.V(3).Infof("Opened file %s for modification", filePath)

		applier := gitdiff.NewTextApplier(&output, targetFile)
		for i, frag := range file.TextFragments {
			glog.V(3).Infof("Fragment #%d: %s", i, frag.String())
			if err := applier.ApplyFragment(frag); err != nil {
				return fmt.Errorf("failed to apply fragment %d: %v", i, err)
			}
			glog.V(3).Infof("Applied fragment %d to file %s", i, filePath)
		}
		err = applier.Close()
		if err != nil {
			return fmt.Errorf("failed to close applier: %v", err)
		}

		// if err := gitdiff.Apply(&output, targetFile, file); err != nil {
		// 	glog.Errorf("Failed to apply git diff for file %q: %v", filePath, err)
		// 	return fmt.Errorf("failed to apply git diff: %v", err)
		// }
		err = os.WriteFile(filePath, output.Bytes(), 0644)
		if err != nil {
			glog.Errorf("Failed to write file %q: %v", filePath, err)
			return fmt.Errorf("failed to write file %q: %v", filePath, err)
		}
		glog.V(2).Infof("Modified content written to %s", filePath)
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
