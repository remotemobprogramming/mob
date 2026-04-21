package timer

import (
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/test"
)

type mockTimer struct {
	active                bool
	startTimerMinutes     int
	startBreakTimerMinutes int
}

func (m *mockTimer) IsActive() bool { return m.active }
func (m *mockTimer) StartTimer(minutes int) error {
	m.startTimerMinutes = minutes
	return nil
}
func (m *mockTimer) StartBreakTimer(minutes int) error {
	m.startBreakTimerMinutes = minutes
	return nil
}

func TestGetActiveTimerReturnsFirstActiveTimer(t *testing.T) {
	inactive := &mockTimer{active: false}
	active := &mockTimer{active: true}

	result := getActiveTimer([]Timer{inactive, active})

	test.Equals(t, active, result)
}

func TestGetActiveTimerReturnsNilWhenNoneActive(t *testing.T) {
	result := getActiveTimer([]Timer{&mockTimer{active: false}})

	test.Equals(t, nil, result)
}

func TestGetActiveTimerPrefersFirstOverSecond(t *testing.T) {
	first := &mockTimer{active: true}
	second := &mockTimer{active: true}

	result := getActiveTimer([]Timer{first, second})

	test.Equals(t, first, result)
}

func TestRunWithPassesMinutesToStartTimer(t *testing.T) {
	output := test.CaptureOutput(t)
	mock := &mockTimer{active: true}

	runWith([]Timer{mock}, "5")

	test.Equals(t, 5, mock.startTimerMinutes)
	test.AssertOutputContains(t, output, "Happy collaborating!")
}

func TestRunBreakWithPassesMinutesToStartBreakTimer(t *testing.T) {
	output := test.CaptureOutput(t)
	mock := &mockTimer{active: true}

	runBreakWith([]Timer{mock}, "10")

	test.Equals(t, 10, mock.startBreakTimerMinutes)
	test.AssertOutputContains(t, output, "So take a break now!")
}

func TestRunTimerReturnsErrorForZeroMinutes(t *testing.T) {
	output := test.CaptureOutput(t)

	err := RunTimer("0", config.GetDefaultConfiguration())

	test.NotEquals(t, nil, err)
	test.AssertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}

func TestRunTimerReturnsErrorForNonNumericInput(t *testing.T) {
	output := test.CaptureOutput(t)

	err := RunTimer("NotANumber", config.GetDefaultConfiguration())

	test.NotEquals(t, nil, err)
	test.AssertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}

func TestRunBreakTimerReturnsErrorForZeroMinutes(t *testing.T) {
	output := test.CaptureOutput(t)

	err := RunBreakTimer("0", config.GetDefaultConfiguration())

	test.NotEquals(t, nil, err)
	test.AssertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}

func TestRunBreakTimerReturnsErrorForNonNumericInput(t *testing.T) {
	output := test.CaptureOutput(t)

	err := RunBreakTimer("NotANumber", config.GetDefaultConfiguration())

	test.NotEquals(t, nil, err)
	test.AssertOutputContains(t, output, "The parameter must be an integer number greater then zero")
}
