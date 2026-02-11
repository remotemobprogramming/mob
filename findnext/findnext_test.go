package findnext

import (
	"testing"

	mobtest "github.com/remotemobprogramming/mob/v5/test"
)

func TestFindNextTypistNoCommits(t *testing.T) {
	lastCommitters := []string{}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "")
	mobtest.Equals(t, history, []string(nil))
}

func TestFindNextTypistOnFirstCommit(t *testing.T) {
	lastCommitters := []string{"alice"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "")
	mobtest.Equals(t, history, []string(nil))
}

func TestFindNextTypistStartingWithFirstCommitterTwice(t *testing.T) {
	lastCommitters := []string{"alice", "alice"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "")
	mobtest.Equals(t, history, []string(nil))
}

func TestFindNextTypistOnlyCurrentCommitterInList(t *testing.T) {
	lastCommitters := []string{"alice", "alice", "alice"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "")
	mobtest.Equals(t, history, []string(nil))
}

func TestFindNextTypistCurrentCommitterAlternatingWithOneOtherPerson(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "alice", "bob", "alice"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "bob")
	mobtest.Equals(t, history, []string{"bob", "alice"})
}

func TestFindNextTypistCommitterFirstSeenInFirstRound(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "craig")
	mobtest.Equals(t, history, []string(nil))
}

func TestFindNextTypistSecondCommitterFirstSeenRunningSession(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "bob"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "craig")
	mobtest.Equals(t, history, []string(nil))
}

func TestFindNextTypistCurrentCommitterCommittedBefore(t *testing.T) {
	lastCommitters := []string{"alice", "alice", "bob", "alice"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "bob")
	mobtest.Equals(t, history, []string{"bob", "alice"})
}

func TestFindNextTypistThreeCommitters(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "craig")
	mobtest.Equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistIgnoreMultipleCommitsFromSamePerson(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "craig", "alice"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "craig")
	mobtest.Equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistSuggestCommitterBeforeLastCommit(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice", "bob", "dan"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "dan")
	mobtest.Equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistSuggestCommitterBeforeLastCommitInThreshold(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice", "bob", "dan", "erik", "fin"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "erik")
	mobtest.Equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistIgnoreCommitterBeforeLastCommitOutsideThreshold(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice", "craig", "bob", "alice", "fin"}

	nextTypist, history := FindNextTypist(lastCommitters, "alice")

	mobtest.Equals(t, nextTypist, "craig")
	mobtest.Equals(t, history, []string{"craig", "bob", "alice"})
}
