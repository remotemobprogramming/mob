package main

import (
	"runtime"
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
)

func lastCommitMessageOfHead() string {
	return silentgit("log", "--format=%B", "-n", "1", "HEAD")
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
