package timer

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	"github.com/remotemobprogramming/mob/v5/say"
)

// Timer abstracts timer functionality so different implementations can be used.
type Timer interface {
	StartTimer(minutes int) error
	StartBreakTimer(minutes int) error
}

// Factory creates a Timer for the given configuration.
// Returns nil if the timer should not be active.
type Factory func(configuration config.Configuration) Timer

var factories []Factory

// Register adds a Timer factory to the registry.
// Implementation packages call this in their init() function.
func Register(f Factory) {
	factories = append(factories, f)
}

// GetTimers returns all registered timers that are active for the given configuration.
func GetTimers(configuration config.Configuration) []Timer {
	var timers []Timer
	for _, createTimer := range factories {
		t := createTimer(configuration)
		if t != nil {
			timers = append(timers, t)
		}
	}
	return timers
}

// RunTimer parses timerInMinutes, starts all active timers and returns any error.
func RunTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting timer at %s for %d minutes (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timerInMinutes))

	timers := GetTimers(configuration)
	if len(timers) == 0 {
		say.Error("No timer configured, not starting timer")
		exit.Exit(1)
	}

	for _, t := range timers {
		if err := t.StartTimer(timeoutInMinutes); err != nil {
			say.Error(err.Error())
			exit.Exit(1)
		}
	}

	say.Info(fmt.Sprintf("It's now %s. %d min timer ends at approx. %s. Happy collaborating! :)", currentTime(), timeoutInMinutes, timeOfTimeout))
	return nil
}

// RunBreakTimer parses timerInMinutes, starts all active break timers and returns any error.
func RunBreakTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting break timer at %s for %d minutes (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timerInMinutes))

	timers := GetTimers(configuration)
	if len(timers) == 0 {
		say.Error("No break timer configured, not starting break timer")
		exit.Exit(1)
	}

	for _, t := range timers {
		if err := t.StartBreakTimer(timeoutInMinutes); err != nil {
			say.Error(err.Error())
			exit.Exit(1)
		}
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
