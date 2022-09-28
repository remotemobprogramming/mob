package main

import (
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/say"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type Replacer func(string) string

func squashWip(configuration config.Configuration) {
	if hasUncommittedChanges() {
		makeWipCommit(configuration)
	}
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	mergeBase := silentgit("merge-base", currentWipBranch.String(), currentBaseBranch.remote(configuration).String())

	originalGitEditor, originalGitSequenceEditor := getEnvGitEditor()
	setEnvGitEditor(
		mobExecutable()+" squash-wip --git-editor",
		mobExecutable()+" squash-wip --git-sequence-editor",
	)
	say.Info("rewriting history of '" + currentWipBranch.String() + "': squashing wip commits while keeping manual commits.")
	git("rebase", "--interactive", "--keep-empty", mergeBase)
	setEnvGitEditor(originalGitEditor, originalGitSequenceEditor)
	say.Info("resulting history is:")
	sayLastCommitsWithMessage(currentBaseBranch.remote(configuration).String(), currentWipBranch.String())
	if lastCommitIsWipCommit(configuration) { // last commit is wip commit
		say.Info("undoing the final wip commit and staging its changes:")
		git("reset", "--soft", "HEAD^")
	}

	git("push", "--force", gitHooksOption(configuration))
}

func lastCommitIsWipCommit(configuration config.Configuration) bool {
	return strings.HasPrefix(lastCommitMessage(), configuration.WipCommitMessage)
}

func lastCommitMessage() string {
	return silentgit("log", "-1", "--pretty=format:%B")
}

func sayLastCommitsWithMessage(currentBaseBranch string, currentWipBranch string) {
	commitsBaseWipBranch := currentBaseBranch + ".." + currentWipBranch
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=oneline", "--abbrev-commit")
	lines := strings.Split(log, "\n")
	if len(lines) > 10 {
		say.Info("wip branch '" + currentWipBranch + "' contains " + strconv.Itoa(len(lines)) + " commits. The last 10 were:")
		lines = lines[:10]
	}
	output := strings.Join(lines, "\n")
	say.Say(output)
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
		return "cd " + wd + " && go run $(ls -1 ./*.go | grep -v _test.go)"
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
func squashWipGitEditor(fileName string, configuration config.Configuration) {
	replaceFileContents(fileName, func(input string) string {
		return commentWipCommits(input, configuration)
	})
}

// used for non-interactive rebase to squash post-wip-commits
func squashWipGitSequenceEditor(fileName string, configuration config.Configuration) {
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

func commentWipCommits(input string, configuration config.Configuration) string {
	var result []string
	ignoreBlock := false
	lines := strings.Split(input, "\n")
	for idx, line := range lines {
		if configuration.IsWipCommitMessage(line) {
			ignoreBlock = true
		} else if line == "" && isNextLineComment(lines, idx) {
			ignoreBlock = false
		}

		if ignoreBlock {
			result = append(result, "# "+line)
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func isNextLineComment(lines []string, currentLineIndex int) bool {
	return len(lines) > currentLineIndex+1 && strings.HasPrefix(lines[currentLineIndex+1], "#")
}

func markPostWipCommitsForSquashing(input string, configuration config.Configuration) string {
	var result []string

	inputLines := strings.Split(input, "\n")
	for index := range inputLines {
		markedLine := markLine(inputLines, index, configuration)
		result = append(result, markedLine)
	}

	return strings.Join(result, "\n")
}

func markLine(inputLines []string, i int, configuration config.Configuration) string {
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

func hasOnlyWipCommits(forthComingLines []string, configuration config.Configuration) bool {
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

func isWipCommitLine(line string, configuration config.Configuration) bool {
	return isPick(line) && isWipCommit(line, configuration)
}

func isManualCommit(line string, configuration config.Configuration) bool {
	return !isWipCommit(line, configuration)
}

func isWipCommit(line string, configuration config.Configuration) bool {
	return strings.Contains(line, configuration.WipCommitMessage)
}

func isPick(line string) bool {
	return strings.HasPrefix(line, "pick ")
}
