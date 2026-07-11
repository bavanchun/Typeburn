package ui

import "fmt"

// displayModeLabel formats a persisted mode identifier for History and Result.
func displayModeLabel(mode string, length int) string {
	switch mode {
	case "time", "words":
		return fmt.Sprintf("%s %d", mode, length)
	case "quote", "code":
		return mode
	case "":
		return "unknown"
	default:
		return mode
	}
}
