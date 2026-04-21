package timer

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	"github.com/remotemobprogramming/mob/v5/say"
	"github.com/remotemobprogramming/mob/v5/timer/localtimer"
	"github.com/remotemobprogramming/mob/v5/timer/webtimer"
)

// Timer abstracts timer functionality so different implementations can be used.
type Timer interface {
	IsActive() bool
	StartTimer(minutes int) error
	StartBreakTimer(minutes int) error
}

func getActiveTimers(configuration config.Configuration) []Timer {
	all := []Timer{
		webtimer.NewWebTimer(configuration),
		localtimer.NewProcessLocalTimer(configuration),
	}
	var active []Timer
	for _, t := range all {
		if t.IsActive() {
			active = append(active, t)
		}
	}
	names := make([]string, len(active))
	for i, t := range active {
		names[i] = fmt.Sprintf("%T", t)
	}
	say.Debug(fmt.Sprintf("Active timers: %v", names))
	return active
}

// RunTimer parses timerInMinutes and starts the first active timer.
func RunTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting timer at %s for %d minutes (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timerInMinutes))

	timers := getActiveTimers(configuration)
	if len(timers) == 0 {
		say.Error("No timer configured, not starting timer")
		exit.Exit(1)
	}

	if err := timers[0].StartTimer(timeoutInMinutes); err != nil {
		say.Error(err.Error())
		exit.Exit(1)
	}

	say.Info(fmt.Sprintf("It's now %s. %d min timer ends at approx. %s. Happy collaborating! :)", currentTime(), timeoutInMinutes, timeOfTimeout))
	return nil
}

// RunBreakTimer parses timerInMinutes and starts the first active break timer.
func RunBreakTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting break timer at %s for %d minutes (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timerInMinutes))

	timers := getActiveTimers(configuration)
	if len(timers) == 0 {
		say.Error("No break timer configured, not starting break timer")
		exit.Exit(1)
	}

	if err := timers[0].StartBreakTimer(timeoutInMinutes); err != nil {
		say.Error(err.Error())
		exit.Exit(1)
	}

	say.Info(fmt.Sprintf("It's now %s. %d min break timer ends at approx. %s. So take a break now! :)", currentTime(), timeoutInMinutes, timeOfTimeout))
	return nil
}

func toMinutes(timerInMinutes string) (error, int) {
	timeoutInMinutes, err := strconv.Atoi(timerInMinutes)
	if err != nil || timeoutInMinutes < 1 {
		say.Error(fmt.Sprintf("The parameter must be an integer number greater then zero"))
		return errors.New("The parameter must be an integer number greater then zero"), 0
	}
	return nil, timeoutInMinutes
}

func currentTime() string {
	return time.Now().Format("15:04")
}
