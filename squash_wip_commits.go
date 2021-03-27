package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func squashWipCommits(configuration Configuration) {
	fmt.Println("TODO")
}

// used for non-interactive fixing of commit messages of squashed commits
func squashWipCommitsGitEditor(configuration Configuration) {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	result := commentWipCommits(lines, configuration)
	fmt.Println(result)
}

// used for non-interactive rebasing when squashing wip commits
func squashWipCommitsGitSequenceEditor(configuration Configuration) {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	result := markPostWipCommitsForSquashing(lines, configuration)
	fmt.Println(result)
}

func commentWipCommits(lines []string, configuration Configuration) string {
	var result = make([]string, len(lines))

	for i, line := range lines {
		if !isComment(line) && line == configuration.WipCommitMessage {
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

func markPostWipCommitsForSquashing(lines []string, configuration Configuration) string {
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
