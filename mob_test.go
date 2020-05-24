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
	assertMobProgramming(t)

	reset()

	assertMobProgramming(t)
	assertLocalMobSessionBranch(t)
	assertRemoteMobSessionBranch(t)
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
	git("checkout", baseBranch)

	assertNotMobProgramming(t)
	assertLocalMobSessionBranch(t)
	assertRemoteMobSessionBranch(t)
}

func TestStartNext(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)
	start()
	createFile(t, "file1.txt", "asdf")

	next()

	assertMobProgramming(t)
}

func TestStartDone(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)
	start()

	done()

	assertNotMobProgramming(t)
	assertNoLocalMobSessionBranch(t)
	assertNoRemoteMobSessionBranch(t)
}

func TestConflictingMobSessions(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)

	workingDir = "/tmp/mob/local"
	start()
	createFile(t, "example.txt", "content")
	next()

	workingDir = "/tmp/mob/localother"
	start()
	next()

	workingDir = "/tmp/mob/local"
	start()
	done()
	git("commit", "-m", "\"finished mob session\"")

	workingDir = "/tmp/mob/local"
	start()
	createFile(t, "example2.txt", "content")
	next()

	workingDir = "/tmp/mob/localother"
	git("fetch", "origin")
	git("status", "--untracked-files=no")
	start()
}

func TestDoneMergeConflict(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)

	workingDir = "/tmp/mob/local"
	start()
	createFile(t, "example.txt", "content")
	next()

	workingDir = "/tmp/mob/localother"
	createFile(t, "example.txt", "asdf")
	git("add", "--all")
	git("commit", "-m", "\"asdf\"")
	git("push")

	workingDir = "/tmp/mob/local"
	start()
	done()
	git("add", "--all") // necessary
	git("commit", "-m", "\"finished mob session\"")
}

func TestDoneMerge(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)

	workingDir = "/tmp/mob/local"
	start()
	createFile(t, "example.txt", "content")
	next()

	workingDir = "/tmp/mob/localother"
	createFile(t, "example2.txt", "asdf")
	git("add", "--all")
	git("commit", "-m", "\"asdf\"")
	git("push")

	workingDir = "/tmp/mob/local"
	start()
	done()
	git("commit", "-m", "\"finished mob session\"")
}

func TestDonePull(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)

	workingDir = "/tmp/mob/local"
	createFile(t, "example3.txt", "asdf")
	git("add", "--all")
	git("commit", "-m", "\"asdf\"")
	start()
	createFile(t, "example.txt", "content")
	next()

	workingDir = "/tmp/mob/localother"
	createFile(t, "example2.txt", "asdf")
	git("add", "--all")
	git("commit", "-m", "\"asdf\"")
	git("push")

	workingDir = "/tmp/mob/local"
	start()
	done()
	git("pull")
	start()
	done()
	git("commit", "-m", "\"finished mob session\"")
}

func TestBothCreateEmptyCommitWithNext(t *testing.T) {
	setDefaults()
	debug = true
	captureOutput()
	createTestbed(t)

	workingDir = "/tmp/mob/local"
	start()
	assertCommits(t, 1)

	workingDir = "/tmp/mob/localother"
	start()
	assertCommits(t, 1)

	workingDir = "/tmp/mob/local"
	next()
	// TODO WHY?????
	assertCommits(t, 2)

	workingDir = "/tmp/mob/localother"
	next()
	git("pull")
	git("push")
	assertCommits(t, 3)

	workingDir = "/tmp/mob/local"
	start()
	assertCommits(t, 3)

	workingDir = "/tmp/mob/localother"
	start()
	assertCommits(t, 3)
}

func TestBothCreateNonemptyCommitWithNext(t *testing.T) {
	setDefaults()
	captureOutput()
	createTestbed(t)

	workingDir = "/tmp/mob/local"
	start()
	createFile(t, "file1.txt", "asdf")

	workingDir = "/tmp/mob/localother"
	start()
	createFile(t, "file2.txt", "asdf")

	workingDir = "/tmp/mob/local"
	next()

	workingDir = "/tmp/mob/localother"
	next()
	git("pull")
	git("push")

	workingDir = "/tmp/mob/local"
	start()
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")

	workingDir = "/tmp/mob/localother"
	start()
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")
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
