package main

import (
	"strconv"
	"strings"

	"github.com/remotemobprogramming/mob/v5/say"
)

const (
	lastFileMarker = "lastFile:"
	lastLineMarker = "lastLine:"
)

// lastFile describes the file (and optionally the line) which is written into the wip commit message.
type lastFile struct {
	Path string // relative to the git root, as written into the commit message (may be quoted)
	Line int    // 1-based; 0 means unknown
}

// lastFileSource is one way of determining the last file of the current typist.
type lastFileSource interface {
	name() string
	find() (lastFile, bool)
}

// lastFileSources are tried in order; the first one that finds a file wins.
// Implementations: lastfile_editorstate.go, lastfile_modified.go
var lastFileSources = []lastFileSource{
	editorStateFileSource{},
	lastModifiedFileSource{},
}

func determineLastFile() lastFile {
	for _, source := range lastFileSources {
		if result, found := source.find(); found {
			say.Debug("Determined last file using " + source.name() + ": " + result.Path)
			return result
		}
		say.Debug("No last file found using " + source.name())
	}
	return lastFile{}
}

func appendLastFile(commitMessage string, file lastFile) string {
	if file.Path == "" {
		return commitMessage
	}
	commitMessage += "\n\n" + lastFileMarker + file.Path
	if file.Line > 0 {
		commitMessage += "\n" + lastLineMarker + strconv.Itoa(file.Line)
	}
	return commitMessage
}

// parseLastFileFromCommitMessage extracts the lastFile and the optional lastLine from a wip commit message.
// The returned path is unquoted and relative to the git root.
func parseLastFileFromCommitMessage(message string) (path string, line int, warning string) {
	split := strings.Split(message, lastFileMarker)
	if len(split) == 1 {
		return "", 0, "Couldn't find last modified file in commit message!"
	}
	if len(split) > 2 {
		return "", 0, "Could not determine last modified file from commit message, separator was used multiple times!"
	}
	path = strings.Split(split[1], "\n")[0]
	if strings.HasPrefix(path, "\"") {
		path, _ = strconv.Unquote(path)
	}
	for _, messageLine := range strings.Split(message, "\n") {
		if value, found := strings.CutPrefix(strings.TrimSpace(messageLine), lastLineMarker); found {
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
				line = parsed
			}
			break
		}
	}
	return path, line, ""
}
