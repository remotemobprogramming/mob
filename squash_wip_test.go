package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestSquashWipCommits_acceptance(t *testing.T) {
	_, configuration := setup(t)

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
	_, configuration := setup(t)
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
	output, configuration := setup(t)
	start(configuration)
	createFile(t, "file2.txt", "irrelevant")
	next(configuration)
	start(configuration)

	squashWip(configuration)

	assertCommitLogContainsMessage(t, gitCurrentBranch().Name, configuration.WipCommitMessage)
	assertOutputContains(t, output, "failed to squash wip commits")
}

func TestSquashWipCommits_failsOnMainBranch(t *testing.T) {
	output, configuration := setup(t)

	squashWip(configuration)

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestSquashWipCommits_worksWithEmptyCommits(t *testing.T) {
	_, configuration := setup(t)

	// change without manual commit
	start(configuration)
	createFile(t, "file1.txt", "irrelevant")
	next(configuration)

	start(configuration)
	silentgit("commit", "--allow-empty", "-m ok")

	squashWip(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, []string{
		"ok",
	}, commitsOnCurrentBranch(configuration))
}

func TestCommitsOnCurrentBranch(t *testing.T) {
	_, configuration := setup(t)
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
	_, configuration := setup(t)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "new file")

	equals(t, false, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_finalWipCommit(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	createFile(t, "file1.txt", "irrelevant")
	next(configuration)
	start(configuration)

	equals(t, true, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_manualThenWipCommit(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "new file")
	createFile(t, "file2.txt", "irrelevant")
	next(configuration)
	start(configuration)

	equals(t, true, endsWithWipCommit(configuration))
}

func TestEndsWithWipCommit_wipThenManualCommit(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	createFile(t, "file2.txt", "irrelevant")
	next(configuration)
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "irrelevant", "new file")

	equals(t, false, endsWithWipCommit(configuration))
}

func TestMarkSquashWip_singleManualCommit(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := `pick c51a56d new file

# Rebase ...`

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, input, result)
}

func TestMarkSquashWip_manyManualCommits(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := `pick c51a56d new file
pick 63ef7a4 another commit

# Rebase ...`

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, input, result)
}

func TestMarkSquashWip_wipCommitFollowedByManualCommit(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := fmt.Sprintf(`pick 01a9a31 %s
pick c51a56d manual commit

# Rebase ...`, configuration.WipCommitMessage)
	expected := fmt.Sprintf(`pick 01a9a31 %s
squash c51a56d manual commit

# Rebase ...`, configuration.WipCommitMessage)

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, expected, result)
}

func TestMarkSquashWip_manyWipCommitsFollowedByManualCommit(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := fmt.Sprintf(`pick 01a9a31 %[1]s
pick 01a9a32 %[1]s
pick 01a9a33 %[1]s
pick c51a56d manual commit

# Rebase ...`, configuration.WipCommitMessage)
	expected := fmt.Sprintf(`pick 01a9a31 %[1]s
squash 01a9a32 %[1]s
squash 01a9a33 %[1]s
squash c51a56d manual commit

# Rebase ...`, configuration.WipCommitMessage)

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, expected, result)
}

func TestCommentWipCommits_oneWipAndOneManualCommit(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := fmt.Sprintf(`# This is a combination of 2 commits.
# This is the 1st commit message:

%s

# This is the commit message #2:

manual commit

# Please enter ...`, configuration.WipCommitMessage)
	expected := fmt.Sprintf(`# This is a combination of 2 commits.
# This is the 1st commit message:

# %s

# This is the commit message #2:

manual commit

# Please enter ...`, configuration.WipCommitMessage)

	result := commentWipCommits(input, configuration)

	equals(t, expected, result)
}

func TestSquashWipCommitGitEditor(t *testing.T) {
	configuration := getDefaultConfiguration()
	createTestbed(t, configuration)
	input := createFile(t, "commits", fmt.Sprintf(
		`# This is a combination of 2 commits.
# This is the 1st commit message:

%s

# This is the commit message #2:

new file

# Please enter the commit message for your changes. Lines starting`, configuration.WipCommitMessage))
	expected := fmt.Sprintf(
		`# This is a combination of 2 commits.
# This is the 1st commit message:

# %s

# This is the commit message #2:

new file

# Please enter the commit message for your changes. Lines starting`, configuration.WipCommitMessage)

	squashWipGitEditor(input, getDefaultConfiguration())

	result, _ := ioutil.ReadFile(input)
	equals(t, expected, string(result))
}

func TestSquashWipCommitGitSequenceEditor(t *testing.T) {
	configuration := getDefaultConfiguration()
	createTestbed(t, configuration)
	input := createFile(t, "rebase", fmt.Sprintf(
		`pick 01a9a31 %[1]s
pick 01a9a32 %[1]s
pick 01a9a33 %[1]s
pick c51a56d manual commit

# Rebase ...
`, configuration.WipCommitMessage))
	expected := fmt.Sprintf(
		`pick 01a9a31 %[1]s
squash 01a9a32 %[1]s
squash 01a9a33 %[1]s
squash c51a56d manual commit

# Rebase ...
`, configuration.WipCommitMessage)

	squashWipGitSequenceEditor(input, getDefaultConfiguration())

	result, _ := ioutil.ReadFile(input)
	equals(t, expected, string(result))
}
