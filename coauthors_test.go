package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestGitStagedCoauthors(t *testing.T) {
	setWorkingDir("/tmp/mob/local")
	testCoauthors := map[string]string{
		"mob-t1": "t1@example.com",
		"mob-t2": "t2@example.com",
	}

	for alias, coauthor := range testCoauthors {
		_, output, err := runCommand("git", "config", "--global", fmt.Sprintf("mob.staged.%s", alias), coauthor)
		if err != nil {
			failWithFailure(t, nil, err.Error()+": "+output)
		}
	}

	var testCoauthorsList []string
	for _, coauthor := range testCoauthors {
		testCoauthorsList = append(testCoauthorsList, coauthor)
	}

	equals(t, testCoauthorsList, gitStagedCoauthors())

	runCommand("git", "config", "--global", "--remove-section", "mob.staged")
}

func TestGitClearRemovesStagedSection(t *testing.T) {
	testCoauthors := map[string]string{
		"mob-t1": "t1@example.com",
		"mob-t2": "t2@example.com",
	}

	for alias, coauthor := range testCoauthors {
		runCommand("git", "config", "--global", fmt.Sprintf("mob.staged.%s", alias), coauthor)
	}

	gitClearStagedCoauthors()
	_, output, _ := runCommand("git", "config", "--global", "--get-regexp", "mob.staged.")
	equals(t, "", output)
}

func TestDoesNotAnnounceClearWhenNoCoauthors(t *testing.T) {
	runCommand("git", "config", "--global", "--remove-section", "mob.staged")

	output := captureOutput()
	clearAndAnnounceClearStagedCoauthors()
	equals(t, "", *output)
}

func TestDoesAnnouncesClearWhenCoauthors(t *testing.T) {
	testCoauthors := map[string]string{
		"mob-t1": "t1@example.com",
		"mob-t2": "t2@example.com",
	}

	for alias, coauthor := range testCoauthors {
		_, output, err := runCommand("git", "config", "--global", fmt.Sprintf("mob.staged.%s", alias), coauthor)
		if err != nil {
			failWithFailure(t, nil, err.Error()+" "+output)
		}
	}

	output := captureOutput()
	clearAndAnnounceClearStagedCoauthors()
	assertOutputContains(t, output, "Cleared previously staged co-authors from ~/.gitconfig")
}

func TestGitStagedEmptyCoauthors(t *testing.T) {
	equals(t, []string{}, gitStagedCoauthors())

	runCommand("git", "config", "--global", "--remove-section", "mob.staged")
}

func TestLoadsKnownAlias(t *testing.T) {
	expectedCoauthors := map[string]string{
		"mob-t1": "t1@example.com",
		"mob-t2": "t2@example.com",
	}

	for alias, coauthor := range expectedCoauthors {
		runCommand("git", "config", "--global", fmt.Sprintf("mob.%s", alias), coauthor)
	}

	testAliasMap := map[string]string{
		"mob-t1": "",
		"mob-t2": "",
	}

	output := captureOutput()
	coauthors, _ := loadCoauthorsFromAliases(testAliasMap)

	for alias, _ := range expectedCoauthors {
		runCommand("git", "config", "--global", "--unset", fmt.Sprintf("mob.%s", alias))
	}

	equals(t, expectedCoauthors, coauthors)
	assertOutputNotContains(t, output, "not listed in ~/.gitconfig. Try using fully qualified co-authors")
}

func TestReturnsErrorForUnknownAlias(t *testing.T) {
	testAliasMap := map[string]string{
		"mob-t1": "",
		"mob-t2": "",
	}

	_, err := loadCoauthorsFromAliases(testAliasMap)

	if !(strings.Contains(err.Error(), "mob-t1") && strings.Contains(err.Error(), "mob-t2")) {
		failWithFailure(t, "mob-t1, mob-t2 to be in error", err.Error())
	}

}

func TestStartDoneCoAuthors(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	createFile(t, "file2.txt", "asdf")
	next(configuration)

	setWorkingDir("/tmp/mob/alice")
	start(configuration)
	createFile(t, "file3.txt", "owqe")
	next(configuration)

	setWorkingDir("/tmp/mob/alice")
	start(configuration)
	createFile(t, "file4.txt", "zcvx")
	next(configuration)

	setWorkingDir("/tmp/mob/bob")
	start(configuration)
	createFile(t, "file5.txt", "oiuo")
	next(configuration)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "file6.txt", "oiuo")
	done(configuration)

	output := run(t, "cat", "/tmp/mob/local/.git/SQUASH_MSG")
	// don't include the person running `mob done`
	assertOutputNotContains(t, output, "Co-authored-by: local <local@example.com>")
	// include everyone else in commit order after removing duplicates
	assertOutputContains(t, output, "\nCo-authored-by: bob <bob@example.com>\nCo-authored-by: alice <alice@example.com>\nCo-authored-by: localother <localother@example.com>\n")
}

func TestAppendWithCoauthors(t *testing.T) {
	setup(t)
	configuration.Coauthors = CoauthorsMap{
		"mob-t1": "bob <bob@example.com>",
		"mob-t2": "alice <alice@example.com>",
	}

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	createFile(t, "file2.txt", "asdf")
	done(configuration)

	output := run(t, "cat", "/tmp/mob/localother/.git/SQUASH_MSG")
	// don't include the person running `mob done`
	assertOutputNotContains(t, output, "Co-authored-by: localother <localother@example.com>")
	// include everyone else in commit order after removing duplicates
	assertOutputContains(t, output, "\nCo-authored-by: bob <bob@example.com>\nCo-authored-by: alice <alice@example.com>\nCo-authored-by: local <local@example.com>\n")
}
