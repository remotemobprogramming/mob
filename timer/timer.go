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
	StartTimer(minutes int, configuration config.Configuration) error
	StartBreakTimer(minutes int, configuration config.Configuration) error
}

// GetTimers returns the list of active timers based on room and configuration.
// Both WebTimer and ProcessLocalTimer can be active simultaneously.
func GetTimers(room string, timerUser string, configuration config.Configuration) []Timer {
	var timers []Timer
	if room != "" {
		timers = append(timers, WebTimer{Room: room, TimerUser: timerUser})
	}
	if configuration.TimerLocal {
		timers = append(timers, ProcessLocalTimer{})
	}
	return timers
}

// RunTimer parses timerInMinutes, starts all configured timers and returns any error.
func RunTimer(timerInMinutes string, room string, timerUser string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	timers := GetTimers(room, timerUser, configuration)
	if len(timers) == 0 {
		say.Error("No timer configured, not starting timer")
		exit.Exit(1)
	}

	for _, t := range timers {
		if err := t.StartTimer(timeoutInMinutes, configuration); err != nil {
			say.Error(err.Error())
			exit.Exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
	return nil
}

// RunBreakTimer parses timerInMinutes, starts all configured break timers and returns any error.
func RunBreakTimer(timerInMinutes string, room string, timerUser string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting break timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	timers := GetTimers(room, timerUser, configuration)
	if len(timers) == 0 {
		say.Error("No break timer configured, not starting break timer")
		exit.Exit(1)
	}

	for _, t := range timers {
		if err := t.StartBreakTimer(timeoutInMinutes, configuration); err != nil {
			say.Error(err.Error())
			exit.Exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min break timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". So take a break now! :)")
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
