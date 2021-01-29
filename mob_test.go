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
	equals(t, configuration.WipBranchQualifier, "")

	command, parameters := parseArgs([]string{"mob", "start", "--branch", "green"})

	equals(t, "start", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "green", configuration.WipBranchQualifier)
}

func TestParseArgsDoneSquash(t *testing.T) {
	configuration = getDefaultConfiguration()
	equals(t, true, configuration.MobDoneSquash)

	command, parameters := parseArgs([]string{"mob", "done", "--no-squash"})

	equals(t, "done", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, false, configuration.MobDoneSquash)
}

func TestParseArgsMessage(t *testing.T) {
	configuration = getDefaultConfiguration()
	equals(t, configuration.WipBranchQualifier, "")

	command, parameters := parseArgs([]string{"mob", "next", "--message", "ci-skip"})

	equals(t, "next", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "ci-skip", configuration.WipCommitMessage)
}

func TestDetermineBranches(t *testing.T) {
	configuration = getDefaultConfiguration()
	configuration.WipBranchQualifierSeparator = "-"
	configuration.Debug = true

	assertDetermineBranches(t, "master", "", []string{}, "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "", []string{}, "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "green", []string{}, "master", "mob-session")

	assertDetermineBranches(t, "master", "green", []string{}, "master", "mob/master-green")
	assertDetermineBranches(t, "mob/master-green", "", []string{}, "master", "mob/master-green")

	assertDetermineBranches(t, "master", "test-branch", []string{}, "master", "mob/master-test-branch")
	assertDetermineBranches(t, "mob/master-test-branch", "test-branch", []string{}, "master", "mob/master-test-branch")

	assertDetermineBranches(t, "feature1", "", []string{}, "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1", "", []string{}, "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1-green", "", []string{}, "feature1", "mob/feature1-green")
	assertDetermineBranches(t, "feature1", "green", []string{}, "feature1", "mob/feature1-green")

	assertDetermineBranches(t, "feature/test", "", []string{"feature/test"}, "feature/test", "mob/feature/test")
	assertDetermineBranches(t, "mob/feature/test", "", []string{"feature/test", "mob/feature/test"}, "feature/test", "mob/feature/test")

	assertDetermineBranches(t, "feature/test-ch", "", []string{"DPL-2638-update-apis", "DPL-2814-create-project", "feature/test-ch", "fix/smallChanges", "master", "pipeship/pipelineupdate-pipeship-pipeline.yaml"}, "feature/test-ch", "mob/feature/test-ch")
}

func assertDetermineBranches(t *testing.T, branch string, qualifier string, branches []string, expectedBase string, expectedWip string) {
	configuration.WipBranchQualifier = qualifier
	baseBranch, wipBranch := determineBranches(branch, branches)
	equals(t, expectedBase, baseBranch)
	equals(t, expectedWip, wipBranch)
}

func TestRemoveWipPrefix(t *testing.T) {
	equals(t, "master-green", removeWipPrefix("mob/master-green"))
	equals(t, "master-green-blue", removeWipPrefix("mob/master-green-blue"))
	equals(t, "main-branch", removeWipPrefix("mob/main-branch"))
}

// todo it should not be possible to set configuration.WipBranchQualifier without setting configuration.WipBranchQualifierSet to true. could encapsulate this behaviour in setter and prevent access
// todo maybe extract function for each, but feels awkward with so many parameters (hard to read imo)
func TestRemoveSuffix(t *testing.T) {
	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "green"
	configuration.WipBranchQualifierSet = true
	equals(t, "master", removeSuffix("master-green"))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "test-branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master", removeSuffix("master-test-branch"))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master-test", removeSuffix("master-test-branch"))

	configuration.WipBranchQualifierSeparator = "/-/"
	configuration.WipBranchQualifier = "branch-qualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main", removeSuffix("main/-/branch-qualifier"))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branchqualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main/branchqualifier", removeSuffix("main/branchqualifier"))

	configuration.WipBranchQualifierSeparator = ""
	configuration.WipBranchQualifier = "branchqualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main", removeSuffix("mainbranchqualifier"))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = false
	equals(t, "main", removeSuffix("main"))
}

func TestMobRemoteNameEnvironmentVariable(t *testing.T) {
	configuration = setEnvVarAndParse("MOB_REMOTE_NAME", "GITHUB")

	equals(t, "GITHUB", configuration.RemoteName)
}

func TestMobRemoteNameEnvironmentVariableEmptyString(t *testing.T) {
	configuration = setEnvVarAndParse("MOB_REMOTE_NAME", "")

	equals(t, "origin", configuration.RemoteName)
}

func TestBooleanEnvironmentVariables(t *testing.T) {
	assertBoolEnvVarParsed(t, "MOB_DONE_SQUASH", true, Configuration.GetMobDoneSquash)
	assertBoolEnvVarParsed(t, "MOB_DEBUG", false, Configuration.GetDebug)
	assertBoolEnvVarParsed(t, "MOB_START_INCLUDE_UNCOMMITTED_CHANGES", false, Configuration.GetMobStartIncludeUncommittedChanges)
	assertBoolEnvVarParsed(t, "MOB_NEXT_STAY", true, Configuration.GetMobNextStay)
}

func assertBoolEnvVarParsed(t *testing.T, envVar string, defaultValue bool, actual func(Configuration) bool) {
	t.Run(envVar, func(t *testing.T) {
		assertEnvVarParsed(t, envVar, "", defaultValue, boolToInterface(actual))
		assertEnvVarParsed(t, envVar, "true", true, boolToInterface(actual))
		assertEnvVarParsed(t, envVar, "false", false, boolToInterface(actual))
		assertEnvVarParsed(t, envVar, "garbage", defaultValue, boolToInterface(actual))
	})
}

func assertEnvVarParsed(t *testing.T, variable string, value string, expected interface{}, actual func(Configuration) interface{}) {
	t.Run(fmt.Sprintf("%s=\"%s\"->(expects:%t)", variable, value, expected), func(t *testing.T) {
		configuration = setEnvVarAndParse(variable, value)
		equals(t, expected, actual(configuration))
	})
}

func setEnvVarAndParse(variable string, value string) Configuration {
	os.Setenv(variable, value)
	defer os.Unsetenv(variable)

	return parseEnvironmentVariables(getDefaultConfiguration())
}

func boolToInterface(actual func(Configuration) bool) func(c Configuration) interface{} {
	return func(c Configuration) interface{} {
		return actual(c)
	}
}

func (c Configuration) GetMobDoneSquash() bool {
	return c.MobDoneSquash
}

func (c Configuration) GetDebug() bool {
	return c.Debug
}

func (c Configuration) GetMobStartIncludeUncommittedChanges() bool {
	return c.MobStartIncludeUncommittedChanges
}

func (c Configuration) GetMobNextStay() bool {
	return c.MobNextStay
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

	execute("status", []string{})

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestExecuteInvalidCommandKicksOffHelp(t *testing.T) {
	output := setup(t)

	execute("whatever", []string{})

	assertOutputContains(t, output, "Basic Commands:")
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

// reproduces #117
func TestStartNextWithBranchContainingHyphen(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "test-branch"

	start()
	assertOnBranch(t, "mob/master-test-branch")
	assertMobSessionBranches(t, "mob/master-test-branch")

	next()
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
	configuration.MobStartIncludeUncommittedChanges = false
	createFile(t, "test.txt", "content")

	start()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	setup(t)
	configuration.MobStartIncludeUncommittedChanges = true
	createFile(t, "test.txt", "content")

	start()

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, "mob-session")
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	setup(t)
	configuration.MobStartIncludeUncommittedChanges = true
	createFile(t, "example.txt", "content")

	start()

	assertOnBranch(t, "mob-session")
}

func TestStartUntrackedFiles(t *testing.T) {
	setup(t)
	configuration.MobStartIncludeUncommittedChanges = false
	createFile(t, "example.txt", "content")

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

func TestStartDoneWithMobDoneSquashTrue(t *testing.T) {
	setup(t)
	configuration.MobDoneSquash = true

	start()
	assertOnBranch(t, "mob-session")

	done()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDoneWithMobDoneSquashFalse(t *testing.T) {
	setup(t)
	configuration.MobDoneSquash = false

	start()
	assertOnBranch(t, "mob-session")

	done()

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDonePublishingOneManualCommit(t *testing.T) {
	setup(t)
	// REFACTOR Replace string with enum value
	configuration.MobDoneSquash = false // default is probably true

	start()
	assertOnBranch(t, "mob-session")
	// should be 1 commit on mob-session so far

	createFileAndCommitIt(t, "example.txt", "content", "[manual-commit-1] publish this commit to master")
	assertCommits(t, 2)

	done() // without squash (configuration)

	assertOnBranch(t, "master")
	assertCommitsOnBranch(t, 2, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDoneSquashTheOneManualCommit(t *testing.T) {
	setup(t)
	// REFACTOR Replace string with enum value
	configuration.MobDoneSquash = true

	start()
	assertOnBranch(t, "mob-session")
	// should be 1 commit on mob-session so far

	createFileAndCommitIt(t, "example.txt", "content", "[manual-commit-1] publish this commit to master")
	assertCommits(t, 2)

	done()

	// MAYBE assertUnstagedChanges()
	assertOnBranch(t, "master")
	assertCommitsOnBranch(t, 1, "master")
	assertCommitsOnBranch(t, 1, "origin/master")
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
	createFileAndCommitIt(t, "example.txt", "content", "asdf")
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

	createFileAndCommitIt(t, "example.txt", "content", "asdf")
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
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
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
	createFileAndCommitIt(t, "example2.txt", "asdf", "asdf")
	git("push")

	setWorkingDir("/tmp/mob/local")
	start()
	done()
	assertOutputContains(t, output, "   git commit")
}

func setup(t *testing.T) *string {
	configuration = getDefaultConfiguration()
	configuration.MobNextStay = false
	output := captureOutput()
	createTestbed(t)
	assertOnBranch(t, "master")
	equals(t, []string{"master"}, gitBranches())
	equals(t, []string{"origin/master"}, gitRemoteBranches())
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
	assertCommitsOnBranch(t, commits, "HEAD")
}

func assertCommitsOnBranch(t *testing.T, commits int, branchName string) {
	result := silentgit("rev-list", "--count", branchName)
	number, _ := strconv.Atoi(strings.TrimSpace(result))
	if number != commits {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, strconv.Itoa(commits)+" commits in "+workingDir, strconv.Itoa(number)+" commits in "+workingDir)
		t.FailNow()
	}
}

func assertCommitLogContainsMessage(t *testing.T, branchName string, commitMessage string) {
	logMessages := silentgit("log", branchName, "--oneline")
	if !strings.Contains(logMessages, commitMessage) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, "git log contains '"+commitMessage+"'", logMessages)
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

func createFileAndCommitIt(t *testing.T, filename string, content string, commitMessage string) {
	createFile(t, filename, content)
	git("add", filename)
	git("commit", "-m", commitMessage)
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
