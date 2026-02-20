package main

import (
	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	timerpkg "github.com/remotemobprogramming/mob/v5/timer"
	_ "github.com/remotemobprogramming/mob/v5/timer/localtimer"
	_ "github.com/remotemobprogramming/mob/v5/timer/webtimer"
)

func StartTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
	}
}

func startTimer(timerInMinutes string, configuration config.Configuration) error {
	configuration.TimerRoom = getMobTimerRoom(configuration)
	return timerpkg.RunTimer(timerInMinutes, configuration)
}

func StartBreakTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startBreakTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
	}
}

func startBreakTimer(timerInMinutes string, configuration config.Configuration) error {
	configuration.TimerRoom = getMobTimerRoom(configuration)
	return timerpkg.RunBreakTimer(timerInMinutes, configuration)
}

func getMobTimerRoom(configuration config.Configuration) string {
	if !isGit() {
		return configuration.TimerRoom
	}

	currentWipBranchQualifier := configuration.WipBranchQualifier
	if currentWipBranchQualifier == "" {
		currentBranch := gitCurrentBranch()
		currentBaseBranch, _ := determineBranches(currentBranch, gitBranches(), configuration)

		if currentBranch.IsWipBranch(configuration) {
			wipBranchWithoutWipPrefix := currentBranch.removeWipPrefix(configuration).Name
			currentWipBranchQualifier = removePrefix(removePrefix(wipBranchWithoutWipPrefix, currentBaseBranch.Name), configuration.WipBranchQualifierSeparator)
		}
	}

	if configuration.TimerRoomUseWipBranchQualifier && currentWipBranchQualifier != "" {
		return currentWipBranchQualifier
	}

	return configuration.TimerRoom
}
