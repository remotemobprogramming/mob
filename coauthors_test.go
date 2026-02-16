package main

import (
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

	output := readFile(t, filepath.Join(tempDir, "local", ".git", "SQUASH_MSG"))

	// don't include the person running `mob done`
	assertOutputNotContains(t, &output, "Co-authored-by: local <local@example.com>")
	// include everyone else in commit order after removing duplicates
	assertOutputContains(t, &output, "\nCo-authored-by: bob <bob@example.com>\nCo-authored-by: alice <alice@example.com>\nCo-authored-by: localother <localother@example.com>\n")
}
