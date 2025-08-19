package main

import (
	"flag"

	"github.com/golang/glog" // Import glog
)

// Config holds the command-line arguments for the coder application.
type Config struct {
	FileList string // Path to a file containing a list of files to process
	Flash    bool   // Whether to use flash mode
	Inplace  bool   // Whether to modify the files in place
	Prompt   string // The prompt to send to the AI
}

func main() {
	// Defer glog.Flush() to ensure all log messages are written to their destination
	// (e.g., stderr or log file) before the application exits.
	defer glog.Flush()

	var cfg Config

	// Define command-line flags. glog also registers its own flags (e.g., -v, -logtostderr).
	flag.StringVar(&cfg.FileList, "file-list", "", "Path to a file containing a list of files to process")
	flag.BoolVar(&cfg.Flash, "flash", false, "Use flash mode for AI interaction")
	flag.BoolVar(&cfg.Inplace, "inplace", false, "Modify the files in place (requires --file-list)")
	flag.StringVar(&cfg.Prompt, "prompt", "", "The prompt string to send to the AI")

	// Parse the flags. This single call parses both custom flags and glog's flags.
	flag.Parse()

	glog.V(1).Info("Application started. Parsing command-line arguments and validating configuration.")

	// Basic validation for required arguments.
	// Using glog.Fatal for unrecoverable startup errors, which also flushes logs and exits.
	if cfg.FileList == "" {
		glog.Error("Validation Error: --file-list is a required argument.")
		flag.Usage() // Prints flag usage information to stderr
		glog.Fatal("Exiting due to missing --file-list argument.")
	}

	if cfg.Prompt == "" {
		glog.Error("Validation Error: --prompt is a required argument.")
		flag.Usage()
		glog.Fatal("Exiting due to missing --prompt argument.")
	}

	// This specific validation is somewhat redundant if --file-list is already required,
	// but kept for consistency with the original code's logic flow.
	if cfg.Inplace && cfg.FileList == "" {
		glog.Error("Validation Error: --inplace requires --file-list to be specified.")
		flag.Usage()
		glog.Fatal("Exiting due to --inplace specified without --file-list.")
	}

	// Log the parsed configuration at verbosity level 0 (always visible by default).
	glog.V(0).Infof("Coder application starting with the following configuration:")
	glog.V(0).Infof("  File List: %q", cfg.FileList)
	glog.V(0).Infof("  Flash Mode: %t", cfg.Flash)
	glog.V(0).Infof("  In-place Modification: %t", cfg.Inplace)
	glog.V(0).Infof("  Prompt provided (length: %d characters).", len(cfg.Prompt))
	// Log the full prompt content at a higher verbosity level for debugging purposes.
	glog.V(2).Infof("  Full Prompt Content: %q", cfg.Prompt)

	// Placeholder for the actual AI coding logic.
	glog.V(0).Info("\n--- Placeholder for actual AI coding logic ---")
	glog.V(0).Infof("Logic will read files from: %q", cfg.FileList)
	// Log a truncated version of the prompt to avoid excessively long log lines for the actual call.
	glog.V(0).Infof("Logic will send prompt to AI (excerpt): %q...", truncateString(cfg.Prompt, 50))
	if cfg.Inplace {
		glog.V(0).Info("Logic will modify files in place.")
	} else {
		glog.V(0).Info("Logic will output modified content (not in-place).")
	}
	glog.V(0).Info("-------------------------------------------")

	// TOD

	glog.V(0).Info("Coder application finished successfully.")
}

// truncateString is a helper function to shorten long strings for logging,
// preventing log lines from becoming excessively long.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
