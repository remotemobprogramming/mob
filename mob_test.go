package main

import (
	"strings"
	"testing"
)

func Test_version(t *testing.T) {
	messages := ""
	printToConsole = func(text string) {
		messages += text
	}

	version()

	if !strings.Contains(messages, versionNumber) {
		t.Error("version command doesn't print current version number")
	}
}
