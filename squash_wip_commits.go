package main

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type Replacer func(string) string

func squashWipCommits(configuration Configuration) {
	if endsWithWipCommit(configuration) {
		sayInfo("Make sure the final commit is a manual commit before squashing")
		exit(1)
		return
	}
	os.Setenv("GIT_EDITOR", mobExecutable()+" squash-wip-commits --git-editor")
	os.Setenv("GIT_SEQUENCE_EDITOR", mobExecutable()+" squash-wip-commits --git-sequence-editor")
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	mergeBase := silentgit("merge-base", currentWipBranch, currentBaseBranch)
	silentgit("rebase", "-i", mergeBase)
}

func mobExecutable() string {
	if isTestEnvironment() {
		return "go run mob.go"
	} else {
		return "mob"
	}
}

func isTestEnvironment() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}

// used for non-interactive fixing of commit messages of squashed commits
func squashWipCommitsGitEditor(fileName string, configuration Configuration) {
	replaceFileContents(fileName, func(input string) string {
		return commentWipCommits(input, configuration)
	})
}

// used for non-interactive rebase to squash post-wip-commits
func squashWipCommitsGitSequenceEditor(fileName string, configuration Configuration) {
	replaceFileContents(fileName, func(input string) string {
		return markPostWipCommitsForSquashing(input, configuration)
	})
}

func replaceFileContents(fileName string, replacer Replacer) {
	file, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	input, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	result := replacer(string(input))

	file.Seek(0, io.SeekStart)
	file.Truncate(0)
	file.WriteString(result)
	file.Close()
}

func commentWipCommits(input string, configuration Configuration) string {
	var result []string
	for _, line := range strings.Split(input, "\n") {
		if configuration.isWipCommitMessage(line) {
			result = append(result, "# "+line)
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func endsWithWipCommit(configuration Configuration) bool {
	return configuration.isWipCommitMessage(commitsOnCurrentBranch(configuration)[0])
}

func commitsOnCurrentBranch(configuration Configuration) []string {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	log := silentgit("--no-pager", "log", currentBaseBranch+".."+currentWipBranch, "--pretty=format:%s")
	lines := strings.Split(log, "\n")
	return lines
}

func markPostWipCommitsForSquashing(input string, configuration Configuration) string {
	var result []string

	var squashNext = false
	for _, line := range strings.Split(input, "\n") {
		if squashNext && isPick(line) {
			result = append(result, markSquash(line))
		} else {
			result = append(result, line)
		}
		squashNext = isRebaseWipCommitLine(line, configuration)
	}

	return strings.Join(result, "\n")
}

func markSquash(line string) string {
	return strings.Replace(line, "pick ", "squash ", 1)
}

func isRebaseWipCommitLine(line string, configuration Configuration) bool {
	return isPick(line) && strings.HasSuffix(line, configuration.WipCommitMessage)
}

func isPick(line string) bool {
	return strings.HasPrefix(line, "pick ")
}
