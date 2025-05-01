package output

import (
	"fmt"
	"log"
	"strings"

	"github.com/nxneeraj/hx-hawks/pkg/types" 
)

const MaxResponseLength = 500 // Limit response preview length in terminal

// PrintResultTerminal formats and prints a single scan result to the terminal with colors.
func PrintResultTerminal(result types.ScanResult) {
	if result.Error != "" {
		log.Printf("[%s] %s - Error: %s", ColorYellow("ERROR"), result.URL, result.Error)
		return
	}

	if result.IsVulnerable {
		fmt.Printf("[%s] %s (Status: %d)\n", ColorRed("VULNERABLE"), result.URL, result.StatusCode)
		// Print response preview in blue
		responsePreview := result.ResponseBody
		if len(responsePreview) > MaxResponseLength {
			responsePreview = responsePreview[:MaxResponseLength] + "..."
		}
		// Highlight keywords in the preview
		highlightedResponse := highlightKeywords(responsePreview, result.MatchedKeywords)
		fmt.Printf("  Response (%s):\n%s\n", ColorBlue("Vulnerable"), ColorBlue(highlightedResponse))

		// Print matched keywords
		if len(result.MatchedKeywords) > 0 {
			fmt.Printf("  [%s]: '%s' %s\n", ColorCyan("MATCHED"), ColorMagenta(strings.Join(result.MatchedKeywords, "', '")), ColorMagenta("🔍"))
		}

	} else {
		fmt.Printf("[%s] %s (Status: %d)\n", ColorGreen("SAFE"), result.URL, result.StatusCode)
		// Optionally print safe response preview in white
		// responsePreview := result.ResponseBody
		// if len(responsePreview) > MaxResponseLength {
		// 	responsePreview = responsePreview[:MaxResponseLength] + "..."
		// }
		// fmt.Printf("  Response (%s):\n%s\n", ColorWhite("Safe"), ColorWhite(responsePreview))
	}
	fmt.Println() // Add a blank line for separation
}

// highlightKeywords highlights occurrences of keywords in the text using Magenta.
// This is a simple string replacement; more sophisticated highlighting might be needed
// for overlapping keywords or case-insensitivity if required.
func highlightKeywords(text string, keywords []string) string {
	highlightedText := text
	for _, keyword := range keywords {
		// Simple case-sensitive replace. Use regex for case-insensitivity or complex patterns.
		highlightedText = strings.ReplaceAll(highlightedText, keyword, ColorMagenta(keyword))
	}
	return highlightedText
}
