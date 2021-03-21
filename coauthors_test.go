package main

import (
	"testing"
)

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

