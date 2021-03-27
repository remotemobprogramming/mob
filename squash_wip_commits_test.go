package main

import (
	"os/exec"
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

	result := markPostWipCommitsForSquashing(strings.Split(lines, "\n"), configuration)

	equals(t, lines, result)
}

func TestMarkSquashWip_manyManualCommits(t *testing.T) {
	configuration = getDefaultConfiguration()
	lines := "pick c51a56d new file\n" +
		"pick 63ef7a4 another commit\n" +
		"\n" +
		"# Rebase ..."

	result := markPostWipCommitsForSquashing(strings.Split(lines, "\n"), configuration)

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

	result := markPostWipCommitsForSquashing(strings.Split(lines, "\n"), configuration)

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

	result := markPostWipCommitsForSquashing(strings.Split(lines, "\n"), configuration)

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

	result := commentWipCommits(strings.Split(lines, "\n"), configuration)

	equals(t, expected, result)
}

func TestSquashWipCommitGitEditor(t *testing.T) {
	command := exec.Command("go", "run", "mob.go", "squash_wip_commits.go", "coauthors.go", "swc", "--git-editor")
	stdin, _ := command.StdinPipe()
	stdin.Write([]byte("# This is a combination of 2 commits.\n" +
		"# This is the 1st commit message:\n \n" +
		"mob next [ci-skip] [ci skip] [skip ci]\n \n" +
		"# This is the commit message #2:\n \n" +
		"new file\n \n" +
		"# Please enter the commit message for your changes. Lines starting\n"))
	stdin.Close()

	outputBinary, _ := command.CombinedOutput()
	output := string(outputBinary)
	command.Run()

	expected := "# This is a combination of 2 commits.\n" +
		"# This is the 1st commit message:\n \n" +
		"# mob next [ci-skip] [ci skip] [skip ci]\n \n" +
		"# This is the commit message #2:\n \n" +
		"new file\n \n" +
		"# Please enter the commit message for your changes. Lines starting\n"
	equals(t, expected, output)
}

func TestSquashWipCommitGitSequenceEditor(t *testing.T) {
	configuration = getDefaultConfiguration()
	command := exec.Command("go", "run", "mob.go", "squash_wip_commits.go", "coauthors.go", "swc", "--git-sequence-editor")
	stdin, _ := command.StdinPipe()
	stdin.Write([]byte("pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"pick 01a9a32 " + configuration.WipCommitMessage + "\n" +
		"pick 01a9a33 " + configuration.WipCommitMessage + "\n" +
		"pick c51a56d manual commit\n" +
		"\n" +
		"# Rebase ...\n"))
	stdin.Close()

	outputBinary, _ := command.CombinedOutput()
	output := string(outputBinary)
	command.Run()

	expected := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a32 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a33 " + configuration.WipCommitMessage + "\n" +
		"squash c51a56d manual commit\n" +
		"\n" +
		"# Rebase ...\n"
	equals(t, expected, output)
}
