package main

import (
	"fmt"
	"regexp"
)

func TestPattern() {
	// Test the pattern from config
	pattern := `^(\\./)?history_viewer(\\s|$)`
	fmt.Printf("Pattern: %s\n", pattern)
	
	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("Error compiling: %v\n", err)
		return
	}
	
	testCommands := []string{
		"./history_viewer &",
		"history_viewer",
		"./history_viewer",
		"go build -o history_viewer && ./history_viewer &",
	}
	
	for _, cmd := range testCommands {
		matches := re.MatchString(cmd)
		fmt.Printf("Command: %q -> Match: %v\n", cmd, matches)
	}
}
