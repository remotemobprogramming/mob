package timer

import (
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
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

	if result != active {
		t.Error("expected the first active timer to be returned")
	}
}

func TestGetActiveTimerReturnsNilWhenNoneActive(t *testing.T) {
	result := getActiveTimer([]Timer{&mockTimer{active: false}})

	if result != nil {
		t.Error("expected nil when no timer is active")
	}
}

func TestGetActiveTimerPrefersFirstOverSecond(t *testing.T) {
	first := &mockTimer{active: true}
	second := &mockTimer{active: true}

	result := getActiveTimer([]Timer{first, second})

	if result != first {
		t.Error("expected the first active timer to take priority")
	}
}

func TestRunWithPassesMinutesToStartTimer(t *testing.T) {
	mock := &mockTimer{active: true}

	runWith([]Timer{mock}, "5")

	if mock.startTimerMinutes != 5 {
		t.Errorf("expected StartTimer to be called with 5, got %d", mock.startTimerMinutes)
	}
}

func TestRunBreakWithPassesMinutesToStartBreakTimer(t *testing.T) {
	mock := &mockTimer{active: true}

	runBreakWith([]Timer{mock}, "10")

	if mock.startBreakTimerMinutes != 10 {
		t.Errorf("expected StartBreakTimer to be called with 10, got %d", mock.startBreakTimerMinutes)
	}
}

func TestRunTimerReturnsErrorForZeroMinutes(t *testing.T) {
	err := RunTimer("0", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for zero minutes")
	}
}

func TestRunTimerReturnsErrorForNonNumericInput(t *testing.T) {
	err := RunTimer("NotANumber", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestRunBreakTimerReturnsErrorForZeroMinutes(t *testing.T) {
	err := RunBreakTimer("0", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for zero minutes")
	}
}

func TestRunBreakTimerReturnsErrorForNonNumericInput(t *testing.T) {
	err := RunBreakTimer("NotANumber", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}
