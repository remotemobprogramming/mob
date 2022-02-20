package main

import (
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type Replacer func(string) string

func squashWip(configuration Configuration) {
	if !isMobProgramming(configuration) {
		sayTodo("to start working together, use", configuration.mob("start"))
		return
	}

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	mergeBase := silentgit("merge-base", currentWipBranch.String(), currentBaseBranch.String())

	originalGitEditor, originalGitSequenceEditor := getEnvGitEditor()
	setEnvGitEditor(
		mobExecutable()+" squash-wip --git-editor",
		mobExecutable()+" squash-wip --git-sequence-editor",
	)
	sayInfo("rewriting the history of the '" + currentWipBranch.String() + "' branch to squash wip commits but keep manual commits.")
	git("rebase", "-i", "--keep-empty", mergeBase)
	setEnvGitEditor(originalGitEditor, originalGitSequenceEditor)
	sayInfo("resulting history is:")
	sayLastCommitsWithMessage(currentBaseBranch.String(), currentWipBranch.String())
	if lastCommitIsWipCommit(configuration) { // last commit is wip commit
		sayInfo("undoing the final wip commit and staging its changes:")
		git("reset", "--soft", "HEAD^")
	}

	git("push", "--force", configuration.gitHooksOption())
}

func lastCommitIsWipCommit(configuration Configuration) bool {
	return lastCommitMessage() == configuration.WipCommitMessage
}

func lastCommitMessage() string {
	return silentgit("log", "-1", "--pretty=format:%s")
}

func sayLastCommitsWithMessage(currentBaseBranch string, currentWipBranch string) {
	commitsBaseWipBranch := currentBaseBranch + ".." + currentWipBranch
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=oneline", "--abbrev-commit")
	lines := strings.Split(log, "\n")
	if len(lines) > 10 {
		sayInfo("wip branch '" + currentWipBranch + "' contains " + strconv.Itoa(len(lines)) + " commits. The last 10 were:")
		lines = lines[:10]
	}
	output := strings.Join(lines, "\n")
	say(output)
}

func setEnvGitEditor(gitEditor string, gitSequenceEditor string) {
	os.Setenv("GIT_EDITOR", gitEditor)
	os.Setenv("GIT_SEQUENCE_EDITOR", gitSequenceEditor)
}

func getEnvGitEditor() (gitEditor string, gitSequenceEditor string) {
	gitEditor = os.Getenv("GIT_EDITOR")
	gitSequenceEditor = os.Getenv("GIT_SEQUENCE_EDITOR")
	return
}

func mobExecutable() string {
	if isTestEnvironment() {
		wd, _ := os.Getwd()
		return "go run $(ls -1 " + wd + "/*.go | grep -v _test.go)"
	} else {
		return "mob"
	}
}

func isTestEnvironment() bool {
	return strings.HasSuffix(os.Args[0], ".test") ||
		strings.HasSuffix(os.Args[0], "_test") ||
		os.Args[1] == "-test.v"
}

// used for non-interactive fixing of commit messages of squashed commits
func squashWipGitEditor(fileName string, configuration Configuration) {
	replaceFileContents(fileName, func(input string) string {
		return commentWipCommits(input, configuration)
	})
}

// used for non-interactive rebase to squash post-wip-commits
func squashWipGitSequenceEditor(fileName string, configuration Configuration) {
	replaceFileContents(fileName, func(input string) string {
		return markPostWipCommitsForSquashing(input, configuration)
	})
}

func replaceFileContents(fileName string, replacer Replacer) {
	file, _ := os.OpenFile(fileName, os.O_RDWR, 0666)
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

func markPostWipCommitsForSquashing(input string, configuration Configuration) string {
	var result []string

	inputLines := strings.Split(input, "\n")
	for index := range inputLines {
		markedLine := markLine(inputLines, index, configuration)
		result = append(result, markedLine)
	}

	return strings.Join(result, "\n")
}

func markLine(inputLines []string, i int, configuration Configuration) string {
	var resultLine = inputLines[i]
	previousLine := previousLine(inputLines, i)
	if isWipCommitLine(previousLine, configuration) {
		forthComingLines := inputLines[i:]

		if hasOnlyWipCommits(forthComingLines, configuration) {
			resultLine = markFixup(inputLines[i])
		} else {
			resultLine = markSquash(inputLines[i])
		}
	}
	return resultLine
}

func previousLine(inputLines []string, currentIndex int) string {
	var previousLine = ""
	if currentIndex > 0 {
		previousLine = inputLines[currentIndex-1]
	}
	return previousLine
}

func hasOnlyWipCommits(forthComingLines []string, configuration Configuration) bool {
	var onlyWipCommits = true
	for _, forthComingLine := range forthComingLines {
		if isPick(forthComingLine) && isManualCommit(forthComingLine, configuration) {
			onlyWipCommits = false
		}
	}
	return onlyWipCommits
}

func markSquash(line string) string {
	return strings.Replace(line, "pick ", "squash ", 1)
}

func markFixup(line string) string {
	return strings.Replace(line, "pick ", "fixup ", 1)
}

func isWipCommitLine(line string, configuration Configuration) bool {
	return isPick(line) && isWipCommit(line, configuration)
}

func isManualCommit(line string, configuration Configuration) bool {
	return !isWipCommit(line, configuration)
}

func isWipCommit(line string, configuration Configuration) bool {
	return strings.HasSuffix(line, configuration.WipCommitMessage)
}

func isPick(line string) bool {
	return strings.HasPrefix(line, "pick ")
}
