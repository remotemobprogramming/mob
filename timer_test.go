package main

import "testing"

func TestOpenTimerInBrowserWithTimerRoom(t *testing.T) {
	mockOpenInBrowser()
	output, configuration := setup(t)
	configuration.TimerRoom = "testroom"

	err := openTimerInBrowser(configuration)

	assertOutputNotContains(t, output, "Timer Room is not configured.")
	assertNoError(t, err)
}

func TestOpenTimerInBrowserWithoutTimerRoom(t *testing.T) {
	mockOpenInBrowser()
	output, configuration := setup(t)

	err := openTimerInBrowser(configuration)

	assertOutputContains(t, output, "Timer Room is not configured.")
	assertNoError(t, err)
}

func TestOpenTimerInBrowserError(t *testing.T) {
	mockOpenInBrowser()
	_, configuration := setup(t)
	configuration.TimerUrl = ""

	err := openTimerInBrowser(configuration)

	assertError(t, err, "Timer url is not configured")
}

func TestTimerNumberLessThen1(t *testing.T) {
	output, configuration := setup(t)

	err := startTimer("0", configuration)

	assertError(t, err, "The parameter must be an integer number greater then zero")
	assertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}

func TestTimerNotANumber(t *testing.T) {
	output, configuration := setup(t)

	err := startTimer("NotANumber", configuration)

	assertError(t, err, "The parameter must be an integer number greater then zero")
	assertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}

func TestTimer(t *testing.T) {
	output, configuration := setup(t)
	configuration.NotifyCommand = ""
	configuration.VoiceCommand = ""

	err := startTimer("1", configuration)

	assertNoError(t, err)
	assertOutputContains(t, output, "1 min timer ends at approx.")
	assertOutputContains(t, output, "Happy collaborating! :)")
}

func TestTimerExportFunction(t *testing.T) {
	output, configuration := setup(t)
	configuration.NotifyCommand = ""
	configuration.VoiceCommand = ""

	StartTimer("1", configuration)

	assertOutputContains(t, output, "1 min timer ends at approx.")
	assertOutputContains(t, output, "Happy collaborating! :)")
}

func TestBreakTimerNumberLessThen1(t *testing.T) {
	output, configuration := setup(t)

	err := startBreakTimer("0", configuration)

	assertError(t, err, "The parameter must be an integer number greater then zero")
	assertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}

func TestBreakTimerNotANumber(t *testing.T) {
	output, configuration := setup(t)

	err := startBreakTimer("NotANumber", configuration)

	assertError(t, err, "The parameter must be an integer number greater then zero")
	assertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}

func TestBreakTimer(t *testing.T) {
	output, configuration := setup(t)
	configuration.NotifyCommand = ""
	configuration.VoiceCommand = ""

	err := startBreakTimer("1", configuration)

	assertNoError(t, err)
	assertOutputContains(t, output, "1 min break timer ends at approx.")
	assertOutputContains(t, output, "So take a break now! :)")
}

func TestBreakTimerExportFunction(t *testing.T) {
	output, configuration := setup(t)
	configuration.NotifyCommand = ""
	configuration.VoiceCommand = ""

	StartBreakTimer("5", configuration)

	assertOutputContains(t, output, "5 min break timer ends at approx.")
	assertOutputContains(t, output, "So take a break now! :)")
}
