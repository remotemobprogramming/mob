package main

import (
	fmt "fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	output := captureOutput()

	version()

	assertOutputContains(t, output, versionNumber)
}

func TestStart(t *testing.T) {
	captureOutput()
	createTestbed(t)

	start()

	assertMobProgramming(t)
	assertLocalMobSessionBranch(t)
	assertRemoteMobSessionBranch(t)
}

func TestReset(t *testing.T) {
	debug = true
	captureOutput()
	createTestbed(t)

	reset()

	assertNotMobProgramming(t)
	assertNoLocalMobSessionBranch(t)
	assertNoRemoteMobSessionBranch(t)
}

func TestResetCommit(t *testing.T) {
	debug = true
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

func TestStartWithUncommittedChanges(t *testing.T) {
	debug = false
	printOutput()
	createTestbed(t)
	createFile(t, "test.txt", "content")
	mobStartIncludeUncommittedChanges = false

	start()

	assertNotMobProgramming(t)
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	printOutput()
	createTestbed(t)
	createFile(t, "test.txt", "content")
	mobStartIncludeUncommittedChanges = true

	start()

	assertMobProgramming(t)
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	t.Skip("not yet implemented")

	printOutput()
	createTestbed(t)
	createFile(t, "example.txt", "content")
	mobStartIncludeUncommittedChanges = true

	start()

	assertMobProgramming(t)
}

func TestStartNext(t *testing.T) {
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
	captureOutput()
	createTestbed(t)
	mobNextStay = true
	start()

	next()

	assertMobProgramming(t)
}

func TestStartDone(t *testing.T) {
	captureOutput()
	createTestbed(t)
	start()

	done()

	assertNotMobProgramming(t)
	assertNoLocalMobSessionBranch(t)
	assertNoRemoteMobSessionBranch(t)
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
	if hasMobProgrammingBranch() {
		t.Error("should have no mob programming branch")
	}
	if hasMobProgrammingBranchOrigin() {
		t.Error("should have no mob programming branch on origin")
	}
}

func assertNoRemoteMobSessionBranch(t *testing.T) {
	if hasMobProgrammingBranchOrigin() {
		t.Error("should have no mob programming branch on origin")
	}
}

func assertNoLocalMobSessionBranch(t *testing.T) {
	if hasMobProgrammingBranch() {
		t.Error("should have no mob programming branch")
	}
}

func assertNotMobProgramming(t *testing.T) {
	if isMobProgramming() {
		t.Error("should not be mob programming")
	}
}

func assertOutputContains(t *testing.T, output *string, contains string) {
	if !strings.Contains(*output, contains) {
		t.Error("expected output to contain " + contains + ", but does not.\nOutput:\n" + *output)
	}
}

func assertRemoteMobSessionBranch(t *testing.T) {
	if !hasMobProgrammingBranchOrigin() {
		t.Error("should have mob programming branch on origin")
	}
}

func assertLocalMobSessionBranch(t *testing.T) {
	if !hasMobProgrammingBranch() {
		t.Error("should have mob programming branch")
	}
}

func assertMobProgramming(t *testing.T) {
	if !isMobProgramming() {
		t.Error("should be mob programming")
	}
}
