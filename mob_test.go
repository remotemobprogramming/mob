package main

import (
	fmt "fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	configuration = getDefaultConfiguration()
	args := []string{"mob", "start", "--branch", "green"}
	assertEquals(t, configuration.WipBranchQualifier, "")
	command, parameters := parseArgs(args)

	assertEquals(t, "start", command)
	assertEquals(t, "", strings.Join(parameters, ""))
	assertEquals(t, "green", configuration.WipBranchQualifier)
}

func TestDetermineBranches(t *testing.T) {
	assertDetermineBranches(t, "master", "", "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "", "master", "mob-session")

	assertDetermineBranches(t, "master", "green", "master", "mob/master/green")
	assertDetermineBranches(t, "mob/master/green", "", "master", "mob/master/green")

	assertDetermineBranches(t, "feature1", "", "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1", "", "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1/green", "", "feature1", "mob/feature1/green")
	assertDetermineBranches(t, "feature1", "green", "feature1", "mob/feature1/green")
}

func assertDetermineBranches(t *testing.T, branch string, qualifier, expectedBase string, expectedWip string) {
	baseBranch, wipBranch := determineBranches(branch, qualifier)
	assertEquals(t, expectedBase, baseBranch)
	assertEquals(t, expectedWip, wipBranch)
}

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
	assertMobSessionBranches(t, "mob-session")
}

func TestStartWithMultipleExistingBranches(t *testing.T) {
	output := setup(t)

	configuration.WipBranchQualifier = "green"
	start()
	next()

	configuration.WipBranchQualifier = ""
	start()
	assertOnBranch(t, "master")
	assertOutputContains(t, output, "qualified mob branches detected")
}

func TestStartWithMultipleExistingBranchesAndEmptyWipBranchQualifier(t *testing.T) {
	output := setup(t)

	configuration.WipBranchQualifier = "green"
	start()
	next()

	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = true
	start()
	assertOnBranch(t, "mob-session")
	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartWithMultipleExistingBranchesWithStay(t *testing.T) {
	output := setup(t)
	configuration.MobNextStay = true

	configuration.WipBranchQualifier = "green"
	assertOnBranch(t, "master")
	start()
	assertOnBranch(t, "mob/master/green")
	next()
	assertOnBranch(t, "mob/master/green")

	configuration.WipBranchQualifier = ""
	start()
	assertOnBranch(t, "mob/master/green")
	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartNextWithBranch(t *testing.T) {
	setup(t)
	assertOnBranch(t, "master")
	configuration.WipBranchQualifier = "green"

	start()
	assertOnBranch(t, "mob/master/green")
	assertMobSessionBranches(t, "mob/master/green")
	configuration.WipBranchQualifier = ""

	next()
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = "green"
	reset()
	assertNoMobSessionBranches(t, "mob/master/green")
}

func TestStartNextStartWithBranch(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "green"
	configuration.MobNextStay = true
	assertOnBranch(t, "master")

	start()
	assertOnBranch(t, "mob/master/green")

	next()
	assertOnBranch(t, "mob/master/green")

	start()
	assertOnBranch(t, "mob/master/green")
}

func TestStartNextOnFeatureWithBranch(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "green"
	configuration.Debug = true
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")

	start()
	assertOnBranch(t, "mob/feature1/green")

	next()
	assertOnBranch(t, "feature1")
}

func TestReset(t *testing.T) {
	setup(t)

	reset()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestResetCommit(t *testing.T) {
	setup(t)
	start()
	createFile(t, "example.txt", "content")
	next()
	assertMobSessionBranches(t, "mob-session")

	reset()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartUnstagedChanges(t *testing.T) {
	output := setup(t)
	createFile(t, "test.txt", "content")
	configuration.MobStartIncludeUncommittedChanges = false

	start()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
	assertOutputContains(t, output, "fix with 'mob start --include-uncommitted-changes'")
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	setup(t)
	createFile(t, "test.txt", "content")
	configuration.MobStartIncludeUncommittedChanges = true

	start()

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, "mob-session")
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	setup(t)
	createFile(t, "example.txt", "content")
	configuration.MobStartIncludeUncommittedChanges = true

	start()

	assertOnBranch(t, "mob-session")
}

func TestStartUntrackedFiles(t *testing.T) {
	setup(t)
	createFile(t, "example.txt", "content")
	configuration.MobStartIncludeUncommittedChanges = false

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
	assertMobSessionBranches(t, "mob-session")
}

func TestStartNextStay(t *testing.T) {
	setup(t)
	configuration.MobNextStay = true
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
	assertNoMobSessionBranches(t, "mob-session")
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
	assertNoMobSessionBranches(t, "mob-session")
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
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDoneLocalFeatureBranch(t *testing.T) {
	output := setup(t)
	git("checkout", "-b", "feature1")

	start()

	assertOnBranch(t, "feature1")
	assertOutputContains(t, output, "fix with 'git push origin feature1 --set-upstream'")
}

func TestBothCreateNonemptyCommitWithNext(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start()
	createFile(t, "file1.txt", "asdf")

	setWorkingDir("/tmp/mob/localother")
	start()
	createFile(t, "file2.txt", "asdf")

	setWorkingDir("/tmp/mob/local")
	next()

	setWorkingDir("/tmp/mob/localother")
	// next() not possible, would fail
	git("pull")
	next()

	setWorkingDir("/tmp/mob/local")
	start()
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")

	setWorkingDir("/tmp/mob/localother")
	start()
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")
}

func TestNothingToCommitCreatesNoCommits(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start()
	assertCommits(t, 1)

	setWorkingDir("/tmp/mob/localother")
	start()
	assertCommits(t, 1)

	setWorkingDir("/tmp/mob/local")
	next()

	setWorkingDir("/tmp/mob/localother")
	next()

	setWorkingDir("/tmp/mob/local")
	start()
	assertCommits(t, 1)

	setWorkingDir("/tmp/mob/localother")
	start()
	assertCommits(t, 1)
}

func TestConflictingMobSessions(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start()
	createFile(t, "example.txt", "content")
	next()

	setWorkingDir("/tmp/mob/localother")
	start()
	next()

	setWorkingDir("/tmp/mob/local")
	start()
	done()
	git("commit", "-m", "\"finished mob session\"")

	setWorkingDir("/tmp/mob/local")
	start()
	createFile(t, "example2.txt", "content")
	next()

	setWorkingDir("/tmp/mob/localother")
	start()
}

func TestDoneMergeConflict(t *testing.T) {
	output := setup(t)

	setWorkingDir("/tmp/mob/local")
	start()
	createFile(t, "example.txt", "content")
	next()

	setWorkingDir("/tmp/mob/localother")
	createFile(t, "example.txt", "asdf")
	git("add", "--all")
	git("commit", "-m", "\"asdf\"")
	git("push")

	setWorkingDir("/tmp/mob/local")
	start()
	done()
	assertOutputContains(t, output, "Automatic merge failed; fix conflicts and then commit the result.")
}

func TestDoneMerge(t *testing.T) {
	output := setup(t)

	setWorkingDir("/tmp/mob/local")
	start()
	createFile(t, "example.txt", "content")
	next()

	setWorkingDir("/tmp/mob/localother")
	createFile(t, "example2.txt", "asdf")
	git("add", "--all")
	git("commit", "-m", "\"asdf\"")
	git("push")

	setWorkingDir("/tmp/mob/local")
	start()
	done()
	assertOutputContains(t, output, "git commit -m 'describe the changes'")
}

func setup(t *testing.T) *string {
	configuration = getDefaultConfiguration()
	output := captureOutput()
	createTestbed(t)
	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
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

	setWorkingDir("/tmp/mob/local")
	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func setWorkingDir(dir string) {
	workingDir = dir
	say("\nSET WORKING DIR TO " + dir + "\n======================\n")
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

func assertOutputNotContains(t *testing.T, output *string, notContains string) {
	if strings.Contains(*output, notContains) {
		t.Error("expected output to not contain " + notContains + ", but it does.\nOutput:\n" + *output)
	}
}

func assertEquals(t *testing.T, expected string, actual string) {
	if expected != actual {
		t.Error("expected " + expected + " but got " + actual)
		debug.PrintStack()
	}
}

func assertMobSessionBranches(t *testing.T, branch string) {
	if !hasRemoteBranch(branch) {
		t.Error("expected to have origin/" + branch + " but got none")
	}
	if !hasLocalBranch(branch) {
		t.Error("expected to have local " + branch + " but got none")
	}
}

func assertNoMobSessionBranches(t *testing.T, branch string) {
	if hasRemoteBranch(branch) {
		t.Error("expected to not have origin/" + branch + " but got it")
	}
	if hasLocalBranch(branch) {
		t.Error("expected to not  have local " + branch + " but got it")
	}
}
