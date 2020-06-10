package main

import (
	fmt "fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	output := setup(t)

	version()

	assertOutputContains(t, output, versionNumber)
}

func TestStatusNotMobProgramming(t *testing.T) {
	output := setup(t)

	status()

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestNextNotMobProgramming(t *testing.T) {
	output := setup(t)

	next()

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestDoneNotMobProgramming(t *testing.T) {
	output := setup(t)

	done()

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestStatusMobProgramming(t *testing.T) {
	output := setup(t)
	start()

	status()

	assertOutputContains(t, output, "you are mob programming")
}

func TestExecuteKicksOffStatus(t *testing.T) {
	output := setup(t)
	var parameters []string

	execute("status", parameters)

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestExecuteInvalidCommandKicksOffHelp(t *testing.T) {
	output := setup(t)
	var parameters []string

	execute("whatever", parameters)

	assertOutputContains(t, output, "USAGE")
}

func TestStart(t *testing.T) {
	setup(t)

	start()

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t)
}

func TestReset(t *testing.T) {
	setup(t)

	reset()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t)
}

func TestResetCommit(t *testing.T) {
	setup(t)
	start()
	createFile(t, "example.txt", "content")
	next()
	assertMobSessionBranches(t)

	reset()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t)
}

func TestStartUnstagedChanges(t *testing.T) {
	output := setup(t)
	createFile(t, "test.txt", "content")
	mobStartIncludeUncommittedChanges = false

	start()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t)
	assertOutputContains(t, output, "fix with 'mob start --include-uncommitted-changes'")
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	setup(t)
	createFile(t, "test.txt", "content")
	mobStartIncludeUncommittedChanges = true

	start()

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t)
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	setup(t)
	createFile(t, "example.txt", "content")
	mobStartIncludeUncommittedChanges = true

	start()

	assertOnBranch(t, "mob-session")
}

func TestStartUntrackedFiles(t *testing.T) {
	setup(t)
	createFile(t, "example.txt", "content")
	mobStartIncludeUncommittedChanges = false

	start()

	assertOnBranch(t, "master")
}

func TestStartNextBackToMaster(t *testing.T) {
	setup(t)
	start()
	createFile(t, "example.txt", "content")
	assertOnBranch(t, "mob-session")

	next()

	assertOnBranch(t, "master")
	assertMobSessionBranches(t)
}

func TestStartNextStay(t *testing.T) {
	setup(t)
	mobNextStay = true
	start()
	createFile(t, "file1.txt", "asdf")
	assertOnBranch(t, "mob-session")

	next()

	assertOnBranch(t, "mob-session")
}

func TestStartDone(t *testing.T) {
	setup(t)
	start()
	assertOnBranch(t, "mob-session")

	done()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t)
}

func TestStartDoneFeatureBranch(t *testing.T) {
	setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start()
	assertOnBranch(t, "mob/feature1")

	done()

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t)
}

func TestStartNextFeatureBranch(t *testing.T) {
	setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start()
	assertOnBranch(t, "mob/feature1")

	next()

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t)
}

func TestStartDoneLocalFeatureBranch(t *testing.T) {
	output := setup(t)
	git("checkout", "-b", "feature1")

	start()

	assertOnBranch(t, "feature1")
	assertOutputContains(t, output, "fix with 'git push origin feature1 --set-upstream'")
}

func setup(t *testing.T) *string {
	setDefaults()
	output := captureOutput()
	createTestbed(t)
	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t)
	return output
}

func assertCommits(t *testing.T, commits int) {
	result := silentgit("rev-list", "--count", "HEAD")
	number, _ := strconv.Atoi(strings.TrimSpace(result))
	if number != commits {
		t.Error("expected " + strconv.Itoa(commits) + " commits but got " + strconv.Itoa(number) + " in " + workingDir)
	}
}

func assertFileExist(t *testing.T, filename string) {
	path := workingDir + "/" + filename
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file " + path + " doesn't exist")
	}
}

func createFile(t *testing.T, filename string, content string) {
	d1 := []byte(content)
	err := ioutil.WriteFile(workingDir+"/"+filename, d1, 0644)
	if err != nil {
		t.Error("Could not create file " + filename + " with content " + content)
	}
}

func captureOutput() *string {
	messages := ""
	printToConsole = func(text string) {
		fmt.Print(text)
		messages += text
	}
	return &messages
}

func run(t *testing.T, name string, args ...string) {
	commandString, output, err := runCommand(name, args...)
	if err != nil {
		fmt.Println(commandString)
		fmt.Println(output)
		fmt.Println(err.Error())
		t.Error("command " + commandString + " failed")
	}
}

func createTestbed(t *testing.T) {
	workingDir = ""
	run(t, "./create-testbed")

	workingDir = "/tmp/mob/local"
	if isMobProgramming() {
		t.Error("should not be mob programming")
	}
	if hasLocalBranch("mob-session") {
		t.Error("should have no mob programming branch")
	}
	if hasRemoteBranch("mob-session") {
		t.Error("should have no mob programming branch on origin")
	}
}

func assertNoMobSessionBranches(t *testing.T) {
	if hasRemoteBranch("mob-session") {
		t.Error("should have no mob programming branch on origin")
	}
	if hasLocalBranch("mob-session") {
		t.Error("should have no mob programming branch")
	}
}

func assertOnBranch(t *testing.T, branch string) {
	currentBranch := gitCurrentBranch()
	if currentBranch != branch {
		t.Error("should be on branch " + branch + " but is on branch " + currentBranch)
	}
}

func assertOutputContains(t *testing.T, output *string, contains string) {
	if !strings.Contains(*output, contains) {
		t.Error("expected output to contain " + contains + ", but does not.\nOutput:\n" + *output)
	}
}

func assertMobSessionBranches(t *testing.T) {
	if !hasRemoteBranch("mob-session") {
		t.Error("should have mob programming branch on origin")
	}
	if !hasLocalBranch("mob-session") {
		t.Error("should have mob programming branch")
	}
}
