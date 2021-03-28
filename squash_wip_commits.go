package main

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func squashWipCommits(configuration Configuration) {
	os.Setenv("GIT_EDITOR", "mob squash-wip-commits --git-editor")
	os.Setenv("GIT_SEQUENCE_EDITOR", "mob squash-wip-commits --git-sequence-editor")
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	//TODO shouldnt this always to TrimSpace?
	mergeBase := strings.TrimSpace(silentgit("merge-base", currentWipBranch, currentBaseBranch))
	silentgit("rebase", "-i", mergeBase)
}

// used for non-interactive fixing of commit messages of squashed commits
func squashWipCommitsGitEditor(fileName string, configuration Configuration) {
	file, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	input, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	result := commentWipCommits(string(input), configuration)

	file.Seek(0, io.SeekStart)
	file.Truncate(0)
	file.WriteString(result)
	file.Close()
}

// used for non-interactive rebasing when squashing wip commits
func squashWipCommitsGitSequenceEditor(fileName string, configuration Configuration) {
	file, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	input, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	result := markPostWipCommitsForSquashing(string(input), configuration)

	file.Seek(0, io.SeekStart)
	file.Truncate(0)
	file.WriteString(result)
	file.Close()
}

func commentWipCommits(input string, configuration Configuration) string {
	var result []string
	for _, line := range strings.Split(input, "\n") {
		if !isComment(line) && line == configuration.WipCommitMessage {
			result = append(result, "# "+line)
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "#")
}

func endsWithWipCommit(configuration Configuration) bool {
	commits := commitsOnCurrentBranch(configuration)
	return commits[0] == configuration.WipCommitMessage
}

func commitsOnCurrentBranch(configuration Configuration) []string {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	log := silentgit("--no-pager", "log", currentBaseBranch+".."+currentWipBranch, "--pretty=format:%s")
	lines := strings.Split(strings.TrimSpace(log), "\n")
	return lines
}

func markPostWipCommitsForSquashing(input string, configuration Configuration) string {
	var result []string

	var squashNext = false
	for _, line := range strings.Split(input, "\n") {
		if squashNext && isRebaseCommitLine(line) {
			result = append(result, strings.Replace(line, "pick ", "squash ", 1))
		} else {
			result = append(result, line)
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
