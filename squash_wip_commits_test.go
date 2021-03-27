package main

import (
	"strings"
	"testing"
)

// TODO if last commit is wip commit, exit with warning

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

func TestMarkSquashWip_singleManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick c51a56d new file\n" +
		"\n" +
		"# Rebase ..."

	result := rebaseMarkSquashWip(lines, configuration)

	equals(t, lines, result)
}

func TestMarkSquashWip_manyManualCommits(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick c51a56d new file\n" +
		"pick 63ef7a4 another commit\n" +
		"\n" +
		"# Rebase ..."

	result := rebaseMarkSquashWip(lines, configuration)

	equals(t, lines, result)
}

func TestMarkSquashWip_wipCommitFollowedByManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"pick c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."
	expected := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"squash c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."

	result := rebaseMarkSquashWip(lines, configuration)

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
	expected := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a32 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a33 " + configuration.WipCommitMessage + "\n" +
		"squash c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."

	result := rebaseMarkSquashWip(lines, configuration)

	equals(t, expected, result)
}

func TestCommentWipCommits_oneWipAndOneManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "# This is a combination of 2 commits.\n" +
		"# This is the 1st commit message:\n" +
		"\n" +
		configuration.WipCommitMessage + "\n" +
		"\n" +
		"# This is the commit message #2:\n" +
		"\n" +
		"manual commit\n" +
		"\n" +
		"# Please enter ..."
	expected := "# This is a combination of 2 commits.\n" +
		"# This is the 1st commit message:\n" +
		"\n" +
		"# " + configuration.WipCommitMessage + "\n" +
		"\n" +
		"# This is the commit message #2:\n" +
		"\n" +
		"manual commit\n" +
		"\n" +
		"# Please enter ..."

	result := commentWipCommits(lines, configuration)

	equals(t, expected, result)
}

func commentWipCommits(content string, configuration Configuration) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var result = make([]string, len(lines))

	for i, line := range lines {
		if isComment(line) {
			result[i] = line
		} else if line == configuration.WipCommitMessage {
			result[i] = "# " + line
		} else {
			result[i] = line
		}
	}

	return strings.Join(result, "\n")
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "#")
}

func endsWithWipCommit(configuration Configuration) bool {
	log := silentgit("--no-pager", "log", "--pretty=format:%s%n")
	lines := strings.Split(strings.TrimSpace(log), "\n")
	return lines[0] == configuration.WipCommitMessage
}

func rebaseMarkSquashWip(content string, configuration Configuration) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var result = make([]string, len(lines))
	var squashNext = false

	for i, line := range lines {
		if squashNext && isRebaseCommitLine(line) {
			result[i] = strings.Replace(line, "pick ", "squash ", 1)
		} else {
			result[i] = line
		}
		squashNext = isRebaseWipCommitLine(line, configuration)
	}

	return strings.Join(result, "\n")
}

func isRebaseWipCommitLine(line string, configuration Configuration) bool {
	return isRebaseCommitLine(line) && strings.HasSuffix(line, configuration.WipCommitMessage)
}

func isRebaseCommitLine(line string) bool {
	return strings.HasPrefix(line, "pick ")
}
