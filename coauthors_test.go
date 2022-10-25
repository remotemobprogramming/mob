package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStartDoneCoAuthors(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file3.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	createFile(t, "file2.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file4.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/bob")
	start(configuration)
	createFile(t, "file5.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)

	outputFile := filepath.Join(tempDir, "local", ".git", "SQUASH_MSG")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		failWithFailure(t, fmt.Sprintf("reading file %s failed with %v", outputFile, err), "error")
	}
	output := string(content)

	// don't include the person running `mob done`
	assertOutputNotContains(t, &output, "Co-authored-by: local <local@example.com>")
	// include everyone else in commit order after removing duplicates
	assertOutputContains(t, &output, "\nCo-authored-by: bob <bob@example.com>\nCo-authored-by: alice <alice@example.com>\nCo-authored-by: localother <localother@example.com>\n")
}

func TestCreateCommitMessage(t *testing.T) {
	equals(t, `

# automatically added all co-authors from WIP commits
# add missing co-authors manually
Co-authored-by: Alice <alice@example.com>
Co-authored-by: Bob <bob@example.com>
`, createCommitMessage([]Author{"Alice <alice@example.com>", "Bob <bob@example.com>"}))
}

func TestSortByLength(t *testing.T) {
	slice := []string{"aa", "b"}

	sortByLength(slice)

	equals(t, []string{"b", "aa"}, slice)
}

func TestRemoveDuplicateValues(t *testing.T) {
	slice := []string{"aa", "b", "c", "b"}

	actual := removeDuplicateValues(slice)

	equals(t, []string{"aa", "b", "c"}, actual)
}
