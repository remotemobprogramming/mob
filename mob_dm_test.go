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
	start()
	assertOnBranch(t, "mob/feature1")
	createFile(t, "dm_example.txt", "content")

	// act
	done()
	// assertOnBranch(t, "feature1")
	// assertNoMobSessionBranches(t, "mob-session")
	// assertFileExist(t, "dm_example.txt")
	git("reset", "--hard")

	// assert
	assertFileExist(t, "dm_example.txt")

	// cleanup
	reset()
}
