package main

import (
	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	"github.com/remotemobprogramming/mob/v5/timer"
)

func StartTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
	}
}

func startTimer(timerInMinutes string, configuration config.Configuration) error {
	configuration = enrichConfigurationWithBranchQualifier(configuration)
	return timer.RunTimer(timerInMinutes, configuration)
}

func StartBreakTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startBreakTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
	}
}

func startBreakTimer(timerInMinutes string, configuration config.Configuration) error {
	configuration = enrichConfigurationWithBranchQualifier(configuration)
	return timer.RunBreakTimer(timerInMinutes, configuration)
}
