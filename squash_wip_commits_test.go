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

func TestMarkSquashWip_singleManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick c51a56d new file\n" +
		"\n" +
		"# Rebase ..."

	result := markSquashWip(lines, configuration)

	equals(t, lines, result)
}

func TestMarkSquashWip_manyManualCommits(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick c51a56d new file\n" +
		"pick 63ef7a4 another commit\n" +
		"\n" +
		"# Rebase ..."

	result := markSquashWip(lines, configuration)

	equals(t, lines, result)
}

func TestMarkSquashWip_wipCommitFollowedByManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"pick c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."
	expected := "squash 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"pick c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."

	result := markSquashWip(lines, configuration)

	equals(t, expected, result)
}

func TestMarkSquashWip_manyWipCommitsFollowedByManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"pick 01a9a32 " + configuration.WipCommitMessage + "\n" +
		"pick 01a9a33 " + configuration.WipCommitMessage + "\n" +
		"pick c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."
	expected := "squash 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a32 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a33 " + configuration.WipCommitMessage + "\n" +
		"pick c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."

	result := markSquashWip(lines, configuration)

	equals(t, expected, result)
}

func markSquashWip(content string, configuration Configuration) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var result = make([]string, len(lines))
	for i, line := range lines {
		if strings.HasPrefix(line, "pick ") && strings.HasSuffix(line, configuration.WipCommitMessage) {
			result[i] = strings.Replace(line, "pick ", "squash ", 1)
		} else {
			result[i] = line
		}
	}
	return strings.Join(result, "\n")
}
