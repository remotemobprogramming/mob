package main

import (
	"strings"
	"testing"
)

func TestEndsWithWipCommit_finalManualCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "owqe", "new file")

	equals(t, false, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_finalWipCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFile(t, "file1.txt", "owqe")
	next(configuration)
	start(configuration)

	equals(t, true, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_manualThenWipCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "owqe", "new file")
	createFile(t, "file2.txt", "owqe")
	next(configuration)
	start(configuration)

	equals(t, true, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_wipThenManualCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFile(t, "file2.txt", "owqe")
	next(configuration)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "owqe", "new file")

	equals(t, false, endsWithWipCommit(configuration))
}

func endsWithWipCommit(configuration Configuration) bool {
	log := silentgit("--no-pager", "log", "--pretty=format:%s%n")
	lines := strings.Split(strings.TrimSpace(log), "\n")
	return lines[0] == configuration.WipCommitMessage
}
