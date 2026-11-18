package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/workdir"
)

func writeEditorState(t *testing.T, state editorState) {
	t.Helper()
	content, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	writeEditorStateRaw(t, string(content))
}

func writeEditorStateRaw(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(gitDir(), filepath.FromSlash(editorStateFile))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func currentEditorState(file string, line int) editorState {
	return editorState{Version: 1, File: file, Line: line, UpdatedAt: time.Now().Add(time.Second), Editor: "test"}
}

func lastCommitMessageOfHead() string {
	return silentgit("log", "--format=%B", "-n", "1", "HEAD")
}

func TestNext_UsesEditorStateFile_WithLine(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createFile(t, "olderFile.txt", "contentIrrelevant")
	createFile(t, "newerFile.txt", "contentIrrelevant")
	writeEditorState(t, currentEditorState("olderFile.txt", 42))

	next(configuration)

	equals(t, configuration.WipCommitMessage+"\n\nlastFile:olderFile.txt\nlastLine:42", lastCommitMessageOfHead())
}

func TestNext_UsesEditorStateFile_FileInSubdirectoryAndUnchanged(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createDirectory(t, "dir")
	createFileInPath(t, workdir.Path+"/dir", "viewed.txt", "contentIrrelevant")
	next(configuration)

	start(configuration)
	createFile(t, "changed.txt", "contentIrrelevant")
	writeEditorState(t, currentEditorState("dir/viewed.txt", 3))
	next(configuration)

	equals(t, configuration.WipCommitMessage+"\n\nlastFile:dir/viewed.txt\nlastLine:3", lastCommitMessageOfHead())
}

func TestNext_UsesEditorStateFile_WithoutLine(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	createFile(t, "file2.txt", "contentIrrelevant")
	writeEditorState(t, currentEditorState("file1.txt", 0))

	next(configuration)

	equals(t, configuration.WipCommitMessage+"\n\nlastFile:file1.txt", lastCommitMessageOfHead())
}

func TestNext_UsesEditorStateFile_QuotesPathWithSpaces(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createFile(t, "file with spaces.txt", "contentIrrelevant")
	createFile(t, "other.txt", "contentIrrelevant")
	writeEditorState(t, currentEditorState("file with spaces.txt", 5))

	next(configuration)

	equals(t, configuration.WipCommitMessage+"\n\nlastFile:\"file with spaces.txt\"\nlastLine:5", lastCommitMessageOfHead())
}

func TestNext_IgnoresEditorStateFile_WhenOlderThanLastCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createFile(t, "olderFile.txt", "contentIrrelevant")
	createFile(t, "newerFile.txt", "contentIrrelevant")
	state := currentEditorState("olderFile.txt", 42)
	state.UpdatedAt = time.Now().Add(-time.Hour)
	writeEditorState(t, state)

	next(configuration)

	equals(t, configuration.WipCommitMessage+"\n\nlastFile:newerFile.txt", lastCommitMessageOfHead())
}

func TestNext_IgnoresEditorStateFile_WhenFileDoesNotExist(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createFile(t, "file.txt", "contentIrrelevant")
	writeEditorState(t, currentEditorState("deleted.txt", 42))

	next(configuration)

	equals(t, configuration.WipCommitMessage+"\n\nlastFile:file.txt", lastCommitMessageOfHead())
}

func TestNext_IgnoresEditorStateFile_WhenInvalid(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createFile(t, "file.txt", "contentIrrelevant")
	writeEditorStateRaw(t, "{ not json")

	next(configuration)

	equals(t, configuration.WipCommitMessage+"\n\nlastFile:file.txt", lastCommitMessageOfHead())
}

func TestValidateEditorState(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, "dir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "dir", "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	lastCommit := time.Now().Add(-time.Minute)
	valid := editorState{Version: 1, File: "dir/file.txt", Line: 1, UpdatedAt: time.Now()}

	equals(t, "", validateEditorState(valid, rootDir, lastCommit))

	cases := map[string]func(s *editorState){
		"unknown version":    func(s *editorState) { s.Version = 2 },
		"missing file":       func(s *editorState) { s.File = "" },
		"absolute path":      func(s *editorState) { s.File = filepath.ToSlash(filepath.Join(rootDir, "dir", "file.txt")) },
		"outside repository": func(s *editorState) { s.File = "../file.txt" },
		"directory":          func(s *editorState) { s.File = "dir" },
		"not existing":       func(s *editorState) { s.File = "dir/missing.txt" },
		"stale":              func(s *editorState) { s.UpdatedAt = lastCommit.Add(-time.Second) },
		"missing timestamp":  func(s *editorState) { s.UpdatedAt = time.Time{} },
	}
	for name, modify := range cases {
		state := valid
		modify(&state)
		if validateEditorState(state, rootDir, lastCommit) == "" {
			t.Errorf("expected editor state to be rejected: %s", name)
		}
	}
}

func TestParseLastFileFromCommitMessage(t *testing.T) {
	path, line, warning := parseLastFileFromCommitMessage("mob next [ci-skip]\n\nlastFile:src/Foo.java\nlastLine:42")
	equals(t, "src/Foo.java", path)
	equals(t, 42, line)
	equals(t, "", warning)

	path, line, _ = parseLastFileFromCommitMessage("mob next [ci-skip]\n\nlastFile:\"a b.txt\"\nlastLine:7\n\nSigned-off-by: someone")
	equals(t, "a b.txt", path)
	equals(t, 7, line)

	path, line, _ = parseLastFileFromCommitMessage("mob next [ci-skip]\n\nlastFile:src/Foo.java")
	equals(t, "src/Foo.java", path)
	equals(t, 0, line)

	_, line, _ = parseLastFileFromCommitMessage("mob next [ci-skip]\n\nlastFile:src/Foo.java\nlastLine:abc")
	equals(t, 0, line)

	_, _, warning = parseLastFileFromCommitMessage("mob next [ci-skip]")
	if warning == "" {
		t.Error("expected a warning when lastFile is missing")
	}
}

func TestQuotePathIfNeeded(t *testing.T) {
	equals(t, "src/Foo.java", quotePathIfNeeded("src/Foo.java"))
	equals(t, "src/Über.java", quotePathIfNeeded("src/Über.java"))
	equals(t, "\"a b.txt\"", quotePathIfNeeded("a b.txt"))
	equals(t, "\"a\\\"b.txt\"", quotePathIfNeeded("a\"b.txt"))
}

func TestOpenCommandFor(t *testing.T) {
	configuration := config.GetDefaultConfiguration()
	assertOpenCommand := func(openCommand string, line int, expectedName string, expectedArgs ...string) {
		t.Helper()
		configuration.OpenCommand = openCommand
		name, args := openCommandFor(configuration, "/repo/dir with space/Foo.java", line)
		equals(t, expectedName, name)
		equals(t, expectedArgs, args)
	}

	assertOpenCommand("idea %s", 42, "idea", "/repo/dir with space/Foo.java")
	assertOpenCommand("idea", 42, "idea", "/repo/dir with space/Foo.java")
	assertOpenCommand("code -g %{file}:%{line}", 42, "code", "-g", "/repo/dir with space/Foo.java:42")
	assertOpenCommand("idea --line %{line} %{file}", 42, "idea", "--line", "42", "/repo/dir with space/Foo.java")
	assertOpenCommand("code -g %s:%{line}", 42, "code", "-g", "/repo/dir with space/Foo.java:42")
	assertOpenCommand("code -g %{file}:%{line}", 0, "code", "-g", "/repo/dir with space/Foo.java:1")
}

func TestStartNextStay_OpenLastFile_WithLinePlaceholder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses touch to observe the open command")
	}
	_, configuration := setup(t)
	configuration.NextStay = true
	configuration.OpenCommand = "touch %{file}-%{line}"

	start(configuration)
	createFile(t, "file.txt", "contentIrrelevant")
	writeEditorState(t, currentEditorState("file.txt", 42))
	next(configuration)

	start(configuration)

	assertGitStatus(t, GitStatus{
		"file.txt-42": "??",
	})
}

func TestStartNextStay_OpenLastFile_WithLinePlaceholderButNoLineInCommit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses touch to observe the open command")
	}
	_, configuration := setup(t)
	configuration.NextStay = true
	configuration.OpenCommand = "touch %{file}-%{line}"

	start(configuration)
	createFile(t, "file.txt", "contentIrrelevant")
	next(configuration)

	start(configuration)

	assertGitStatus(t, GitStatus{
		"file.txt-1": "??",
	})
}

// The exact format the IntelliJ plugin writes (Instant.toString() with millisecond precision).
func TestEditorStateFile_ParsesFormatWrittenByIntellijPlugin(t *testing.T) {
	for _, updatedAt := range []string{"2026-11-18T10:15:03.123Z", "2026-11-18T10:15:03Z"} {
		var state editorState
		content := "{\n  \"version\": 1,\n  \"file\": \"dir with space/Über.java\",\n  \"line\": 42,\n  \"updatedAt\": \"" + updatedAt + "\",\n  \"editor\": \"IntelliJ IDEA 2024.2.5\",\n  \"plugin\": \"Mob Last File\",\n  \"pluginVersion\": \"0.1.0\"\n}\n"
		if err := json.Unmarshal([]byte(content), &state); err != nil {
			t.Fatal(err)
		}
		equals(t, 1, state.Version)
		equals(t, "dir with space/Über.java", state.File)
		equals(t, 42, state.Line)
		equals(t, "IntelliJ IDEA 2024.2.5", state.Editor)
		equals(t, "Mob Last File", state.Plugin)
		equals(t, "0.1.0", state.PluginVersion)
		equals(t, time.Date(2026, 11, 18, 10, 15, 3, 0, time.UTC), state.UpdatedAt.UTC().Truncate(time.Second))
	}
}
