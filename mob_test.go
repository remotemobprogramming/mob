package main

import (
	"bufio"
	"fmt"
	"github.com/remotemobprogramming/mob/v5/ask"
	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/open"
	"github.com/remotemobprogramming/mob/v5/say"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

var (
	tempDir              string
	originalExitFunction func(int)
)

type GitStatus = map[string]string

func TestCurrentCliName(t *testing.T) {
	equals(t, "mob", currentCliName("mob"))
	equals(t, "mob", currentCliName("mob.exe"))
	equals(t, "mob", currentCliName("./mob"))
	equals(t, "mob", currentCliName("folder/mob"))
	// Check with platform specific path separators as well
	equals(t, "mob", currentCliName(filepath.Join("folder", "another", "mob.exe")))
	equals(t, "other_name", currentCliName(filepath.Join("folder", "another", "other_name")))
}

func TestDetermineBranches(t *testing.T) {
	configuration := config.GetDefaultConfiguration()
	configuration.WipBranchQualifierSeparator = "-"

	assertDetermineBranches(t, "master", "", []string{}, "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "", []string{}, "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "green", []string{}, "master", "mob-session")

	assertDetermineBranches(t, "master", "green", []string{}, "master", "mob/master-green")
	assertDetermineBranches(t, "mob/master-green", "", []string{}, "master", "mob/master-green")

	assertDetermineBranches(t, "master", "test-branch", []string{}, "master", "mob/master-test-branch")
	assertDetermineBranches(t, "mob/master-test-branch", "", []string{}, "master", "mob/master-test-branch")

	assertDetermineBranches(t, "feature1", "", []string{}, "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1", "", []string{}, "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1-green", "", []string{}, "feature1", "mob/feature1-green")
	assertDetermineBranches(t, "feature1", "green", []string{}, "feature1", "mob/feature1-green")

	assertDetermineBranches(t, "feature/test", "", []string{"feature/test"}, "feature/test", "mob/feature/test")
	assertDetermineBranches(t, "mob/feature/test", "", []string{"feature/test", "mob/feature/test"}, "feature/test", "mob/feature/test")

	assertDetermineBranches(t, "feature/test-ch", "", []string{"DPL-2638-update-apis", "DPL-2814-create-project", "feature/test-ch", "fix/smallChanges", "master", "pipeship/pipelineupdate-pipeship-pipeline.yaml"}, "feature/test-ch", "mob/feature/test-ch")
}

func assertDetermineBranches(t *testing.T, branch string, qualifier string, branches []string, expectedBase string, expectedWip string) {
	configuration := config.GetDefaultConfiguration()
	configuration.WipBranchQualifier = qualifier
	baseBranch, wipBranch := determineBranches(newBranch(branch), branches, configuration)
	equals(t, newBranch(expectedBase), baseBranch)
	equals(t, newBranch(expectedWip), wipBranch)
}

func TestRemoveWipPrefix(t *testing.T) {
	configuration := config.GetDefaultConfiguration()
	configuration.WipBranchPrefix = "mob/"
	equals(t, "master-green", newBranch("mob/master-green").removeWipPrefix(configuration).Name)
	equals(t, "master-green-blue", newBranch("mob/master-green-blue").removeWipPrefix(configuration).Name)
	equals(t, "main-branch", newBranch("mob/main-branch").removeWipPrefix(configuration).Name)
}

func TestRemoveWipBranchQualifier(t *testing.T) {
	var configuration config.Configuration

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "green"
	equals(t, "master", newBranch("master-green").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "test-branch"
	equals(t, "master", newBranch("master-test-branch").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branch"
	equals(t, "master-test", newBranch("master-test-branch").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branch"
	equals(t, "master-test", newBranch("master-test-branch").removeWipQualifier([]string{"master-test"}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "/-/"
	configuration.WipBranchQualifier = "branch-qualifier"
	equals(t, "main", newBranch("main/-/branch-qualifier").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branchqualifier"
	equals(t, "main/branchqualifier", newBranch("main/branchqualifier").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = ""
	configuration.WipBranchQualifier = "branchqualifier"
	equals(t, "main", newBranch("mainbranchqualifier").removeWipQualifier([]string{}, configuration).Name)
}

func TestRemoveWipBranchQualifierWithoutBranchQualifierSet(t *testing.T) {
	var configuration config.Configuration

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = ""
	equals(t, "main", newBranch("main").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = ""
	equals(t, "master", newBranch("master-test-branch").removeWipQualifier([]string{}, configuration).Name)
}

func TestVersion(t *testing.T) {
	output, _ := setup(t)

	version()

	assertOutputContains(t, output, versionNumber)
}

func TestHasCommits(t *testing.T) {
	_, _ = setup(t)

	commits := hasCommits()

	equals(t, true, commits)
}

func TestHasCommits_NoCommits(t *testing.T) {
	tempDir = t.TempDir()
	setWorkingDir(tempDir)
	git("init")

	commits := hasCommits()

	equals(t, false, commits)
}

func TestNextNotMobProgramming(t *testing.T) {
	output, configuration := setup(t)

	next(configuration)

	assertOutputContains(t, output, "to start working together")
}

func TestRequireCommitMessage(t *testing.T) {
	output, configuration := setup(t)
	configuration.NextStay = true
	configuration.RequireCommitMessage = true
	start(configuration)

	next(configuration)
	// ensure we don't complain if there's nothing to commit
	// https://github.com/remotemobprogramming/mob/pull/107#issuecomment-761298861
	assertOutputContains(t, output, "nothing to commit")

	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)
	// failure message should make sense regardless of whether we
	// provided commit message via `-m` or MOB_WIP_COMMIT_MESSAGE
	// https://github.com/remotemobprogramming/mob/pull/107#issuecomment-761591039
	assertOutputContains(t, output, "commit message required")
}

func TestDoneNotMobProgramming(t *testing.T) {
	output, configuration := setup(t)

	done(configuration)

	assertOutputContains(t, output, "to start working together")
}

func TestExecuteInvalidCommandKicksOffHelp(t *testing.T) {
	output, _ := setup(t)

	execute("whatever", []string{}, config.GetDefaultConfiguration())

	assertOutputContains(t, output, "Basic Commands:")
}

func TestExecuteAnyCommandWithHelpArgumentShowsHelpOutput(t *testing.T) {
	output, _ := setup(t)

	execute("s", []string{"10", "--help"}, config.GetDefaultConfiguration())
	assertOutputContains(t, output, "Basic Commands:")

	execute("next", []string{"help"}, config.GetDefaultConfiguration())
	assertOutputContains(t, output, "Basic Commands:")
}

func TestStart(t *testing.T) {
	_, configuration := setup(t)

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDespiteGitHook(t *testing.T) {
	_, configuration := setup(t)
	createExecutableFileInPath(t, workingDir+"/.git/hooks", "pre-commit", "#!/bin/sh\necho 'boo'\nexit 1\n")

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartWithCISkip(t *testing.T) {
	output, configuration := setup(t)
	configuration.SkipCiPushOptionEnabled = true
	mockExit()

	start(configuration)

	assertOutputContains(t, output, "git push --push-option ci.skip --no-verify --set-upstream origin mob-session:mob-session")
	assertOutputContains(t, output, "Disable the push option ci.skip in your .mob file or set the expected environment variable")
	assertOutputContains(t, output, "export MOB_SKIP_CI_PUSH_OPTION_ENABLED=false")
	resetExit()
}

func TestStartWithOutCISkip(t *testing.T) {
	output, configuration := setup(t)
	configuration.SkipCiPushOptionEnabled = false

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
	assertCommitLogNotContainsMessage(t, "mob-session", configuration.StartCommitMessage)
	assertOutputNotContains(t, output, "--push-option ci.skip")

}

func TestStartWithMultipleExistingBranches(t *testing.T) {
	output, configuration := setup(t)

	configuration.WipBranchQualifier = "green"
	start(configuration)
	assertOnBranch(t, "mob/master-green")
	next(configuration)
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = ""
	start(configuration)
	assertOnBranch(t, "mob-session")
	assertOutputContains(t, output, "preexisting wip branches have been detected")
	assertOutputContains(t, output, "mob/master-green")
}

func TestStartWithMultipleExistingBranchesAndEmptyWipBranchQualifier(t *testing.T) {
	output, configuration := setup(t)

	configuration.WipBranchQualifier = "green"
	start(configuration)
	next(configuration)

	configuration.WipBranchQualifier = ""
	start(configuration)
	assertOnBranch(t, "mob-session")
	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartWithMultipleExistingBranchesWithStay(t *testing.T) {
	output, configuration := setup(t)
	configuration.NextStay = true

	configuration.WipBranchQualifier = "green"
	assertOnBranch(t, "master")
	start(configuration)
	assertOnBranch(t, "mob/master-green")
	next(configuration)
	assertOnBranch(t, "mob/master-green")

	configuration.WipBranchQualifier = ""
	start(configuration)
	assertOnBranch(t, "mob/master-green")
	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartNextWithBranch(t *testing.T) {
	_, configuration := setup(t)
	configuration.ResetDeleteRemoteWipBranch = true
	assertOnBranch(t, "master")
	configuration.WipBranchQualifier = "green"

	start(configuration)
	assertOnBranch(t, "mob/master-green")
	assertMobSessionBranches(t, configuration, "mob/master-green")
	configuration.WipBranchQualifier = ""

	next(configuration)
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = "green"
	reset(configuration)
	assertNoMobSessionBranches(t, configuration, "mob/master-green")
}

func TestStartNextStartWithBranch(t *testing.T) {
	_, configuration := setup(t)
	configuration.WipBranchQualifier = "green"
	configuration.NextStay = true
	assertOnBranch(t, "master")

	start(configuration)
	assertOnBranch(t, "mob/master-green")

	next(configuration)
	assertOnBranch(t, "mob/master-green")

	start(configuration)
	assertOnBranch(t, "mob/master-green")
}

func TestStartWarnsAboutPreexistingWipBranches(t *testing.T) {
	output, configuration := setup(t)
	checkoutAndPushBranch("feature-something")
	checkoutAndPushBranch("feature-something-2")

	assertOnBranch(t, "feature-something-2")
	start(configuration)
	assertOnBranch(t, "mob/feature-something-2")
	next(configuration)

	git("checkout", "feature-something")
	start(configuration)
	assertOnBranch(t, "mob/feature-something")
	assertOutputContains(t, output, "preexisting wip branches have been detected")
	assertOutputContains(t, output, "mob/feature-something-2")
}

func TestStartWarnsOnDivergingWipBranch(t *testing.T) {
	output, configuration := setup(t)

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	next(configuration)

	git("checkout", "master")
	createFileAndCommitIt(t, "example.txt", "other", "other")
	git("push")

	start(configuration)

	assertOutputContains(t, output, "Careful, your wip branch (mob-session) diverges from your main branch (origin/master) !")
}

func TestStartJoinDoesNotWarn(t *testing.T) {
	output, configuration := setup(t)

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)

	assertOutputNotContains(t, output, "Careful, your wip branch (mob-session) diverges from your main branch (origin/master) !")
}

func TestStartNextOnFeatureWithBranch(t *testing.T) {
	_, configuration := setup(t)
	configuration.WipBranchQualifier = "green"
	checkoutAndPushBranch("feature1")
	assertOnBranch(t, "feature1")

	start(configuration)
	assertOnBranch(t, "mob/feature1-green")

	next(configuration)
	assertOnBranch(t, "feature1")
}

// reproduces #117
func TestStartNextWithBranchContainingHyphen(t *testing.T) {
	_, configuration := setup(t)
	configuration.WipBranchQualifier = "test-branch"
	start(configuration)
	assertOnBranch(t, "mob/master-test-branch")
	assertMobSessionBranches(t, configuration, "mob/master-test-branch")

	configuration.WipBranchQualifier = ""
	next(configuration)
}

func TestStartWithPushDefaultTracking(t *testing.T) {
	_, configuration := setup(t)
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	git("push", "origin", "master")
	git("config", "push.default", "tracking")

	start(configuration)
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartWithJoiningNonExistingSession(t *testing.T) {
	_, configuration := setup(t)
	assertOnBranch(t, "master")
	configuration.StartJoin = true
	start(configuration)
	assertOnBranch(t, "master")
}

func TestReset(t *testing.T) {
	output, configuration := setup(t)

	reset(configuration)

	assertOutputContains(t, output, "Executing this command deletes the mob branch for everyone. Are you sure you want to continue? (Y/n)")
}

func TestResetDeleteRemoteWipBranch(t *testing.T) {
	_, configuration := setup(t)
	configuration.ResetDeleteRemoteWipBranch = true

	reset(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestResetCommit(t *testing.T) {
	output, configuration := setup(t)
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)
	assertMobSessionBranches(t, configuration, "mob-session")
	simulateUserInput("y")

	reset(configuration)

	assertOutputContains(t, output, "Executing this command deletes the mob branch for everyone. Are you sure you want to continue? (Y/n)")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestResetDeleteRemoteWipBranchCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.ResetDeleteRemoteWipBranch = true
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)
	assertMobSessionBranches(t, configuration, "mob-session")

	reset(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestResetCommitBranch(t *testing.T) {
	output, configuration := setup(t)
	configuration.WipBranchQualifier = "green"
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)
	assertMobSessionBranches(t, configuration, "mob/master-green")
	simulateUserInput("n")

	reset(configuration)

	assertOutputContains(t, output, "Executing this command deletes the mob branch for everyone. Are you sure you want to continue? (Y/n)")
	assertOutputContains(t, output, "Aborted")
	assertMobSessionBranches(t, configuration, "mob/master-green")
}

func TestResetDeleteRemoteWipBranchCommitBranch(t *testing.T) {
	_, configuration := setup(t)
	configuration.WipBranchQualifier = "green"
	configuration.ResetDeleteRemoteWipBranch = true
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)
	assertMobSessionBranches(t, configuration, "mob/master-green")

	reset(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob/master-green")
}

func TestClean(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "mob-session")

	clean(configuration)

	assertOnBranch(t, "master")
	assertNoLocalBranch(t, "mob-session")
}

func TestCleanAfterStart(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)

	clean(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestCleanNotFullyMergedMissingRemoteBranch(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)

	createFile(t, "example.txt", "contentIrrelevant")

	next(configuration)

	git("push", "origin", "mob-session", "--delete")

	clean(configuration)

	assertOnBranch(t, "master")
	assertNoLocalBranch(t, "mob-session")
}

func TestCleanFeatureOrphanWipBranch(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	git("checkout", "-b", "mob/feature1")

	clean(configuration)

	assertOnBranch(t, "feature1")
	assertNoLocalBranch(t, "mob/feature1")
}

func TestCleanMissingBaseBranch(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "mob/feature1")

	clean(configuration)

	assertOnBranch(t, "master")
	assertNoLocalBranch(t, "mob/feature1")
}

func TestStartUnstagedChanges(t *testing.T) {
	output, configuration := setup(t)
	configuration.HandleUncommittedChanges = config.FailWithError
	createFile(t, "test.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
	assertOutputContains(t, output, "mob start --discard-uncommitted-changes")
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	_, configuration := setup(t)
	configuration.HandleUncommittedChanges = config.IncludeChanges
	createFile(t, "test.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDiscardUnstagedChanges(t *testing.T) {
	_, configuration := setup(t)
	configuration.HandleUncommittedChanges = config.DiscardChanges
	createFile(t, "test.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
	assertCleanGitStatus(t)
}

func TestStartIncludeUnstagedChangesInNewWorkingDirectory(t *testing.T) {
	output, configuration := setup(t)
	configuration.HandleUncommittedChanges = config.IncludeChanges
	createDirectory(t, "subdirnew")
	setWorkingDir(tempDir + "/local/subdirnew")
	createFile(t, "test.txt", "contentIrrelevant")
	assertFileExist(t, tempDir+"/local/subdirnew/test.txt")

	start(configuration)

	assertOutputContains(t, output, "cannot start; current working dir is an uncommitted subdir")
}

func TestStartHasUnpushedCommits(t *testing.T) {
	output, configuration := setup(t)
	createFileAndCommitIt(t, "test.txt", "contentIrrelevant", "unpushed change")

	start(configuration)

	assertOutputContains(t, output, "cannot start; unpushed changes")
	assertOutputContains(t, output, "unpushed commits")
}

func TestBranch(t *testing.T) {
	output, configuration := setup(t)
	start(configuration)

	branch(configuration)

	assertOutputContains(t, output, "\norigin/mob-session\n")
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	_, configuration := setup(t)
	configuration.HandleUncommittedChanges = config.IncludeChanges
	createFile(t, "example.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "mob-session")
}

func TestStartUntrackedFiles(t *testing.T) {
	_, configuration := setup(t)
	configuration.HandleUncommittedChanges = config.FailWithError
	createFile(t, "example.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "master")
}

func TestStartOnUnpushedFeatureBranch(t *testing.T) {
	output, configuration := setup(t)
	git("checkout", "-b", "feature1")

	start(configuration)

	assertOnBranch(t, "feature1")
	assertOutputContains(t, output, "Remote branch origin/feature1 is missing")
	assertOutputContains(t, output, "mob start --create")
}

func TestStartOnUnpushedFeatureBranchWithUncommitedChanges(t *testing.T) {
	output, configuration := setup(t)
	git("checkout", "-b", "feature1")
	createFile(t, "file.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "feature1")
	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
	assertOutputContains(t, output, "mob start --discard-uncommitted-changes")
}

func TestStartCreateOnUnpushedFeatureBranch(t *testing.T) {
	output, configuration := setup(t)
	git("checkout", "-b", "feature1")

	configuration.StartCreate = true
	start(configuration)

	assertOutputNotContains(t, output, "Remote branch origin/feature1 already exists")
	assertOnBranch(t, "mob/feature1")
}

func TestStartCreateOnUnpushedFeatureBranchWithBranchPostfix(t *testing.T) {
	output, configuration := setup(t)
	git("checkout", "-b", "feature1")

	configuration.StartCreate = true
	configuration.WipBranchQualifier = "green"
	start(configuration)

	assertOutputNotContains(t, output, "Remote branch origin/feature1 already exists")
	assertOnBranch(t, "mob/feature1-green")
}

func TestStartCreateOnUnpushedFeatureBranchWithUncommitedChanges(t *testing.T) {
	output, configuration := setup(t)
	git("checkout", "-b", "feature1")
	createFile(t, "file.txt", "contentIrrelevant")

	configuration.StartCreate = true
	start(configuration)

	assertOutputContains(t, output, "To start, including uncommitted changes and create the remote branch, use")
	assertOutputContains(t, output, "mob start --create --include-uncommitted-changes")
	assertOutputContains(t, output, "mob start --create --discard-uncommitted-changes")
}

func TestStartCreateIncludeUncommitedChangesOnUnpushedFeatureBranchWithUncommitedChanges(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	createFile(t, "file.txt", "contentIrrelevant")

	configuration.StartCreate = true
	configuration.HandleUncommittedChanges = config.IncludeChanges
	start(configuration)

	assertOnBranch(t, "mob/feature1")
}

func TestStartCreateIncludeUncommitedChangesOnUnpushedFeatureBranchWithUncommitedChangesAndBranchPostfix(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	createFile(t, "file.txt", "contentIrrelevant")

	configuration.StartCreate = true
	configuration.HandleUncommittedChanges = config.IncludeChanges
	configuration.WipBranchQualifier = "green"
	start(configuration)

	assertOnBranch(t, "mob/feature1-green")
}

func TestStartCreateOnPushedFeatureBranch(t *testing.T) {
	output, configuration := setup(t)
	checkoutAndPushBranch("feature1")

	configuration.StartCreate = true
	start(configuration)

	assertOutputContains(t, output, "Remote branch origin/feature1 already exists")
	assertOnBranch(t, "mob/feature1")
}

func TestStartCreateOnPushedFeatureBranchWithBranchPostfix(t *testing.T) {
	output, configuration := setup(t)
	checkoutAndPushBranch("feature1")

	configuration.StartCreate = true
	configuration.WipBranchQualifier = "green"
	start(configuration)

	assertOutputContains(t, output, "Remote branch origin/feature1 already exists")
	assertOnBranch(t, "mob/feature1-green")
}

func TestStartCreateOnPushedFeatureBranchWhichIsAhead(t *testing.T) {
	_, configuration := setup(t)
	checkoutAndPushBranch("feature1")
	createFile(t, "file.txt", "contentIrrelevant")
	git("add", ".")
	git("commit", "-m", "commit ahead")
	git("push")
	git("reset", "--hard", "HEAD~1")

	configuration.StartCreate = true
	start(configuration)

	assertOnBranch(t, "mob/feature1")
	assertCommitLogContainsMessage(t, "mob/feature1", "commit ahead")
}

func TestStartCreateOnPushedFeatureBranchWhichIsBehind(t *testing.T) {
	output, configuration := setup(t)
	checkoutAndPushBranch("feature1")
	createFile(t, "file.txt", "contentIrrelevant")
	git("add", ".")
	git("commit", "-m", "commit ahead")

	configuration.StartCreate = true
	start(configuration)

	assertOnBranch(t, "feature1")
	assertOutputContains(t, output, "ERROR cannot start; unpushed changes on base branch must be pushed upstream")
	assertOutputContains(t, output, "git push origin feature1")
}

func TestStartPushOnWIPBranchWithOptions(t *testing.T) {
	output, configuration := setup(t)

	start(configuration)

	assertOutputContains(t, output, "git push --no-verify --set-upstream origin mob-session")
}

func TestStartPushOnWIPBranchWithOptionsShouldFailAndRetry(t *testing.T) {
	output, configuration := setup(t)

	start(configuration)

	assertOutputContains(t, output, "git push --no-verify --set-upstream origin mob-session")
	assertOutputContains(t, output, "you are on wip branch 'mob-session' (base branch 'master')")
}

func TestStartNextBackToMaster(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	assertOnBranch(t, "mob-session")

	next(configuration)

	assertOnBranch(t, "master")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartNextStay(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	assertOnBranch(t, "mob-session")

	next(configuration)

	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage+"\n\nlastFile:file1.txt")
	assertOnBranch(t, "mob-session")
}

func TestStartNextStay_WriteLastModifiedFileInCommit_WhenFileIsAdded(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true

	start(configuration)
	createFile(t, "olderFile.txt", "contentIrrelevant")
	createFile(t, "newerFile.txt", "contentIrrelevant")
	next(configuration)

	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage+"\n\nlastFile:newerFile.txt")
}

func TestStartNextStay_WriteLastModifiedFileInCommit_WhenFileIsModified(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	createFile(t, "file2.txt", "contentIrrelevant")
	next(configuration)

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevantButModified")
	next(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage+"\n\nlastFile:file1.txt")
}

func TestStartNextStay_WriteLastModifiedFileInCommit_WhenFileIsModifiedAndWorkingDirIsNotProjectRoot(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	createFile(t, "file2.txt", "contentIrrelevant")
	next(configuration)

	start(configuration)
	createDirectory(t, "dir")
	createFile(t, "file1.txt", "contentIrrelevantButModified")
	setWorkingDir(workingDir + "/dir")
	next(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage+"\n\nlastFile:file1.txt")
}

func TestStartNextStay_DoNotWriteLastModifiedFileInCommit_WhenFileIsDeleted(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	createFile(t, "file2.txt", "contentIrrelevant")
	next(configuration)

	start(configuration)
	removeFile(t, filepath.Join(workingDir, "file1.txt"))
	next(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage)
}

func TestStartNextStay_DoNotWriteLastModifiedFileInCommit_WhenFileIsMoved(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	next(configuration)

	start(configuration)
	createDirectory(t, "dir")
	moveFile(t, filepath.Join(workingDir, "file1.txt"), filepath.Join(workingDir, "dir", "file1.txt"))
	next(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage)
}

func TestStartNextStay_OpenLastModifiedFile(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	if runtime.GOOS == "windows" {
		configuration.OpenCommand = "cmd.exe /C type nul > %s-1"
	} else {
		configuration.OpenCommand = "touch %s-1"
	}

	start(configuration)
	createFile(t, "file.txt", "contentIrrelevant")
	assertOnBranch(t, "mob-session")
	next(configuration)

	start(configuration)

	assertGitStatus(t, GitStatus{
		"file.txt-1": "??",
	})
}

func TestStartNextStay_OpenLastModifiedFile_WhenLastModifiedFilePathContainsSpaces(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	if runtime.GOOS == "windows" {
		configuration.OpenCommand = "cmd.exe /C type nul > %s-1"
	} else {
		configuration.OpenCommand = "touch %s-1"
	}

	start(configuration)
	createFile(t, "file with spaces.txt", "contentIrrelevant")
	assertOnBranch(t, "mob-session")
	next(configuration)

	start(configuration)

	assertGitStatus(t, GitStatus{
		"file with spaces.txt-1": "??",
	})
}

func TestRunOutput(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	output := readFile(t, filepath.Join(tempDir, "local", "file1.txt"))
	assertOutputContains(t, &output, "asdf")
}

func TestTestbed(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	createFile(t, "file2.txt", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file3.txt", "owqe")
	next(configuration)

	setWorkingDir(tempDir + "/bob")
	start(configuration)
	createFile(t, "file4.txt", "zcvx")
	next(configuration)

	setWorkingDir(tempDir + "/local-symlink")
	start(configuration)
	createFile(t, "file5.txt", "uiop")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)

	output := silentgit("log", "--pretty=format:'%ae'")
	assertOutputContains(t, &output, "local")
	assertOutputContains(t, &output, "localother")
	assertOutputContains(t, &output, "alice")
	assertOutputContains(t, &output, "bob")
	assertOutputContains(t, &output, "local-symlink")
}

func TestStartDoneWithMobDoneSquash(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.Squash

	start(configuration)
	assertOnBranch(t, "mob-session")

	done(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneWithMobDoneSquashWithOldEmptyMobStartCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.Squash

	start(configuration)
	assertOnBranch(t, "mob-session")
	assertCommitsOnBranch(t, 1, "mob-session")

	git("commit", gitHooksOption(configuration), "--allow-empty", "-m", configuration.StartCommitMessage)
	git("push")
	assertCommitsOnBranch(t, 2, "mob-session")

	createFile(t, "test1.txt", "contentIrrelevant")
	next(configuration)
	assertCommitsOnBranch(t, 3, "mob-session")

	start(configuration)
	createFile(t, "test2.txt", "contentIrrelevant")

	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"test1.txt": "A",
		"test2.txt": "A",
	})
	assertCommitsOnBranch(t, 1, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneWithMobDoneNoSquashWithOldEmptyMobStartCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.NoSquash

	start(configuration)
	assertOnBranch(t, "mob-session")
	assertCommitsOnBranch(t, 1, "mob-session")

	git("commit", gitHooksOption(configuration), "--allow-empty", "-m", configuration.StartCommitMessage)
	git("push")
	assertCommitsOnBranch(t, 2, "mob-session")

	createFile(t, "test1.txt", "contentIrrelevant")
	next(configuration)
	assertCommitsOnBranch(t, 3, "mob-session")

	start(configuration)
	createFile(t, "test2.txt", "contentIrrelevant")

	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"test2.txt": "A",
	})
	assertCommitsOnBranch(t, 3, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneWithMobDoneSquashWipWithOldEmptyMobStartCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.SquashWip

	start(configuration)
	assertOnBranch(t, "mob-session")
	assertCommitsOnBranch(t, 1, "mob-session")

	git("commit", gitHooksOption(configuration), "--allow-empty", "-m", configuration.StartCommitMessage)
	git("push")
	assertCommitsOnBranch(t, 2, "mob-session")

	manualCommit(t, configuration, "test1.txt", "test1")
	assertCommitsOnBranch(t, 3, "mob-session")

	start(configuration)
	createFile(t, "test2.txt", "contentIrrelevant")
	next(configuration)
	assertCommitsOnBranch(t, 4, "mob-session")

	start(configuration)
	createFile(t, "test3.txt", "contentIrrelevant")

	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"test2.txt": "A",
		"test3.txt": "A",
	})
	assertCommitsOnBranch(t, 2, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneWithMobDoneBugMergeTwice(t *testing.T) {
	output, configuration := setup(t)

	start(configuration)
	assertOnBranch(t, "mob-session")

	done(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
	assertOutputNotContains(t, output, "git merge --squash --ff mob-session\n  git merge --squash --ff mob-session\n")
}

func TestStartDoneSquashWithUnpushedCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.Squash

	// now in /local
	createFileAndCommitIt(t, "file1.txt", "owqe", "not a mob session yet")

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file2.txt", "zcvx")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	git("push")

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	done(configuration)

	assertFileExist(t, "file1.txt")
}

func TestStartDoneSquashWipWithUnpushedCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.SquashWip

	// now in /local
	createFileAndCommitIt(t, "file1.txt", "owqe", "not a mob session yet")

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file2.txt", "zcvx")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	git("push")

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	done(configuration)

	assertFileExist(t, "file1.txt")
}

func TestStartDoneWithMobDoneNoSquash(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.NoSquash

	start(configuration)
	assertOnBranch(t, "mob-session")

	done(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDonePublishingOneManualCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.NoSquash

	start(configuration)
	assertOnBranch(t, "mob-session")
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "[manual-commit-1] publish this commit to master")
	assertCommits(t, 2)

	done(configuration) // without squash (configuration)

	assertOnBranch(t, "master")
	assertCleanGitStatus(t)
	assertCommitsOnBranch(t, 2, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashTheOneManualCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.Squash

	start(configuration)
	assertOnBranch(t, "mob-session")
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "[manual-commit-1] publish this commit to master")
	assertCommits(t, 2)

	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"example.txt": "A",
	})
	assertCommitsOnBranch(t, 1, "master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneWithUncommittedChanges(t *testing.T) {
	_, configuration := setup(t)

	start(configuration) // should be 1 commit on mob-session so far
	createFile(t, "example.txt", "contentIrrelevant")

	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"example.txt": "A",
	})
	assertCommitsOnBranch(t, 1, "master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneNoSquashWithUncommittedChanges(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.NoSquash

	start(configuration)
	createFile(t, "example.txt", "content")

	done(configuration) // without squash (configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"example.txt": "A",
	})
	assertCommitsOnBranch(t, 1, "master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashWipPublishingOneManualCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.SquashWip

	start(configuration)
	createFile(t, "some.txt", "contentIrrelevant")
	next(configuration) // this wip commit will be squashed

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "[manual-commit-1] publish this commit to master")

	done(configuration)

	assertOnBranch(t, "master")
	assertCleanGitStatus(t)
	assertCommitsOnBranch(t, 2, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashWipWithUncommittedChanges(t *testing.T) {
	_, configuration := setup(t)

	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")

	configuration.DoneSquash = config.SquashWip
	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"example.txt": "A",
	})
	assertCommitsOnBranch(t, 1, "master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashWipPublishingOneManualCommitHasUncommittedModifications(t *testing.T) {
	_, configuration := setup(t)
	configuration.DoneSquash = config.SquashWip

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "[manual-commit-1] publish this commit to master")

	createFile(t, "example.txt", "contentIrrelevant2") // modify previously committed file
	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"example.txt": "M",
	})
	assertCommitsOnBranch(t, 2, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashWipOneWipCommitAfterManualCommit(t *testing.T) {
	_, configuration := setup(t)

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "[manual-commit-1] publish this commit to master")
	next(configuration)

	start(configuration)
	createFile(t, "file.txt", "contentIrrelevant") // the user should see these changes staged after done
	next(configuration)

	start(configuration)
	configuration.DoneSquash = config.SquashWip
	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"file.txt": "A",
	})
	assertCommitsOnBranch(t, 2, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashWipManyWipCommitsAfterManualCommit(t *testing.T) {
	_, configuration := setup(t)

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "[manual-commit-1] publish this commit to master")
	next(configuration)

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant") // the user should see these changes staged after done
	next(configuration)

	start(configuration)
	createFile(t, "file2.txt", "contentIrrelevant") // the user should see these changes staged after done
	next(configuration)

	start(configuration)
	configuration.DoneSquash = config.SquashWip
	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"file1.txt": "A",
		"file2.txt": "A",
	})
	assertCommitsOnBranch(t, 2, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashWipOnlyWipCommits(t *testing.T) {
	_, configuration := setup(t)

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant") // the user should see these changes staged after done
	next(configuration)

	start(configuration)
	createFile(t, "file2.txt", "contentIrrelevant") // the user should see these changes staged after done
	next(configuration)

	start(configuration)
	configuration.DoneSquash = config.SquashWip
	done(configuration)

	assertOnBranch(t, "master")
	assertGitStatus(t, GitStatus{
		"file1.txt": "A",
		"file2.txt": "A",
	})
	assertCommitsOnBranch(t, 1, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashWipOnlyManualCommits(t *testing.T) {
	_, configuration := setup(t)

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "[manual-commit-1] publish this commit to master")
	next(configuration)

	start(configuration)
	createFileAndCommitIt(t, "example2.txt", "contentIrrelevant", "[manual-commit-2] publish this commit to master")
	next(configuration)

	start(configuration)
	configuration.DoneSquash = config.SquashWip
	done(configuration)

	assertOnBranch(t, "master")
	assertCleanGitStatus(t)
	assertCommitsOnBranch(t, 3, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-2] publish this commit to master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestDoneSquashWipWithoutStartDoesNotLooseChanges(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFileAndCommitIt(t, "file1.txt", "owqe", "not a mob session yet")
	configuration.NextStay = true
	next(configuration)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file2.txt", "zcvx")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	assertOnBranch(t, "mob-session")
	configuration.DoneSquash = config.SquashWip
	done(configuration)

	assertOnBranch(t, "master")
	assertFileExist(t, "file2.txt")
}

func TestStartDoneFeatureBranch(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	done(configuration)

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartNextFeatureBranch(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	next(configuration)

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneFeatureBranchWithDash(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feat/load_test_DLC-253")
	git("push", "origin", "feat/load_test_DLC-253", "--set-upstream")
	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	done(configuration)

	assertNoMobSessionBranches(t, configuration, "mob/feat/load_test_DLC-253")
}

func TestGitRootDir(t *testing.T) {
	setup(t)
	expectedPath, _ := filepath.EvalSymlinks(tempDir + "/local")
	equals(t, expectedPath, filepath.FromSlash(gitRootDir()))
}

func TestGitRootDirWithSymbolicLink(t *testing.T) {
	setup(t)
	symlinkDir := tempDir + "/local-symlink"
	setWorkingDir(symlinkDir)
	expectedLocalSymlinkPath, _ := filepath.EvalSymlinks(symlinkDir)
	equals(t, expectedLocalSymlinkPath, filepath.FromSlash(gitRootDir()))
}

func TestBothCreateNonemptyCommitWithNext(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	createFile(t, "file2.txt", "contentIrrelevant")

	setWorkingDir(tempDir + "/local")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	// next(configuration) not possible, would fail
	git("pull")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")
}

func TestNothingToCommitCreatesNoCommits(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir(tempDir + "/local")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertCommits(t, 1)
}

func TestStartNextPushManualCommits(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertFileExist(t, "example.txt")
}

func TestStartBranchWithUncommitedChangesFixWithBranch(t *testing.T) {
	output, _ := setup(t)
	mockExit()

	setWorkingDir(tempDir + "/local")

	createFile(t, "uncommited.txt", "contentIrrelevant")
	runMob(t, tempDir+"/local", "start", "-b", "green")

	assertOutputContains(t, output, "mob start --branch green --include-uncommitted-changes")
	assertOutputContains(t, output, "mob start --branch green --discard-uncommitted-changes")
	resetExit()
}

func TestStartBranchEnvWithUncommitedChangesFixWithoutBranch(t *testing.T) {
	output, _ := setup(t)
	mockExit()

	setWorkingDir(tempDir + "/local")
	t.Setenv("MOB_WIP_BRANCH_QUALIFIER", "red")
	createFile(t, "uncommited.txt", "contentIrrelevant")
	runMob(t, tempDir+"/local", "start")

	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
	assertOutputContains(t, output, "mob start --discard-uncommitted-changes")
	resetExit()
}

func TestStartCreateBranchWithUncommitedChangesFixWithBranch(t *testing.T) {
	output, _ := setup(t)
	mockExit()

	setWorkingDir(tempDir + "/local")

	git("checkout", "-b", "unpushedBranch")
	createFile(t, "uncommited.txt", "contentIrrelevant")
	runMob(t, tempDir+"/local", "start", "--create", "-b", "green")

	assertOutputContains(t, output, "mob start --create --branch green --include-uncommitted-changes")
	assertOutputContains(t, output, "mob start --create --branch green --discard-uncommitted-changes")
	resetExit()
}

func TestStartCreateBranchEnvWithUncommitedChangesFixWithoutBranch(t *testing.T) {
	output, _ := setup(t)
	mockExit()

	setWorkingDir(tempDir + "/local")
	os.Setenv("MOB_WIP_BRANCH_QUALIFIER", "red")
	git("checkout", "-b", "unpushedBranch")
	createFile(t, "uncommited.txt", "contentIrrelevant")
	runMob(t, tempDir+"/local", "start", "--create")

	os.Unsetenv("MOB_WIP_BRANCH_QUALIFIER")
	assertOutputContains(t, output, "mob start --create --include-uncommitted-changes")
	assertOutputContains(t, output, "mob start --create --discard-uncommitted-changes")
	resetExit()
}

func TestStartNextPushManualCommitsFeatureBranch(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")

	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	createFileAndCommitIt(t, "example.txt", "contentIrrelevant", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	git("fetch")
	git("checkout", "feature1")
	start(configuration)
	assertFileExist(t, "example.txt")
}

func TestConflictingMobSessions(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	git("commit", "-m", "\"finished mob session\"")

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example2.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
}

func TestConflictingMobSessionsNextStay(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	git("commit", "-m", "\"finished mob session\"")

	setWorkingDir(tempDir + "/localother")
	start(configuration)
}

func TestDoneMergeConflict(t *testing.T) {
	output, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	git("push")

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	assertOutputContains(t, output, "To fix this, solve the merge conflict manually, commit, push, and afterwards delete mob-session")
}

func TestDoneMerge(t *testing.T) {
	output, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	createFileAndCommitIt(t, "example2.txt", "contentIrrelevant", "asdf")
	git("push")

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	assertOutputContains(t, output, "  git commit")
}

func TestDoneSquashNoChanges(t *testing.T) {
	output, configuration := setup(t)
	setWorkingDir(tempDir + "/local")
	checkoutAndPushBranch("feature-something")

	start(configuration)
	done(configuration)

	assertOutputContains(t, output, "nothing to commit")
}

func TestDoneSquashWipStartCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	configuration.DoneSquash = config.SquashWip

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	next(configuration)
	assertCommitsOnBranch(t, 2, "mob-session")
	done(configuration)
	assertCommitsOnBranch(t, 1, "master")
}

func TestDonePullsIfAlreadyDone(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true

	setWorkingDir(tempDir + "/bob")
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	done(configuration)
	git("commit", "-am", "\"mob done by Alice\"")
	git("push")

	setWorkingDir(tempDir + "/bob")
	done(configuration)

	assertFileExist(t, "example.txt")
}

func TestDoneNoSquashStartCommit(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	configuration.DoneSquash = config.NoSquash

	start(configuration)
	createFile(t, "file1.txt", "contentIrrelevant")
	next(configuration)
	assertCommitsOnBranch(t, 2, "mob-session")
	done(configuration)
	assertCommitsOnBranch(t, 2, "master")
}

func TestStartAndNextInSubdir(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local/subdir")
	start(configuration)
	createFile(t, "example.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/localother/subdir")
	start(configuration)
	createFile(t, "example2.txt", "contentIrrelevant")
	createFile(t, "../example3.txt", "contentIrrelevant")
	next(configuration)

	setWorkingDir(tempDir + "/local/subdir")
	start(configuration)
	done(configuration)

	setWorkingDir(tempDir + "/local")
	assertFileExist(t, "subdir/example.txt")
	assertFileExist(t, "subdir/example2.txt")
	assertFileExist(t, "example3.txt")
}

func TestIsGitIdentifiesGitRepo(t *testing.T) {
	setup(t)
	equals(t, true, isGit())
}

func TestIsGitIdentifiesOutsideOfGitRepo(t *testing.T) {
	setWorkingDir(tempDir + "/notgit")
	equals(t, false, isGit())
}

func TestEmptyGitStatus(t *testing.T) {
	setup(t)

	status := gitStatus()

	equals(t, 0, len(status))
	assertCleanGitStatus(t)
}

func TestGitStatusWithOneFile(t *testing.T) {
	setup(t)
	createFile(t, "hello.txt", "contentIrrelevant")

	status := gitStatus()

	equals(t, GitStatus{
		"hello.txt": "??",
	}, status)
}

func TestGitStatusWithManyFiles(t *testing.T) {
	setup(t)
	createFile(t, "hello.txt", "contentIrrelevant")
	createFile(t, "added.txt", "contentIrrelevant")
	git("add", "added.txt")

	status := gitStatus()

	equals(t, GitStatus{
		"added.txt": "A",
		"hello.txt": "??",
	}, status)
}

func TestBranchesDoNotDiverge(t *testing.T) {
	setup(t)
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	git("checkout", "-b", "diverges")

	diverge := doBranchesDiverge("master", "diverges")

	equals(t, false, diverge)
}

func TestBranchesDoDiverge(t *testing.T) {
	setup(t)
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	git("checkout", "-b", "diverges")
	createFileAndCommitIt(t, "example.txt", "other", "asdf")
	git("checkout", "master")
	createFileAndCommitIt(t, "diverging-commit.txt", "asdf", "diverging")

	diverge := doBranchesDiverge("master", "diverges")

	equals(t, true, diverge)
}

func TestHelpRequested(t *testing.T) {
	equals(t, false, helpRequested([]string{""}))
	equals(t, false, helpRequested([]string{"a", "mob", "21"}))
	equals(t, true, helpRequested([]string{"--help"}))
	equals(t, true, helpRequested([]string{"a", "help", "12"}))
	equals(t, true, helpRequested([]string{"s", "10", "-h"}))
}

func TestMobClean(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file1.txt", "abc")
	next(configuration)

	setWorkingDir(tempDir + "/bob")
	start(configuration)
	createFile(t, "file2.txt", "def")
	next(configuration)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file3.txt", "ghi")
	done(configuration)

	setWorkingDir(tempDir + "/bob")
	clean(configuration)

	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestMobStartOnWipBranchWithoutCheckedOutBaseBranchWithoutHyphens(t *testing.T) {
	output, configuration := setup(t)

	setWorkingDir(tempDir + "/alice")
	git("checkout", "-b", "basebranchwithouthyphen")
	configuration.StartCreate = true
	start(configuration)
	assertOnBranch(t, "mob/basebranchwithouthyphen")
	createFile(t, "file1.txt", "abc")
	next(configuration)
	assertOnBranch(t, "basebranchwithouthyphen")

	setWorkingDir(tempDir + "/bob")
	git("checkout", "-b", "mob/basebranchwithouthyphen")
	configuration.StartCreate = false

	assertNoError(t, start(configuration))
	assertOnBranch(t, "mob/basebranchwithouthyphen")
	assertOutputContains(t, output, "joining existing session from origin/mob/basebranchwithouthyphen")

	createFile(t, "file2.txt", "abc")
	done(configuration)
	assertOnBranch(t, "basebranchwithouthyphen")
}

func TestGitVersionParse(t *testing.T) {
	// Check real examples
	equals(t, GitVersion{2, 34, 1}, parseGitVersion("git version 2.34.1"))
	equals(t, GitVersion{2, 38, 1}, parseGitVersion("git version 2.38.1.windows.1"))
	// Check missing prefix
	equals(t, GitVersion{1, 2, 3}, parseGitVersion("git 1.2.3"))
	equals(t, GitVersion{4, 5, 6}, parseGitVersion("4.5.6"))
	// Check missing minor and patch
	equals(t, GitVersion{2, 5, 0}, parseGitVersion("git version 2.5"))
	equals(t, GitVersion{2, 0, 0}, parseGitVersion("git version 2"))
	equals(t, GitVersion{4, 0, 0}, parseGitVersion("4"))
	// Invalid versions
	equals(t, GitVersion{0, 0, 0}, parseGitVersion("not version"))
	equals(t, GitVersion{2, 0, 0}, parseGitVersion("2.xyz3.5"))
	equals(t, GitVersion{2, 0, 0}, parseGitVersion("2.9999999999999999999999.5"))
}

func TestGitVersionCompare(t *testing.T) {
	// Check real examples
	equals(t, true, (&GitVersion{2, 12, 0}).Less(GitVersion{2, 13, 0}))
	equals(t, false, (&GitVersion{2, 13, 0}).Less(GitVersion{2, 13, 0}))
	equals(t, false, (&GitVersion{2, 14, 0}).Less(GitVersion{2, 13, 0}))
	// Test each part of the version number
	equals(t, true, (&GitVersion{1, 2, 3}).Less(GitVersion{5, 2, 3}))
	equals(t, true, (&GitVersion{1, 2, 3}).Less(GitVersion{1, 5, 3}))
	equals(t, true, (&GitVersion{1, 2, 3}).Less(GitVersion{1, 2, 5}))
}

func TestMobConfigWorksOutsideOfGitRepository(t *testing.T) {
	output := captureOutput(t)
	runMob(t, t.TempDir(), "config")

	assertOutputNotContains(t, output, "ERROR")
	assertOutputContains(t, output, "MOB_CLI_NAME=\"mob\"")
}

func TestMobHelpWorksOutsideOfGitRepository(t *testing.T) {
	output := captureOutput(t)
	runMob(t, t.TempDir(), "help")

	assertOutputNotContains(t, output, "ERROR")
	assertOutputContains(t, output, "Basic Commands:")
}

func TestMobShowsHelpIfCommandIsUnknownAndOutsideOfGitRepository(t *testing.T) {
	output := captureOutput(t)
	runMob(t, t.TempDir(), "unknown")

	assertOutputNotContains(t, output, "ERROR")
	assertOutputContains(t, output, "Basic Commands:")
}

func TestMobMooWorksOutsideOfGitRepository(t *testing.T) {
	output := captureOutput(t)
	runMob(t, t.TempDir(), "help")

	assertOutputNotContains(t, output, "ERROR")
	assertOutputContains(t, output, "moo")
}

func TestMobVersionWorksOutsideOfGitRepository(t *testing.T) {
	output := captureOutput(t)
	runMob(t, t.TempDir(), "version")

	assertOutputNotContains(t, output, "ERROR")
	assertOutputContains(t, output, "v"+versionNumber)
}

func runMob(t *testing.T, workingDir string, args ...string) {
	setWorkingDir(workingDir)
	newArgs := append([]string{"mob"}, args...)
	run(newArgs)
}

func gitStatus() GitStatus {
	shortStatus := silentgit("status", "--porcelain")
	statusLines := strings.Split(shortStatus, "\n")
	var statusMap = make(GitStatus)
	for _, line := range statusLines {
		if len(line) == 0 {
			continue
		}
		fields := strings.Fields(line)
		file := strings.Join(fields[1:], " ")
		if strings.HasPrefix(file, "\"") {
			file, _ = strconv.Unquote(file)
		}
		statusMap[file] = fields[0]
	}
	return statusMap
}

func setup(t *testing.T) (output *string, configuration config.Configuration) {
	configuration = config.GetDefaultConfiguration()
	// Test setup does not support push options
	configuration.SkipCiPushOptionEnabled = false
	configuration.NextStay = false
	createTestbed(t, configuration)
	assertOnBranch(t, "master")
	equals(t, []string{"master"}, gitBranches())
	equals(t, []string{"origin/master"}, gitRemoteBranches())
	assertNoMobSessionBranches(t, configuration, "mob-session")
	output = captureOutput(t)
	return output, configuration
}

func simulateUserInput(a string) {
	ask.ReadFromConsole = func(reader io.Reader) *bufio.Reader {
		return bufio.NewReader(strings.NewReader(a))
	}
}

func captureOutput(t *testing.T) *string {
	messages := ""
	say.PrintToConsole = func(text string) {
		t.Log(strings.TrimRight(text, "\n"))
		messages += text
	}
	return &messages
}

func mockOpenInBrowser() {
	open.OpenInBrowser = func(url string) error {
		fmt.Printf("call to mock OpenInBrowser with url: %s \n", url)
		return nil
	}
}

func mockExit() {
	originalExitFunction = Exit
	Exit = func(code int) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("exit(%d)\n", code)
			}
		}()

		panic(code)
	}
}

func resetExit() {
	Exit = originalExitFunction
}

func createTestbed(t *testing.T, configuration config.Configuration) {
	workingDir = ""

	tempDir = t.TempDir()

	say.Say("Creating testbed in temporary directory " + tempDir)
	createTestbedIn(t, tempDir)

	setWorkingDir(tempDir + "/local")
	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func createTestbedIn(t *testing.T, temporaryDirectory string) {
	say.Debug("Creating temporary test assets in " + temporaryDirectory)
	err := os.MkdirAll(temporaryDirectory, 0755)
	if err != nil {
		say.Error("Could not create temporary dir " + temporaryDirectory)
		say.Error(err.Error())
		return
	}
	say.Debug("Create remote repository")
	remoteDirectory := getRemoteDirectory(temporaryDirectory)
	cleanRepository(remoteDirectory)
	createRemoteRepository(remoteDirectory)

	say.Debug("Create first local repository")
	localDirectory := getLocalDirectory(temporaryDirectory)
	cleanRepository(localDirectory)
	cloneRepository(localDirectory, remoteDirectory)

	say.Debug("Populate, initial import and push")
	workingDir = localDirectory
	createFile(t, "test.txt", "test")
	createDirectory(t, "subdir")
	createFileInPath(t, localDirectory+"/subdir", "subdir.txt", "subdir")
	git("checkout", "-b", "master")
	git("add", ".")
	git("commit", "-m", "\"initial import\"")
	git("push", "--set-upstream", "--all", "origin")

	for _, name := range [3]string{"localother", "alice", "bob"} {
		cleanRepository(temporaryDirectory + "/" + name)
		cloneRepository(temporaryDirectory+"/"+name, remoteDirectory)
		say.Debug("Created local repository " + name)
	}

	notGitDirectory := getNotGitDirectory(temporaryDirectory)
	err = os.MkdirAll(notGitDirectory, 0755)
	if err != nil {
		say.Error("Count not create directory " + notGitDirectory)
		say.Error(err.Error())
		return
	}

	say.Debug("Creating local repository with .git symlink")
	symlinkDirectory := getSymlinkDirectory(temporaryDirectory)
	symlinkGitDirectory := getSymlinkGitDirectory(temporaryDirectory)
	cleanRepositoryWithSymlink(symlinkDirectory, symlinkGitDirectory)
	cloneRepositoryWithSymlink(symlinkDirectory, symlinkGitDirectory, remoteDirectory)
	say.Debug("Done.")
}

func setWorkingDir(dir string) {
	workingDir = dir
	say.Say("\n===== cd " + dir)
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		failWithFailure(t, nil, err)
	}

}

func assertError(t *testing.T, err error, errorMessage string) {
	if err == nil {
		failWithFailure(t, errorMessage, "No Error thrown")
	}
	if err.Error() != errorMessage {
		failWithFailure(t, errorMessage, err.Error())
	}
}

func assertCommits(t *testing.T, commits int) {
	assertCommitsOnBranch(t, commits, "HEAD")
}

func assertCommitsOnBranch(t *testing.T, commits int, branchName string) {
	result := silentgit("rev-list", "--count", branchName)
	number, _ := strconv.Atoi(result)
	if number != commits {
		failWithFailure(t, strconv.Itoa(commits)+" commits in "+workingDir, strconv.Itoa(number)+" commits in "+workingDir)
	}
}

func assertCommitLogContainsMessage(t *testing.T, branchName string, commitMessage string) {
	logMessages := silentgit("log", branchName, "--oneline")
	if !strings.Contains(logMessages, commitMessage) {
		failWithFailure(t, "git log contains '"+commitMessage+"'", logMessages)
	}
}

func assertCommitLogNotContainsMessage(t *testing.T, branchName string, commitMessage string) {
	logMessages := silentgit("log", branchName, "--oneline")
	if strings.Contains(logMessages, commitMessage) {
		failWithFailure(t, "git log does not contain '"+commitMessage+"'", logMessages)
	}
}

func assertFileExist(t *testing.T, filename string) {
	path := workingDir + "/" + filename
	if strings.Index(filename, "/") == 0 {
		path = filename
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		failWithFailure(t, "existing file "+path, "no file at "+path)
	}
}

func createFileAndCommitIt(t *testing.T, filename string, content string, commitMessage string) {
	createFile(t, filename, content)
	git("add", filename)
	git("commit", "-m", commitMessage)
}

func createFile(t *testing.T, filename string, content string) (pathToFile string) {
	return createFileInPath(t, workingDir, filename, content)
}

func createFileInPath(t *testing.T, path, filename, content string) (pathToFile string) {
	contentAsBytes := []byte(content)
	pathToFile = path + "/" + filename
	err := os.WriteFile(pathToFile, contentAsBytes, 0644)
	if err != nil {
		failWithFailure(t, "creating file "+filename+" with content "+content, "error")
	}
	return
}

func createExecutableFileInPath(t *testing.T, path, filename, content string) (pathToFile string) {
	ensureDirectoryExists(t, path)

	pathToFile = path + "/" + filename
	contentAsBytes := []byte(content)
	err := os.WriteFile(pathToFile, contentAsBytes, 0755)
	if err != nil {
		failWithFailure(t, "creating file "+filename+" with content "+content, "error")
	}
	return
}

func createDirectory(t *testing.T, directory string) (pathToDirectory string) {
	return ensureDirectoryExists(t, workingDir+"/"+directory)
}

func ensureDirectoryExists(t *testing.T, path string) (pathToDirectory string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		failWithFailure(t, "creating folder "+path, "error")
	}
	return path
}

func removeFile(t *testing.T, path string) {
	err := os.Remove(path)
	if err != nil {
		failWithFailure(t, "no error", fmt.Sprintf("error %v occured deleting file %s", err, path))
	}
}

func moveFile(t *testing.T, oldPath string, newPath string) {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		failWithFailure(t, "no error", fmt.Sprintf("error %v occured moving %s to %s", err, oldPath, newPath))
	}
}

func readFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		failWithFailure(t, "no error", fmt.Sprintf("reading file %s failed with %v", path, err))
	}
	output := string(content)
	return output
}

func assertOnBranch(t *testing.T, branch string) {
	currentBranch := gitCurrentBranch()
	if currentBranch.Name != branch {
		failWithFailure(t, "on branch "+branch, "on branch "+currentBranch.String())
	}
}

func assertOutputContains(t *testing.T, output *string, contains string) {
	currentOutput := *output
	if !strings.Contains(currentOutput, contains) {
		failWithFailure(t, "output contains '"+contains+"'", currentOutput)
	}
}

func assertOutputNotContains(t *testing.T, output *string, notContains string) {
	if strings.Contains(*output, notContains) {
		failWithFailure(t, "output not contains "+notContains, output)
	}
}

func assertMobSessionBranches(t *testing.T, configuration config.Configuration, branchName string) {
	branch := newBranch(branchName)
	if !branch.hasRemoteBranch(configuration) {
		failWithFailure(t, branch.remote(configuration).Name, "none")
	}
	if !branch.hasLocalBranch() {
		failWithFailure(t, branchName, "none")
	}
}

func assertLocalBranch(t *testing.T, branch string) {
	if !newBranch(branch).hasLocalBranch() {
		failWithFailure(t, branch, "none")
	}
}

func assertNoLocalBranch(t *testing.T, branch string) {
	if newBranch(branch).hasLocalBranch() {
		failWithFailure(t, branch, "none")
	}
}

func assertNoMobSessionBranches(t *testing.T, configuration config.Configuration, branchName string) {
	branch := newBranch(branchName)
	if branch.hasRemoteBranch(configuration) {
		failWithFailure(t, "none", branch.remote(configuration).Name)
	}
	if branch.hasLocalBranch() {
		failWithFailure(t, "none", branchName)
	}
}

func assertGitStatus(t *testing.T, expected map[string]string) {
	equals(t, expected, gitStatus())
}

func assertCleanGitStatus(t *testing.T) {
	status := gitStatus()
	if len(status) != 0 {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texpected a clean git status, but contained %s\"\n", filepath.Base(file), line, status)
		t.FailNow()
	}
}

func equals(t *testing.T, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		t.Log(string(debug.Stack()))
		failWithFailure(t, exp, act)
	}
}

func failWithFailure(t *testing.T, exp interface{}, act interface{}) {
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
	t.FailNow()
}

func checkoutAndPushBranch(branch string) {
	git("checkout", "-b", branch)
	git("push", "origin", branch, "--set-upstream")
}

func cleanRepository(path string) {
	say.Debug("cleanrepository: Delete " + path)
	err := os.RemoveAll(path)
	if err != nil {
		say.Error("Could not remove directory " + path)
		say.Error(err.Error())
		return
	}
}

func createRemoteRepository(path string) {
	branch := "master" // fixed to master for now
	say.Debug("createremoterepository: Creating remote repository " + path)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		say.Error("Could not create directory " + path)
		say.Error(err.Error())
		return
	}
	workingDir = path
	say.Debug("before git init")
	git("--bare", "init")
	say.Debug("before symbolic-ref")
	git("symbolic-ref", "HEAD", "refs/heads/"+branch)
	// see bug #346 changes the output of git status --short
	git("config", "--local", "status.branch", "true")
	say.Debug("finished")
}

func cloneRepository(path, remoteDirectory string) {
	say.Debug("clonerepository: Cloning remote " + remoteDirectory + " to " + path)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		say.Error("Could not create directory " + path)
		say.Error(err.Error())
		return
	}
	workingDir = path
	name := basename(path)
	git("clone", "--origin", "origin", "file://"+remoteDirectory, ".")
	git("config", "--local", "user.name", name)
	git("config", "--local", "user.email", name+"@example.com")
	// see bug #346 changes the output of git status --short
	git("config", "--local", "status.branch", "true")
}

func cloneRepositoryWithSymlink(path, gitDirectory, remoteDirectory string) {
	cloneRepository(path, remoteDirectory)
	say.Debug(fmt.Sprintf("clonerepositorywithsymlink: move .git to %s and create symlink to it", gitDirectory))
	err := os.Rename(filepath.FromSlash(path+"/.git"), gitDirectory)
	if err != nil {
		say.Error("Could not move directory " + path + " to " + gitDirectory)
		say.Error(err.Error())
		return
	}
	err = os.Symlink(gitDirectory, filepath.FromSlash(path+"/.git"))
	if err != nil {
		say.Error("Could not create symlink from " + gitDirectory + " to " + path + "/.git")
		say.Error(err.Error())
		return
	}
}

func cleanRepositoryWithSymlink(path, gitDirectory string) {
	cleanRepository(path)
	say.Debug("cleanrepositorywithsymlink: Delete " + gitDirectory)
	err := os.RemoveAll(gitDirectory)
	if err != nil {
		say.Error("Could not remove directory " + gitDirectory)
		say.Error(err.Error())
		return
	}
}

func basename(path string) string {
	split := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	return split[len(split)-1]
}

func getRemoteDirectory(path string) string {
	return path + "/remote"
}

func getLocalDirectory(path string) string {
	return path + "/local"
}

func getNotGitDirectory(path string) string {
	return path + "/notgit"
}

func getSymlinkGitDirectory(path string) string {
	return path + "/local-symlink.git"
}

func getSymlinkDirectory(path string) string {
	return path + "/local-symlink"
}
