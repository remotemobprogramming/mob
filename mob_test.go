package main

import (
	fmt "fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	configuration = getDefaultConfiguration()
	args := []string{"mob", "start", "--branch", "green"}
	equals(t, configuration.WipBranchQualifier, "")
	command, parameters := parseArgs(args)

	equals(t, "start", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "green", configuration.WipBranchQualifier)
}

func TestParseArgsMessage(t *testing.T) {
	configuration = getDefaultConfiguration()
	args := []string{"mob", "next", "--message", "ci-skip"}
	equals(t, configuration.WipBranchQualifier, "")
	command, parameters := parseArgs(args)

	equals(t, "next", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "ci-skip", configuration.WipCommitMessage)
}

func TestDetermineBranches(t *testing.T) {
	configuration = getDefaultConfiguration()
	configuration.WipBranchQualifierSeparator = "-"

	assertDetermineBranches(t, "master", "", "", "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "", "", "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "green", "", "master", "mob-session")

	assertDetermineBranches(t, "master", "green", "", "master", "mob/master-green")
	assertDetermineBranches(t, "mob/master-green", "", "", "master", "mob/master-green")

	assertDetermineBranches(t, "feature1", "", "", "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1", "", "", "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1-green", "", "", "feature1", "mob/feature1-green")
	assertDetermineBranches(t, "feature1", "green", "", "feature1", "mob/feature1-green")

	assertDetermineBranches(t, "feature/test", "", "feature/test", "feature/test", "mob/feature/test")
	assertDetermineBranches(t, "mob/feature/test", "", "feature/test\nmob/feature/test", "feature/test", "mob/feature/test")

	assertDetermineBranches(t, "feature/test-ch", "", "DPL-2638-update-apis\nDPL-2814-create-project\nfeature/test-ch\nfix/smallChanges\nmaster\npipeship/pipelineupdate-pipeship-pipeline.yaml\n", "feature/test-ch", "mob/feature/test-ch")
}

func assertDetermineBranches(t *testing.T, branch string, qualifier string, branches string, expectedBase string, expectedWip string) {
	baseBranch, wipBranch := determineBranches(branch, qualifier, branches)
	equals(t, expectedBase, baseBranch)
	equals(t, expectedWip, wipBranch)
}

func TestEnvironmentVariables(t *testing.T) {
	configuration = getDefaultConfiguration()

	os.Setenv("MOB_REMOTE_NAME", "GITHUB")
	defer os.Unsetenv("MOB_REMOTE_NAME")

	os.Setenv("MOB_DEBUG", "true")
	defer os.Unsetenv("MOB_DEBUG")

	configuration = parseEnvironmentVariables(getDefaultConfiguration())
	equals(t, "GITHUB", configuration.RemoteName)
	equals(t, true, configuration.Debug)
}

func TestEnvironmentVariablesEmptyString(t *testing.T) {
	configuration = getDefaultConfiguration()

	os.Setenv("MOB_REMOTE_NAME", "")
	defer os.Unsetenv("MOB_REMOTE_NAME")

	configuration = parseEnvironmentVariables(getDefaultConfiguration())
	equals(t, "origin", configuration.RemoteName)
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
	assertOnBranch(t, "mob/master-green")
	next()
	assertOnBranch(t, "master")

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
	assertOnBranch(t, "mob/master-green")
	next()
	assertOnBranch(t, "mob/master-green")

	configuration.WipBranchQualifier = ""
	start()
	assertOnBranch(t, "mob/master-green")
	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartNextWithBranch(t *testing.T) {
	setup(t)
	assertOnBranch(t, "master")
	configuration.WipBranchQualifier = "green"

	start()
	assertOnBranch(t, "mob/master-green")
	assertMobSessionBranches(t, "mob/master-green")
	configuration.WipBranchQualifier = ""

	next()
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = "green"
	reset()
	assertNoMobSessionBranches(t, "mob/master-green")
}

func TestStartNextStartWithBranch(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "green"
	configuration.MobNextStay = true
	assertOnBranch(t, "master")

	start()
	assertOnBranch(t, "mob/master-green")

	next()
	assertOnBranch(t, "mob/master-green")

	start()
	assertOnBranch(t, "mob/master-green")
}

func TestStartNextOnFeatureWithBranch(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "green"
	configuration.Debug = true
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")

	start()
	assertOnBranch(t, "mob/feature1-green")

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
	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
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

	equals(t, strings.TrimSpace(silentgit("log", "--format=%B", "-n", "1", "HEAD")), configuration.WipCommitMessage)
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
	assertOutputContains(t, output, "git push origin feature1 --set-upstream")
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

func TestStartNextPushManualCommits(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")

	start()
	createFile(t, "example.txt", "content")
	git("add", "--all")
	git("commit", "-m", "asdf")
	next()

	setWorkingDir("/tmp/mob/localother")
	start()
	assertFileExist(t, "example.txt")
}

func TestStartNextPushManualCommitsFeatureBranch(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")

	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start()
	assertOnBranch(t, "mob/feature1")

	createFile(t, "example.txt", "content")
	git("add", "--all")
	git("commit", "-m", "asdf")
	next()

	setWorkingDir("/tmp/mob/localother")
	git("fetch")
	git("checkout", "feature1")
	start()
	assertFileExist(t, "example.txt")
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

func TestConflictingMobSessionsNextStay(t *testing.T) {
	setup(t)
	configuration.MobNextStay = true

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
	configuration.WipBranchQualifierSeparator = "-"
	output := captureOutput()
	createTestbed(t)
	assertOnBranch(t, "master")
	equals(t, "master", gitBranches())
	equals(t, "origin/master", gitRemoteBranches())
	assertNoMobSessionBranches(t, "mob-session")
	return output
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

func assertCommits(t *testing.T, commits int) {
	result := silentgit("rev-list", "--count", "HEAD")
	number, _ := strconv.Atoi(strings.TrimSpace(result))
	if number != commits {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, strconv.Itoa(commits)+" commits in "+workingDir, strconv.Itoa(number)+" commits in "+workingDir)
		t.FailNow()
	}
}

func assertFileExist(t *testing.T, filename string) {
	path := workingDir + "/" + filename
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "existing file "+path, "no file at "+path)
		t.FailNow()
	}
}

func createFile(t *testing.T, filename string, content string) {
	d1 := []byte(content)
	err := ioutil.WriteFile(workingDir+"/"+filename, d1, 0644)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "creating file "+filename+" with content "+content, "error")
		t.FailNow()
	}
}

func assertOnBranch(t *testing.T, branch string) {
	currentBranch := gitCurrentBranch()
	if currentBranch != branch {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "on branch "+branch, "on branch "+currentBranch)
		t.FailNow()
	}
}

func assertOutputContains(t *testing.T, output *string, contains string) {
	currentOutput := *output
	if !strings.Contains(currentOutput, contains) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "output contains '"+contains+"'", currentOutput)
		t.FailNow()
	}
}

func assertOutputNotContains(t *testing.T, output *string, notContains string) {
	if strings.Contains(*output, notContains) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "output not contains "+notContains, output)
		t.FailNow()
	}
}

func assertMobSessionBranches(t *testing.T, branch string) {
	if !hasRemoteBranch(branch) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, configuration.RemoteName+"/"+branch, "none")
		t.FailNow()
	}
	if !hasLocalBranch(branch) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, branch, "none")
		t.FailNow()
	}
}

func assertNoMobSessionBranches(t *testing.T, branch string) {
	if hasRemoteBranch(branch) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "none", configuration.RemoteName+"/"+branch)
		t.FailNow()
	}
	if hasLocalBranch(branch) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "none", branch)
		t.FailNow()
	}
}

func equals(t *testing.T, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		t.Log(string(debug.Stack()))
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		t.FailNow()
	}
}
