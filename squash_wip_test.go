package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestSquashWipCommits_acceptance(t *testing.T) {
	_, configuration := setup(t)
	wipCommit(t, configuration, "file1.txt")
	manualCommit(t, configuration, "file2.txt", "first manual commit")

	// manual commit followed by a wip commit
	start(configuration)
	createFileAndCommitIt(t, "file3.txt", "irrelevant", "second manual commit")
	createFile(t, "file4.txt", "irrelevant")
	next(configuration)

	// final manual commit
	start(configuration)
	createFileAndCommitIt(t, "file5.txt", "irrelevant", "third manual commit")

	squashWip(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, []string{
		"third manual commit",
		"second manual commit",
		"first manual commit",
	}, commitsOnCurrentBranch(configuration))
	equals(t, commitsOnCurrentBranch(configuration), commitsOnRemoteBranch(configuration))
}

func TestSquashWipCommits_withFinalWipCommit(t *testing.T) {
	_, configuration := setup(t)
	wipCommit(t, configuration, "file1.txt")
	manualCommit(t, configuration, "file2.txt", "first manual commit")
	wipCommit(t, configuration, "file3.txt")
	start(configuration)

	squashWip(configuration)

	assertOnBranch(t, "mob-session")
	assertGitStatus(t, GitStatus{
		"file3.txt": "A",
	})
	equals(t, []string{
		"first manual commit",
	}, commitsOnCurrentBranch(configuration))
}

func TestSquashWipCommits_withManyFinalWipCommits(t *testing.T) {
	_, configuration := setup(t)
	wipCommit(t, configuration, "file1.txt")
	manualCommit(t, configuration, "file2.txt", "first manual commit")
	wipCommit(t, configuration, "file3.txt")
	wipCommit(t, configuration, "file4.txt")
	start(configuration)

	squashWip(configuration)

	assertOnBranch(t, "mob-session")
	assertGitStatus(t, GitStatus{
		"file3.txt": "A",
		"file4.txt": "A",
	})
	equals(t, []string{
		"first manual commit",
	}, commitsOnCurrentBranch(configuration))
}

func TestSquashWipCommits_onlyWipCommits(t *testing.T) {
	_, configuration := setup(t)
	wipCommit(t, configuration, "file1.txt")
	wipCommit(t, configuration, "file2.txt")
	wipCommit(t, configuration, "file3.txt")
	start(configuration)

	squashWip(configuration)

	assertOnBranch(t, "mob-session")
	assertGitStatus(t, GitStatus{
		"file1.txt": "A",
		"file2.txt": "A",
		"file3.txt": "A",
	})
	equals(t, []string{""}, commitsOnCurrentBranch(configuration))
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

func TestSquashWipCommits_failsOnMainBranch(t *testing.T) {
	output, configuration := setup(t)

	squashWip(configuration)

	assertOutputContains(t, output, "to start working together")
}

func TestSquashWipCommits_worksWithEmptyCommits(t *testing.T) {
	_, configuration := setup(t)
	wipCommit(t, configuration, "file1.txt")

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

func TestMarkSquashWip_manualCommitFollowedByWipCommit(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := fmt.Sprintf(`pick c51a56d manual commit
pick 01a9a31 %[1]s

# Rebase ...`, configuration.WipCommitMessage)
	expected := fmt.Sprintf(`pick c51a56d manual commit
pick 01a9a31 %[1]s

# Rebase ...`, configuration.WipCommitMessage)

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, expected, result)
}

func TestMarkSquashWip_manualCommitFollowedByManyWipCommits(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := fmt.Sprintf(`pick c51a56d manual commit
pick 01a9a31 %[1]s
pick 01a9a32 %[1]s
pick 01a9a33 %[1]s

# Rebase ...`, configuration.WipCommitMessage)
	expected := fmt.Sprintf(`pick c51a56d manual commit
pick 01a9a31 %[1]s
fixup 01a9a32 %[1]s
fixup 01a9a33 %[1]s

# Rebase ...`, configuration.WipCommitMessage)

	result := markPostWipCommitsForSquashing(input, configuration)

	equals(t, expected, result)
}

func TestMarkSquashWip_wipThenManualCommitFollowedByManyWipCommits(t *testing.T) {
	configuration := getDefaultConfiguration()
	input := fmt.Sprintf(`pick 01a9a31 %[1]s
pick c51a56d manual commit
pick 01a9a32 %[1]s
pick 01a9a33 %[1]s

# Rebase ...`, configuration.WipCommitMessage)
	expected := fmt.Sprintf(`pick 01a9a31 %[1]s
squash c51a56d manual commit
pick 01a9a32 %[1]s
fixup 01a9a33 %[1]s

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

func wipCommit(t *testing.T, configuration Configuration, filename string) {
	start(configuration)
	createFile(t, filename, "irrelevant")
	next(configuration)
}

func manualCommit(t *testing.T, configuration Configuration, filename string, message string) {
	start(configuration)
	createFileAndCommitIt(t, filename, "irrelevant", message)
	next(configuration)
}

func commitsOnCurrentBranch(configuration Configuration) []string {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	commitsBaseWipBranch := currentBaseBranch.String() + ".." + currentWipBranch.String()
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%s")
	lines := strings.Split(log, "\n")
	return lines
}

func commitsOnRemoteBranch(configuration Configuration) []string {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	commitsBaseWipBranch := currentBaseBranch.String() + ".." + configuration.RemoteName + "/" + currentWipBranch.String()
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%s")
	lines := strings.Split(log, "\n")
	return lines
}
