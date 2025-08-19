package utils

// TruncateString is a helper function to shorten long strings for logging or display,
// preventing them from becoming excessively long. If the string's length exceeds
// maxLen, it is truncated and "..." is appended.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}