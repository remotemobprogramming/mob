package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/remotemobprogramming/mob/v5/say"
)

const (
	// editorStateFile is written by editor plugins (e.g. the IntelliJ mob last file plugin)
	// into the git dir of the repository. See README "Editor plugins: the editor state file".
	editorStateFile          = "mob/last-file.json"
	editorStateFormatVersion = 1
)

// editorStateFileSource reads the file an editor plugin wrote into <git dir>/mob/last-file.json.
type editorStateFileSource struct{}

type editorState struct {
	Version   int       `json:"version"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	UpdatedAt time.Time `json:"updatedAt"`
	// informational only
	Editor        string `json:"editor"`
	Plugin        string `json:"plugin"`
	PluginVersion string `json:"pluginVersion"`
}

func (editorStateFileSource) name() string { return "editor state file" }

func (editorStateFileSource) find() (lastFile, bool) {
	path := filepath.Join(gitDir(), filepath.FromSlash(editorStateFile))
	content, err := os.ReadFile(path)
	if err != nil {
		say.Debug("No editor state file found (" + path + ")")
		return lastFile{}, false
	}
	var state editorState
	if err := json.Unmarshal(content, &state); err != nil {
		say.Debug("Ignoring editor state file, it is not valid JSON: " + err.Error())
		return lastFile{}, false
	}
	if reason := validateEditorState(state, gitRootDir(), lastCommitTime()); reason != "" {
		say.Debug("Ignoring editor state file: " + reason)
		return lastFile{}, false
	}
	say.Debug("Using editor state file written by plugin '" + state.Plugin + "' " + state.PluginVersion + " in '" + state.Editor + "' at " + state.UpdatedAt.String())
	line := state.Line
	if line < 1 {
		line = 0
	}
	return lastFile{Path: quotePathIfNeeded(state.File), Line: line}, true
}

// validateEditorState returns an empty string if the state is usable, otherwise the reason why it is not.
func validateEditorState(state editorState, rootDir string, lastCommit time.Time) string {
	if state.Version != editorStateFormatVersion {
		return "unsupported version " + strconv.Itoa(state.Version)
	}
	if state.File == "" {
		return "no file given"
	}
	localPath := filepath.FromSlash(state.File)
	if !filepath.IsLocal(localPath) {
		return "file '" + state.File + "' is not a relative path inside the repository"
	}
	if !state.UpdatedAt.After(lastCommit) {
		return "it was written (" + state.UpdatedAt.String() + ") before the last commit (" + lastCommit.String() + ")"
	}
	info, err := os.Stat(filepath.Join(rootDir, localPath))
	if err != nil || !info.Mode().IsRegular() {
		return "file '" + state.File + "' does not exist in the repository"
	}
	return ""
}

func lastCommitTime() time.Time {
	seconds, err := strconv.ParseInt(strings.TrimSpace(silentgit("log", "-1", "--format=%ct")), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(seconds, 0)
}

// quotePathIfNeeded quotes paths the same way 'git status --porcelain' does for paths containing
// spaces or special characters, so every mob version can read them back via strconv.Unquote.
func quotePathIfNeeded(path string) string {
	for _, r := range path {
		if r == ' ' || r == '"' || r == '\\' || r < 0x20 || r == 0x7f {
			return strconv.Quote(path)
		}
	}
	return path
}
