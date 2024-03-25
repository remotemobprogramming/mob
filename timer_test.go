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
