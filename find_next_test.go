package main

import (
	"testing"
)

func TestFindNextTypistNoCommits(t *testing.T) {
	lastCommitters := []string{}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "")
	equals(t, history, []string(nil))
}

func TestFindNextTypistOnlyCurrentCommitterInList(t *testing.T) {
	lastCommitters := []string{"alice", "alice", "alice"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "alice")
	equals(t, history, []string{"alice"})
}

func TestFindNextTypistCurrentCommitterAlternatingWithOneOtherPerson(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "alice", "bob", "alice"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "bob")
	equals(t, history, []string{"bob", "alice"})
}

func TestFindNextTypistCurrentCommitterCommittedBefore(t *testing.T) {
	lastCommitters := []string{"alice", "alice", "bob", "alice"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "bob")
	equals(t, history, []string{"bob", "alice"})
}

func TestFindNextTypistThreeCommitters(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "craig")
	equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistIgnoreMultipleCommitsFromSamePerson(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "craig", "alice"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "craig")
	equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistSuggestCommitterBeforeLastCommit(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice", "bob", "dan"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "dan")
	equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistSuggestCommitterBeforeLastCommitInThreshold(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice", "bob", "dan", "erik", "fin"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "erik")
	equals(t, history, []string{"craig", "bob", "alice"})
}

func TestFindNextTypistIgnoreCommitterBeforeLastCommitOutsideThreshold(t *testing.T) {
	lastCommitters := []string{"alice", "bob", "craig", "alice", "craig", "bob", "alice", "fin"}

	nextTypist, history := findNextTypist(lastCommitters, "alice")

	equals(t, nextTypist, "craig")
	equals(t, history, []string{"craig", "bob", "alice"})
}
