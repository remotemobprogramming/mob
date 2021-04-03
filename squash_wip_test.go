package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestSquashWipCommits_acceptance(t *testing.T) {
	_, configuration := localSetup(t)

	// change without manual commit
	start(configuration)
	createFile(t, "file1.txt", "irrelevant")
	next(configuration)

	// change with a manual commit
	start(configuration)
	createFileAndCommitIt(t, "file2.txt", "irrelevant", "first manual commit")
	next(configuration)

	// change with a manual commit followed by an uncommited change
	start(configuration)
	createFileAndCommitIt(t, "file3.txt", "irrelevant", "second manual commit")
	createFile(t, "file4.txt", "irrelevant")
	next(configuration)

	// change with a final manual commit
	start(configuration)
	createFileAndCommitIt(t, "file5.txt", "irrelevant", "third manual commit")

	squashWip(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, []string{
		"third manual commit",
		"second manual commit",
		"first manual commit",
	}, commitsOnCurrentBranch(configuration))
}

func TestSquashWipCommits_resetsEnv(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "new file")
	originalGitEditor := "irrelevant"
	originalGitSequenceEditor := "irrelevant, too"
	os.Setenv("GIT_EDITOR", originalGitEditor)
	os.Setenv("GIT_SEQUENCE_EDITOR", originalGitSequenceEditor)

	squashWip(configuration)

	equals(t, originalGitEditor, os.Getenv("GIT_EDITOR"))
	equals(t, originalGitSequenceEditor, os.Getenv("GIT_SEQUENCE_EDITOR"))
}

func TestSquashWipCommits_failsOnFinalWipCommit(t *testing.T) {
	output, configuration := localSetup(t)
	start(configuration)
	createFile(t, "file2.txt", "irrelevant")
	next(configuration)
	start(configuration)
	exitedWithCode := -1
	exit = func(code int) {
		exitedWithCode = code
	}

	squashWip(configuration)

	equals(t, 1, exitedWithCode)
	assertOutputContains(t, output, "Make sure the final commit is a manual commit before squashing")
}

func TestSquashWipCommits_failsOnMainBranch(t *testing.T) {
	output, configuration := localSetup(t)
	exitedWithCode := -1
	exit = func(code int) {
		exitedWithCode = code
	}

	squashWip(configuration)

	equals(t, 1, exitedWithCode)
	assertOutputContains(t, output, "Make sure you are on the wip-branch before running quash-wip")
}

func TestCommitsOnCurrentBranch(t *testing.T) {
	_, configuration := localSetup(t)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "not on branch")
	silentgit("push")
	start(configuration)
	createFileAndCommitIt(t, "file2.txt", "irrelevant", "on branch")
	createFile(t, "file3.txt", "irrelevant")
	next(configuration)
	start(configuration)

	commits := commitsOnCurrentBranch(configuration)

	equals(t, []string{
		configuration.WipCommitMessage,
		"on branch",
	}, commits)
}

func TestEndsWithWipCommit_finalManualCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "new file")

	equals(t, false, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_finalWipCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFile(t, "file1.txt", "irrelevant")
	next(configuration)
	start(configuration)

	equals(t, true, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_manualThenWipCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "new file")
	createFile(t, "file2.txt", "irrelevant")
	next(configuration)
	start(configuration)

	equals(t, true, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_wipThenManualCommit(t *testing.T) {
	_, configuration := localSetup(t)
	start(configuration)
	createFile(t, "file2.txt", "irrelevant")
	next(configuration)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "new file")

	equals(t, false, endsWithWipCommit(configuration))
}

func TestMarkSquashWip_singleManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	input := "pick c51a56d new file\n" +
		"\n" +
		"# Rebase ..."

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, input, result)
}

func TestMarkSquashWip_manyManualCommits(t *testing.T) {
	configuration = getDefaultConfiguration()
	input := "pick c51a56d new file\n" +
		"pick 63ef7a4 another commit\n" +
		"\n" +
		"# Rebase ..."

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, input, result)
}

func TestMarkSquashWip_wipCommitFollowedByManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	input := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"pick c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."
	expected := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"squash c51a56d manual commit\n" +
		"\n" +
		"# Rebase ..."

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, expected, result)
}

func TestMarkSquashWip_manyWipCommitsFollowedByManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	input := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
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

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, expected, result)
}

func TestCommentWipCommits_oneWipAndOneManualCommit(t *testing.T) {
	configuration = getDefaultConfiguration()
	input := "# This is a combination of 2 commits.\n" +
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

	result := commentWipCommits(input, configuration)

	equals(t, expected, result)
}

func TestSquashWipCommitGitEditor(t *testing.T) {
	createTestbed(t)
	path := createFile(t, "commits",
		"# This is a combination of 2 commits.\n"+
			"# This is the 1st commit message:\n \n"+
			"mob next [ci-skip] [ci skip] [skip ci]\n \n"+
			"# This is the commit message #2:\n \n"+
			"new file\n \n"+
			"# Please enter the commit message for your changes. Lines starting\n")
	expected := "# This is a combination of 2 commits.\n" +
		"# This is the 1st commit message:\n \n" +
		"# mob next [ci-skip] [ci skip] [skip ci]\n \n" +
		"# This is the commit message #2:\n \n" +
		"new file\n \n" +
		"# Please enter the commit message for your changes. Lines starting\n"

	squashWipGitEditor(path, getDefaultConfiguration())

	result, _ := ioutil.ReadFile(path)
	equals(t, expected, string(result))
}

func TestSquashWipCommitGitSequenceEditor(t *testing.T) {
	createTestbed(t)
	configuration = getDefaultConfiguration()
	path := createFile(t, "rebase",
		"pick 01a9a31 "+configuration.WipCommitMessage+"\n"+
			"pick 01a9a32 "+configuration.WipCommitMessage+"\n"+
			"pick 01a9a33 "+configuration.WipCommitMessage+"\n"+
			"pick c51a56d manual commit\n"+
			"\n"+
			"# Rebase ...\n")
	expected := "pick 01a9a31 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a32 " + configuration.WipCommitMessage + "\n" +
		"squash 01a9a33 " + configuration.WipCommitMessage + "\n" +
		"squash c51a56d manual commit\n" +
		"\n" +
		"# Rebase ...\n"

	squashWipGitSequenceEditor(path, getDefaultConfiguration())

	result, _ := ioutil.ReadFile(path)
	equals(t, expected, string(result))
}
