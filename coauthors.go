package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Author is a coauthor "Full Name <email>"
type Author = string

func collectCoauthorsFromWipCommits(file *os.File) []Author {
	// Here we parse the SQUASH_MSG file for the list of authors on
	// the WIP branch.  If this technique later turns out to be
	// problematic, an alternative would be to instead fetch the
	// authors' list from the git log, using e.g.:
	//
	// silentgit("log", fmt.Sprintf("%s..", currentBaseBranch), "--reverse", "--pretty=format:%an <%ae>")
	//
	// For details and background, see https://github.com/remotemobprogramming/mob/issues/81

	coauthors := parseCoauthors(file)
	debugInfo("Parsed coauthors")
	debugInfo(strings.Join(coauthors, ","))

	coauthors = removeElementsContaining(coauthors, gitUserEmail())
	debugInfo("Parsed coauthors without committer")
	debugInfo(strings.Join(coauthors, ","))

	coauthors = removeDuplicateValues(coauthors)
	debugInfo("Unique coauthors without committer")
	debugInfo(strings.Join(coauthors, ","))

	sortByLength(coauthors)
	debugInfo("Sorted unique coauthors without committer")
	debugInfo(strings.Join(coauthors, ","))

	return coauthors
}

func parseCoauthors(file *os.File) []Author {
	var coauthors []Author

	authorOrCoauthorMatcher := regexp.MustCompile("(?i).*(author)+.+<+.*>+")
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if authorOrCoauthorMatcher.MatchString(line) {
			author := stripToAuthor(line)
			coauthors = append(coauthors, author)
		}
	}
	return coauthors
}

func stripToAuthor(line string) Author {
	return strings.TrimSpace(strings.Join(strings.Split(line, ":")[1:], ""))
}

func sortByLength(slice []string) {
	sort.Slice(slice, func(i, j int) bool {
		return len(slice[i]) < len(slice[j])
	})
}

func removeElementsContaining(slice []string, containsFilter string) []string {
	var result []string

	for _, entry := range slice {
		if !strings.Contains(entry, containsFilter) {
			result = append(result, entry)
		}
	}
	return result
}

func removeDuplicateValues(slice []string) []string {
	var result []string

	keys := make(map[string]bool)
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			result = append(result, entry)
		}
	}
	return result
}

func appendCoauthorsToSquashMsg(gitDir string) error {
	squashMsgPath := path.Join(gitDir, "SQUASH_MSG")
	debugInfo("opening " + squashMsgPath)
	file, err := os.OpenFile(squashMsgPath, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			debugInfo(squashMsgPath + " does not exist")
			// No wip commits, nothing to squash, this isn't really an error
			return nil
		}
		return err
	}

	defer file.Close()

	// read from repo/.git/SQUASH_MSG
	coauthors := collectCoauthorsFromWipCommits(file)

	if len(coauthors) > 0 {
		coauthorSuffix := createCommitMessage(coauthors)

		// append to repo/.git/SQUASH_MSG
		writer := bufio.NewWriter(file)
		writer.WriteString(coauthorSuffix)
		err = writer.Flush()
	}

	return err
}

func createCommitMessage(coauthors []Author) string {
	commitMessage := "\n\n"
	commitMessage += "# mob automatically added all co-authors from WIP commits\n"
	commitMessage += "# add missing co-authors manually\n"
	for _, coauthor := range coauthors {
		commitMessage += fmt.Sprintf("Co-authored-by: %s\n", coauthor)
	}
	return commitMessage
}
