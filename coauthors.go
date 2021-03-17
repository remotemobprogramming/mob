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

func gitStagedCoauthors() []string {
	coauthors := []string{}
	_, output, _ := runCommand("git", "config", "--global", "--get-regexp", "mob.staged.")

	// This by all rights should be an empty array when there are no staged coauthors,
	// but it's an array containing the empty string?
	staged := strings.Split(strings.TrimSpace(output), "\n")

	for _, stagedCoauthorWithKey := range staged {
		if stagedCoauthorWithKey == "" {
			continue
		}

		coauthors = append(coauthors, strings.Join(strings.Split(stagedCoauthorWithKey, " ")[1:], " "))
	}

	return coauthors
}

func clearAndAnnounceClearStagedCoauthors() error {
	var err error

	if len(gitStagedCoauthors()) > 0 {
		err = gitClearStagedCoauthors()
		if err == nil {
			sayInfo("Cleared previously staged co-authors from ~/.gitconfig")
		}
	}

	return err
}

func gitClearStagedCoauthors() error {
	_, _, err := runCommand("git", "config", "--global", "--remove-section", "mob.staged")
	return err
}

func loadCoauthorsFromAliases(coauthors CoauthorsMap) (CoauthorsMap, error) {
	missingAliases := []string{}

	for alias, coauthor := range coauthors {
		if coauthor == "" {
			coauthor, err := loadCoauthorFromAlias(alias)
			if err != nil {
				missingAliases = append(missingAliases, alias)
			} else {
				coauthors[alias] = coauthor
			}
		}
	}

	var err error
	if len(missingAliases) == 0 {
		err = nil
	} else {
		err = fmt.Errorf("%s were not listed in ~/.gitconfig. Try using fully qualified co-authors", strings.Join(missingAliases, ", "))
	}

	return coauthors, err
}

func loadCoauthorFromAlias(alias string) (string, error) {
	return gitconfig(true, fmt.Sprintf("mob.%s", alias))
}

func writeCoauthorsToGitConfig(coauthors CoauthorsMap) {
	clearAndAnnounceClearStagedCoauthors()

	if len(coauthors) == 0 {
		return
	}

	allCoauthors := make([]Author, 0, len(coauthors))
	newCoauthors := make([]Author, 0, len(coauthors))
	newCoauthorAliases := make([]Alias, 0, len(coauthors))

	for alias, coauthor := range coauthors {
		previous, _ := loadCoauthorFromAlias(alias)
		allCoauthors = append(allCoauthors, coauthor)
		gitconfig(true, fmt.Sprintf("mob.staged.%s", alias), coauthor)

		if previous != coauthor {
			gitconfig(true, fmt.Sprintf("mob.%s", alias), coauthor)
			if previous != "" {
				sayInfo(fmt.Sprintf("mob alias `%s` was updated to refer to `%s`", alias, coauthor))
			} else {
				newCoauthors = append(newCoauthors, coauthor)
				newCoauthorAliases = append(newCoauthorAliases, alias)
			}
		}
	}

	if len(newCoauthorAliases) > 0 {
		var beingVerb string
		if len(newCoauthorAliases) == 1 {
			beingVerb = "was"
		} else {
			beingVerb = "were"
		}

		sayInfo(fmt.Sprintf("%s %s saved to ~/.gitconfig", strings.Join(newCoauthors, ", "), beingVerb))
		sayIndented(fmt.Sprintf("Next time you can use `mob start --with \"%s\"`", strings.Join(newCoauthorAliases, ", ")))
	}

	var pluralizedCoauthors string
	var presentPassiveVerb string
	if len(allCoauthors) == 1 {
		pluralizedCoauthors = "a co-author"
		presentPassiveVerb = "has"
	} else {
		pluralizedCoauthors = "co-authors"
		presentPassiveVerb = "have"
	}

	sayInfo(fmt.Sprintf("%s %s been staged as %s in ~/.gitconfig", strings.Join(allCoauthors, ", "), presentPassiveVerb, pluralizedCoauthors))
	sayIndented(fmt.Sprintf("They will appear as %s on your next WIP commit,", pluralizedCoauthors))
	sayIndented(fmt.Sprintf("and they will appear as %s after `mob done`.", pluralizedCoauthors))
}

func collectCoauthorsFromWipCommits(file *os.File) []Author {
	var committer string
	coauthorsHashSet := make(map[Author]bool)

	authorOrCoauthorMatcher := regexp.MustCompile("(?i).*(author)+.+<+.*>+")
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if authorOrCoauthorMatcher.MatchString(line) {
			author := stripToAuthor(line)

			// committer of this commit should
			// not be included as a co-author
			if committer == "" || author == committer {
				committer = author
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
