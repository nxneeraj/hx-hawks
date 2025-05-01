package output

import "github.com/fatih/color"

// Define color functions for terminal output
var (
	ColorGreen   = color.New(color.FgGreen).SprintFunc()
	ColorRed     = color.New(color.FgRed).SprintFunc()
	ColorWhite   = color.New(color.FgWhite).SprintFunc()
	ColorBlue    = color.New(color.FgBlue).SprintFunc()
	ColorMagenta = color.New(color.FgMagenta).SprintFunc() // Pink/Magenta
	ColorYellow  = color.New(color.FgYellow).SprintFunc()  // For warnings or info
	ColorCyan    = color.New(color.FgCyan).SprintFunc()    // For details like keywords
)
