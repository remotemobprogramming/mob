package main

import (
	"fmt"
	"strings"
)

var Debug = false // override with --debug parameter

func parseDebug(args []string) {
	// debug needs to be parsed at the beginning to have DEBUG enabled as quickly as possible
	// otherwise, parsing others or other parameters don't have debug enabled
	for i := 0; i < len(args); i++ {
		if args[i] == "--debug" {
			Debug = true
		}
	}
}

func sayError(text string) {
	sayWithPrefix(text, "ERROR ")
}

func sayFix(instruction string, command string) {
	sayWithPrefix(instruction, "👉 ")
	sayEmptyLine()
	sayIndented(command)
	sayEmptyLine()
}

func sayNext(instruction string, command string) {
	sayWithPrefix(instruction, "👉 ")
	sayEmptyLine()
	sayIndented(command)
	sayEmptyLine()
}

func sayInfo(text string) {
	sayWithPrefix(text, "> ")
}

func sayInfoIndented(text string) {
	sayWithPrefix(text, "    ")
}

func sayWarning(text string) {
	sayWithPrefix(text, "⚠ ")
}

func sayIndented(text string) {
	sayWithPrefix(text, "  ")
}

func debugInfo(text string) {
	if Debug {
		sayWithPrefix(text, "DEBUG ")
	}
}

func sayWithPrefix(s string, prefix string) {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := 0; i < len(lines); i++ {
		printToConsole(prefix + strings.TrimSpace(lines[i]) + "\n")
	}
}

func say(s string) {
	if len(s) == 0 {
		return
	}
	printToConsole(strings.TrimRight(s, " \r\n\t\v\f") + "\n")
}

func sayEmptyLine() {
	printToConsole("\n")
}

var printToConsole = func(message string) {
	fmt.Print(message)
}
