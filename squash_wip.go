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
		sayError("you aren't mob programming")
		sayTodo("to start, use", "mob start")
		return
	}

	if endsWithWipCommit(configuration) {
		sayError(`failed to squash wip commits
last commit must be a manual commit`)
		sayEmptyLine()
		sayTodo("create a manual commit with a commit message to fix this:", "git commit --allow-empty -m \"your message goes here\"")
		return
	}

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	mergeBase := silentgit("merge-base", currentWipBranch.String(), currentBaseBranch.String())

	originalGitEditor, originalGitSequenceEditor := getEnvGitEditor()
	setEnvGitEditor(
		mobExecutable()+" squash-wip --git-editor",
		mobExecutable()+" squash-wip --git-sequence-editor",
	)
	silentgit("rebase", "-i", "--keep-empty", mergeBase)
	setEnvGitEditor(originalGitEditor, originalGitSequenceEditor)
	sayInfo("the history of your '" + currentWipBranch.String() + "' branch has been rewritten to combine all wip commits with their following manual commits:")
	sayEmptyLine()
	sayLastCommitsWithMessage(currentBaseBranch.String(), currentWipBranch.String())
	sayEmptyLine()
	sayTodo("to finally put the changes into the base branch preserving the resulting commits, call:", "mob done --no-squash")
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

func endsWithWipCommit(configuration Configuration) bool {
	return configuration.isWipCommitMessage(commitsOnCurrentBranch(configuration)[0])
}

func commitsOnCurrentBranch(configuration Configuration) []string {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	commitsBaseWipBranch := currentBaseBranch.String() + ".." + currentWipBranch.String()
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%s")
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
