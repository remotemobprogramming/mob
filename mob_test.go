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
	setDefaults()
	output := captureOutput()

	version()

	assertOutputContains(t, output, versionNumber)
}

func TestStart(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)

	start()

	assertMobProgramming(t)
	assertLocalMobSessionBranch(t)
	assertRemoteMobSessionBranch(t)
}

func TestReset(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)

	reset()

	assertNotMobProgramming(t)
	assertNoLocalMobSessionBranch(t)
	assertNoRemoteMobSessionBranch(t)
}

func TestResetCommit(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)
	start()
	createFile(t, "example.txt", "content")
	next()

	reset()

	assertNotMobProgramming(t)
	assertNoLocalMobSessionBranch(t)
	assertNoRemoteMobSessionBranch(t)
}

func TestStartUnstagedChanges(t *testing.T) {
	setDefaults()
	printOutput()
	createTestbed(t)
	createFile(t, "test.txt", "content")
	mobStartIncludeUncommittedChanges = false

	start()

	assertNotMobProgramming(t)
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	setDefaults()
	printOutput()
	createTestbed(t)
	createFile(t, "test.txt", "content")
	mobStartIncludeUncommittedChanges = true

	start()

	assertMobProgramming(t)
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	setDefaults()
	printOutput()
	createTestbed(t)
	createFile(t, "example.txt", "content")
	mobStartIncludeUncommittedChanges = true

	start()

	assertMobProgramming(t)
}

func TestStartUntrackedFiles(t *testing.T) {
	setDefaults()
	printOutput()
	createTestbed(t)
	createFile(t, "example.txt", "content")
	mobStartIncludeUncommittedChanges = false

	start()

	assertNotMobProgramming(t)
}

func TestStartNextBackToMaster(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)
	start()
	createFile(t, "example.txt", "content")

	next()

	assertNotMobProgramming(t)
	assertLocalMobSessionBranch(t)
	assertRemoteMobSessionBranch(t)
}

func TestStartNextStay(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)
	mobNextStay = true
	start()
	createFile(t, "file1.txt", "asdf")

	next()

	assertMobProgramming(t)
}

func TestStartDone(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)
	assertOnBranch(t, "master")
	start()
	assertOnBranch(t, "mob-session")

	done()
	assertOnBranch(t, "master")

	assertNotMobProgramming(t)
	assertNoLocalMobSessionBranch(t)
	assertNoRemoteMobSessionBranch(t)
}

func TestStartDoneFeatureBranch(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start()
	assertOnBranch(t, "mob-session/feature1")

	done()
	assertOnBranch(t, "feature1")

	assertNotMobProgramming(t)
	assertNoLocalMobSessionBranch(t)
	assertNoRemoteMobSessionBranch(t)
}

func TestStartDoneLocalFeatureBranch(t *testing.T) {
	setDefaults()
	debug = true
	captureOutput()
	createTestbed(t)
	git("checkout", "-b", "feature1")
	assertOnBranch(t, "feature1")
	start()
	assertOnBranch(t, "feature1")
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

func printOutput() *string {
	return captureOutput()
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

func assertNoRemoteMobSessionBranch(t *testing.T) {
	if hasRemoteBranch("mob-session") {
		t.Error("should have no mob programming branch on origin")
	}
}

func assertNoLocalMobSessionBranch(t *testing.T) {
	if hasLocalBranch("mob-session") {
		t.Error("should have no mob programming branch")
	}
}

func assertNotMobProgramming(t *testing.T) {
	if isMobProgramming() {
		t.Error("should not be mob programming")
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

func assertRemoteMobSessionBranch(t *testing.T) {
	if !hasRemoteBranch("mob-session") {
		t.Error("should have mob programming branch on origin")
	}
}

func assertLocalMobSessionBranch(t *testing.T) {
	if !hasLocalBranch("mob-session") {
		t.Error("should have mob programming branch")
	}
}

func assertMobProgramming(t *testing.T) {
	if !isMobProgramming() {
		t.Error("should be mob programming")
	}
}
