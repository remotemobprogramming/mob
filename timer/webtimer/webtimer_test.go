package webtimer_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/test"
	"github.com/remotemobprogramming/mob/v5/timer/webtimer"
)

func newCapturingServer(t *testing.T) (*httptest.Server, *string, *[]byte) {
	t.Helper()
	var capturedMethod string
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server, &capturedMethod, &capturedBody
}

func TestIsActiveWhenRoomIsSet(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "testroom"

	timer := webtimer.NewWebTimer(cfg)

	test.Equals(t, true, timer.IsActive())
}

func TestIsInactiveWhenRoomIsEmpty(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = ""

	timer := webtimer.NewWebTimer(cfg)

	test.Equals(t, false, timer.IsActive())
}

func TestUsesWipBranchQualifierAsRoom(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = ""
	cfg.TimerRoomUseWipBranchQualifier = true
	cfg.WipBranchQualifier = "feature-x"

	timer := webtimer.NewWebTimer(cfg)

	test.Equals(t, true, timer.IsActive())
}

func TestUsesTimerRoomWhenWipBranchQualifierIsEmpty(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "myroom"
	cfg.TimerRoomUseWipBranchQualifier = true
	cfg.WipBranchQualifier = ""

	timer := webtimer.NewWebTimer(cfg)

	test.Equals(t, true, timer.IsActive())
}

func TestStartTimerSendsPutWithTimerAndUser(t *testing.T) {
	server, capturedMethod, capturedBody := newCapturingServer(t)

	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "testroom"
	cfg.TimerUser = "testuser"
	cfg.TimerUrl = server.URL + "/"
	timer := webtimer.NewWebTimer(cfg)

	err := timer.StartTimer(10)

	var body map[string]interface{}
	json.Unmarshal(*capturedBody, &body)
	test.Equals(t, nil, err)
	test.Equals(t, "PUT", *capturedMethod)
	test.Equals(t, float64(10), body["timer"])
	test.Equals(t, "testuser", body["user"])
}

func TestStartBreakTimerSendsPutWithBreakTimerAndUser(t *testing.T) {
	server, capturedMethod, capturedBody := newCapturingServer(t)

	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "testroom"
	cfg.TimerUser = "testuser"
	cfg.TimerUrl = server.URL + "/"
	timer := webtimer.NewWebTimer(cfg)

	err := timer.StartBreakTimer(5)

	var body map[string]interface{}
	json.Unmarshal(*capturedBody, &body)
	test.Equals(t, nil, err)
	test.Equals(t, "PUT", *capturedMethod)
	test.Equals(t, float64(5), body["breaktimer"])
	test.Equals(t, "testuser", body["user"])
}
