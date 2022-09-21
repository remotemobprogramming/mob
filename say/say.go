package say

import (
	"fmt"
	"strings"
)

var isDebug = false // override with --debug parameter

func TurnOnDebugging() {
	isDebug = true
}

func TurnOnDebuggingByArgs(args []string) {
	// debug needs to be parsed at the beginning to have DEBUG enabled as quickly as possible
	// otherwise, parsing others or other parameters don't have debug enabled
	for i := 0; i < len(args); i++ {
		if args[i] == "--debug" {
			isDebug = true
		}
	}
}

func Error(text string) {
	WithPrefix(text, "ERROR ")
}

func Warning(text string) {
	WithPrefix(text, "⚠ ")
}

func Info(text string) {
	WithPrefix(text, "> ")
}

func InfoIndented(text string) {
	WithPrefix(text, "    ")
}

func Indented(text string) {
	WithPrefix(text, "  ")
}

func Fix(instruction string, command string) {
	WithPrefix(instruction, "👉 ")
	emptyLine()
	Indented(command)
	emptyLine()
}

func Next(instruction string, command string) {
	WithPrefix(instruction, "👉 ")
	emptyLine()
	Indented(command)
	emptyLine()
}

func WithPrefix(s string, prefix string) {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := 0; i < len(lines); i++ {
		PrintToConsole(prefix + strings.TrimSpace(lines[i]) + "\n")
	}
}

func emptyLine() {
	PrintToConsole("\n")
}

func Say(s string) {
	if len(s) == 0 {
		return
	}
	PrintToConsole(strings.TrimRight(s, " \r\n\t\v\f") + "\n")
}

func Debug(text string) {
	if isDebug {
		WithPrefix(text, "DEBUG ")
	}
}

var PrintToConsole = func(message string) {
	fmt.Print(message)
}
