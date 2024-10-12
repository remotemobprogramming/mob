package main

import (
	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/say"
)

func status(configuration config.Configuration) {
	if isMobProgramming(configuration) {
		currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		say.Info("you are on wip branch " + currentWipBranch.String() + " (base branch " + currentBaseBranch.String() + ")")

		sayLastCommitsList(currentBaseBranch, currentWipBranch, configuration)
	} else {
		currentBaseBranch, _ := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		say.Info("you are on base branch '" + currentBaseBranch.String() + "'")
		showActiveMobSessions(configuration, currentBaseBranch)
	}
}

func showActiveMobSessions(configuration config.Configuration, currentBaseBranch Branch) {
	existingWipBranches := getWipBranchesForBaseBranch(currentBaseBranch, configuration)
	if len(existingWipBranches) > 0 {
		say.Info("remote wip branches detected:")
		for _, wipBranch := range existingWipBranches {
			time := silentgit("log", "-1", "--pretty=format:(%ar)", wipBranch)
			say.WithPrefix(wipBranch+" "+time, "  - ")
		}
	} else {
		say.Info("no remote wip branches detected!")
	}
}
