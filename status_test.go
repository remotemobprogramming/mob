package main

import (
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"strconv"
	"testing"
	"time"
)

func TestExecuteKicksOffStatus(t *testing.T) {
	output, _ := setup(t)

	execute("status", []string{}, config.GetDefaultConfiguration())

	assertOutputContains(t, output, "you are on base branch 'master'")
}

func TestStatusMobProgramming(t *testing.T) {
	output, configuration := setup(t)
	start(configuration)

	status(configuration)

	assertOutputContains(t, output, "you are on wip branch mob-session")
}

func TestStatusWithMoreThan5LinesOfLog(t *testing.T) {
	output, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)

	for i := 0; i < 6; i++ {
		createFile(t, "test"+strconv.Itoa(i)+".txt", "contentIrrelevant")
		next(configuration)
	}

	status(configuration)
	// 6 wip commits + 1 start commit
	assertOutputContains(t, output, "wip branch 'mob-session' contains 7 commits.")
}

func TestStatusDetectsWipBranches(t *testing.T) {
	output, configuration := setup(t)
	start(configuration)
	createFile(t, "test.txt", "contentIrrelevant")
	next(configuration)
	git("checkout", "master")
	time.Sleep(2 * time.Second)

	status(configuration)

	assertOutputContains(t, output, "remote wip branches detected:\n  - origin/mob-session")
	assertOutputContains(t, output, " seconds ago)")
}
