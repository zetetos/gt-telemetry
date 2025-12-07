package main

import (
	"github.com/fatih/color"
)

// colorPrinter wraps fatih/color functions for consistent color output.
type colorPrinter struct {
	red    *color.Color
	green  *color.Color
	yellow *color.Color
	cyan   *color.Color
}

// newColorPrinter creates a new colorPrinter with optional color disabling.
func newColorPrinter(noColor bool) *colorPrinter {
	color.NoColor = noColor

	return &colorPrinter{
		red:    color.New(color.FgRed),
		green:  color.New(color.FgGreen),
		yellow: color.New(color.FgYellow),
		cyan:   color.New(color.FgCyan),
	}
}

// Red returns a red-colored string.
func (c *colorPrinter) Red(s string) string { return c.red.Sprint(s) }

// Green returns a green-colored string.
func (c *colorPrinter) Green(s string) string { return c.green.Sprint(s) }

// Yellow returns a yellow-colored string.
func (c *colorPrinter) Yellow(s string) string { return c.yellow.Sprint(s) }

// Cyan returns a cyan-colored string.
func (c *colorPrinter) Cyan(s string) string { return c.cyan.Sprint(s) }
