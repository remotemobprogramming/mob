package main

import (
	"testing"
)

func TestStartCommitDoneOnFeatureBranch(t *testing.T) {
	// arrange
	setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	assertCommits(t, 1) // feature1 branch fresh state
	start()
	assertOnBranch(t, "mob/feature1")
	createFile(t, "example.txt", "content")

	// act
	done()
	git("reset", "--hard")

	// assert
	assertOnBranch(t, "feature1")
	assertCommits(t, 2) // feature1 branch should have a 2nd commit
	assertFileExist(t, "example.txt")

	// cleanup
	reset()
}
