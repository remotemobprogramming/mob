package main

import (
	"fmt"
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/say"
	"github.com/remotemobprogramming/mob/v4/test"
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

var (
	tempDir string
)

type GitStatus = map[string]string
type TestBedOptions struct {
	enablePushOptions bool
}

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

func TestNextNotMobProgramming(t *testing.T) {
	output, configuration := setup(t)

	next(configuration)

	assertOutputContains(t, output, "to start working together")
}

func TestRequireCommitMessage(t *testing.T) {
	output, _ := setup(t)
	configuration := config.GetDefaultConfiguration()
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
	assertOutputContains(t, output, "wip branch 'mob-session' contains 6 commits.")
}

func TestExecuteKicksOffStatus(t *testing.T) {
	output, _ := setup(t)

	execute("status", []string{}, config.GetDefaultConfiguration())

	assertOutputContains(t, output, "you are on base branch 'master'")
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

func TestReset(t *testing.T) {
	output, configuration := setup(t)

	reset(configuration)

	assertOutputContains(t, output, "mob reset --delete-remote-wip-branch")
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

	reset(configuration)

	assertOutputContains(t, output, "mob reset --delete-remote-wip-branch")
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

	reset(configuration)

	assertOutputContains(t, output, "mob reset --delete-remote-wip-branch")
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
	configuration.StartIncludeUncommittedChanges = false
	createFile(t, "test.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	_, configuration := setup(t)
	configuration.StartIncludeUncommittedChanges = true
	createFile(t, "test.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartIncludeUnstagedChangesInNewWorkingDirectory(t *testing.T) {
	output, configuration := setup(t)
	configuration.StartIncludeUncommittedChanges = true
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
	configuration.StartIncludeUncommittedChanges = true
	createFile(t, "example.txt", "contentIrrelevant")

	start(configuration)

	assertOnBranch(t, "mob-session")
}

func TestStartUntrackedFiles(t *testing.T) {
	_, configuration := setup(t)
	configuration.StartIncludeUncommittedChanges = false
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
}

func TestStartCreateIncludeUncommitedChangesOnUnpushedFeatureBranchWithUncommitedChanges(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	createFile(t, "file.txt", "contentIrrelevant")

	configuration.StartCreate = true
	configuration.StartIncludeUncommittedChanges = true
	start(configuration)

	assertOnBranch(t, "mob/feature1")
}

func TestStartCreateIncludeUncommitedChangesOnUnpushedFeatureBranchWithUncommitedChangesAndBranchPostfix(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	createFile(t, "file.txt", "contentIrrelevant")

	configuration.StartCreate = true
	configuration.StartIncludeUncommittedChanges = true
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

	assertOutputNotContains(t, output, "git push --no-verify --set-upstream origin mob-session")
	assertOutputContains(t, output, "git push --push-option ci.skip --no-verify --set-upstream origin mob-session")
}

func TestStartPushOnWIPBranchWithOptionsShouldFailAndRetry(t *testing.T) {
	output, configuration := setupWithOptions(t, TestBedOptions{enablePushOptions: false})

	start(configuration)

	assertOutputContains(t, output, "git push --push-option ci.skip --no-verify --set-upstream origin mob-session")
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
	file1Path := filepath.Join(workingDir, "file1.txt")
	err := os.Remove(file1Path)
	if err != nil {
		failWithFailure(t, "no error", fmt.Sprintf("error %v occured deleting file %s", err, file1Path))
	}
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
	oldPath := filepath.Join(workingDir, "file1.txt")
	newPath := filepath.Join(workingDir, "dir", "file1.txt")
	err := os.Rename(oldPath, newPath)
	if err != nil {
		failWithFailure(t, "no error", fmt.Sprintf("error %v occured moving %s to %s", err, oldPath, newPath))
	}
	next(configuration)

	assertOnBranch(t, "mob-session")
	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage)
}

func TestStartNextStay_OpenLastModifiedFile(t *testing.T) {
	_, configuration := setup(t)
	configuration.NextStay = true
	configuration.OpenCommand = "touch %s-1"

	start(configuration)
	createFile(t, "file.txt", "contentIrrelevant")
	assertOnBranch(t, "mob-session")
	next(configuration)

	start(configuration)

	assertGitStatus(t, GitStatus{
		"file.txt-1": "??",
	})
}

func TestRunOutput(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	outputFile := filepath.Join(tempDir, "local", "file1.txt")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		failWithFailure(t, "no error", fmt.Sprintf("error %v occured reading %s", err, outputFile))
	}
	output := string(content)
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
	// should be 1 commit on mob-session so far

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
	// should be 1 commit on mob-session so far

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

	start(configuration) // should be 1 commit on mob-session so far
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

	start(configuration) // should be 1 commit on mob-session so far
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

func TestGitRootDir(t *testing.T) {
	setup(t)
	expectedPath, _ := filepath.EvalSymlinks(tempDir + "/local")
	equals(t, expectedPath, gitRootDir())
}

func TestGitRootDirWithSymbolicLink(t *testing.T) {
	setup(t)
	symlinkDir := tempDir + "/local-symlink"
	setWorkingDir(symlinkDir)
	expectedLocalSymlinkPath, _ := filepath.EvalSymlinks(symlinkDir)
	equals(t, expectedLocalSymlinkPath, gitRootDir())
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

func TestAbortTimerIfNewTimerIsStarted(t *testing.T) {
	_, configuration := setup(t)
	startTimer("10", configuration)
	assertSingleTimerProcess(t)

	startTimer("10", configuration)

	assertSingleTimerProcess(t)
	abortRunningTimers()
}

func assertSingleTimerProcess(t *testing.T) {
	test.Await(t, func() bool { return 1 == len(findMobTimerProcessIds()) }, "exactly 1 mob timer process found")
}

func assertNoTimerProcess(t *testing.T) {
	test.Await(t, func() bool { return 0 == len(findMobTimerProcessIds()) }, "no mob timer process found")
}

func TestAbortBreakTimerIfNewBreakTimerIsStarted(t *testing.T) {
	_, configuration := setup(t)
	startBreakTimer("10", configuration)
	assertSingleTimerProcess(t)

	startBreakTimer("10", configuration)

	assertSingleTimerProcess(t)
	abortRunningTimers()
}

func TestAbortTimerIfMobNext(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	startTimer("10", configuration)
	assertSingleTimerProcess(t)

	next(configuration)

	assertNoTimerProcess(t)
}

func TestAbortTimerIfMobDone(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	startTimer("10", configuration)
	assertSingleTimerProcess(t)

	done(configuration)

	assertNoTimerProcess(t)
}

func gitStatus() GitStatus {
	shortStatus := silentgit("status", "--short")
	statusLines := strings.Split(shortStatus, "\n")
	var statusMap = make(GitStatus)
	for _, line := range statusLines {
		if len(line) == 0 {
			continue
		}
		file := strings.Fields(line)
		statusMap[file[1]] = file[0]
	}
	return statusMap
}

func setupWithOptions(t *testing.T, options TestBedOptions) (output *string, configuration config.Configuration) {
	configuration = config.GetDefaultConfiguration()
	configuration.NextStay = false
	output = captureOutput(t)
	createTestbed(t, configuration, options)
	assertOnBranch(t, "master")
	equals(t, []string{"master"}, gitBranches())
	equals(t, []string{"origin/master"}, gitRemoteBranches())
	assertNoMobSessionBranches(t, configuration, "mob-session")
	abortRunningTimers()
	return output, configuration
}

func setup(t *testing.T) (output *string, configuration config.Configuration) {
	return setupWithOptions(t, TestBedOptions{
		enablePushOptions: true,
	})
}

func captureOutput(t *testing.T) *string {
	messages := ""
	say.PrintToConsole = func(text string) {
		t.Log(strings.TrimRight(text, "\n"))
		messages += text
	}
	return &messages
}

func createTestbed(t *testing.T, configuration config.Configuration, options TestBedOptions) {
	workingDir = ""

	tempDir = t.TempDir()

	say.Say("Creating testbed in temporary directory " + tempDir)
	createTestbedIn(t, tempDir, options)

	setWorkingDir(tempDir + "/local")
	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func createTestbedIn(t *testing.T, temporaryDirectory string, options TestBedOptions) {
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
	createRemoteRepository(remoteDirectory, options)

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
	err := ioutil.WriteFile(pathToFile, contentAsBytes, 0644)
	if err != nil {
		failWithFailure(t, "creating file "+filename+" with content "+content, "error")
	}
	return
}

func createDirectory(t *testing.T, directory string) (pathToFile string) {
	return createDirectoryInPath(t, workingDir, directory)
}

func createDirectoryInPath(t *testing.T, path, directory string) (pathToFile string) {
	pathToFile = path + "/" + directory
	err := os.Mkdir(pathToFile, 0755)
	if err != nil {
		failWithFailure(t, "creating directory "+pathToFile, "error")
	}
	return
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

func assertMobSessionBranches(t *testing.T, configuration config.Configuration, branch string) {
	if !newBranch(branch).hasRemoteBranch(configuration) {
		failWithFailure(t, newBranch(branch).remote(configuration).Name, "none")
	}
	if !hasLocalBranch(branch) {
		failWithFailure(t, branch, "none")
	}
}

func assertLocalBranch(t *testing.T, branch string) {
	if !hasLocalBranch(branch) {
		failWithFailure(t, branch, "none")
	}
}

func assertNoLocalBranch(t *testing.T, branch string) {
	if hasLocalBranch(branch) {
		failWithFailure(t, branch, "none")
	}
}

func assertNoMobSessionBranches(t *testing.T, configuration config.Configuration, branch string) {
	if newBranch(branch).hasRemoteBranch(configuration) {
		failWithFailure(t, "none", newBranch(branch).remote(configuration).Name)
	}
	if hasLocalBranch(branch) {
		failWithFailure(t, "none", branch)
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

func createRemoteRepository(path string, options TestBedOptions) {
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
	git("config", "receive.advertisePushOptions", strconv.FormatBool(options.enablePushOptions))
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
		say.Error("Could not create smylink from " + gitDirectory + " to " + path + "/.git")
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
