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

	committerEmail := gitUserEmail()
	coauthorsHashSet := make(map[Author]bool)

	authorOrCoauthorMatcher := regexp.MustCompile("(?i).*(author)+.+<+.*>+")
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if authorOrCoauthorMatcher.MatchString(line) {
			author := stripToAuthor(line)

			// committer of this commit should
			// not be included as a co-author
			if strings.Contains(author, committerEmail) {
				continue
			}
			coauthorsHashSet[author] = true
		}
	}

	coauthors := make([]string, 0, len(coauthorsHashSet))

	for k := range coauthorsHashSet {
		coauthors = append(coauthors, k)
	}
	sort.Sort(byLength(coauthors))

	return coauthors
}

func appendCoauthorsToSquashMsg(workingDir string) error {
	squashMsgPath := path.Join(workingDir, ".git", "SQUASH_MSG")
	file, err := os.OpenFile(squashMsgPath, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		if err == os.ErrNotExist {
			// No wip commits, nothing to squash, this isn't really an error
			return nil
		}
		return err
	}

	defer file.Close()

	coauthors := collectCoauthorsFromWipCommits(file)

	if len(coauthors) > 0 {
		coauthorSuffix := "\n\n"
		coauthorSuffix += "# mob automatically added all co-authors from WIP commits\n"
		coauthorSuffix += "# add missing co-authors manually\n"

		coauthorSuffix += coauthorsForCommitMsg(coauthors)

		writer := bufio.NewWriter(file)
		_, err = writer.WriteString(coauthorSuffix)
		err = writer.Flush()
	}

	return err
}

func coauthorsForCommitMsg(coauthors []Author) string {
	var commitMsg string

	for _, coauthor := range coauthors {
		commitMsg += fmt.Sprintf("Co-authored-by: %s\n", coauthor)
	}

	return commitMsg
}

type byLength []string

func (s byLength) Len() int {
	return len(s)
}
func (s byLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byLength) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func stripToAuthor(line string) string {
	return strings.TrimSpace(strings.Join(strings.Split(line, ":")[1:], ""))
}
