package webtimer_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/timer/webtimer"
)

func TestIsActiveWhenRoomIsSet(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "testroom"

	timer := webtimer.NewWebTimer(cfg)

	if !timer.IsActive() {
		t.Error("expected timer to be active when TimerRoom is set")
	}
}

func TestIsInactiveWhenRoomIsEmpty(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = ""

	timer := webtimer.NewWebTimer(cfg)

	if timer.IsActive() {
		t.Error("expected timer to be inactive when TimerRoom is empty")
	}
}

func TestUsesWipBranchQualifierAsRoom(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = ""
	cfg.TimerRoomUseWipBranchQualifier = true
	cfg.WipBranchQualifier = "feature-x"

	timer := webtimer.NewWebTimer(cfg)

	if !timer.IsActive() {
		t.Error("expected timer to be active when WipBranchQualifier is used as room")
	}
}

func TestUsesTimerRoomWhenWipBranchQualifierIsEmpty(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "myroom"
	cfg.TimerRoomUseWipBranchQualifier = true
	cfg.WipBranchQualifier = ""

	timer := webtimer.NewWebTimer(cfg)

	if !timer.IsActive() {
		t.Error("expected timer to use TimerRoom when WipBranchQualifier is empty")
	}
}

func TestStartTimerSendsPutWithTimerAndUser(t *testing.T) {
	var capturedBody []byte
	var capturedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "testroom"
	cfg.TimerUser = "testuser"
	cfg.TimerUrl = server.URL + "/"
	timer := webtimer.NewWebTimer(cfg)

	err := timer.StartTimer(10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMethod != "PUT" {
		t.Errorf("expected PUT, got %s", capturedMethod)
	}
	var body map[string]interface{}
	json.Unmarshal(capturedBody, &body)
	if body["timer"] != float64(10) {
		t.Errorf("expected timer=10, got %v", body["timer"])
	}
	if body["user"] != "testuser" {
		t.Errorf("expected user=testuser, got %v", body["user"])
	}
}

func TestStartBreakTimerSendsPutWithBreakTimerAndUser(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.GetDefaultConfiguration()
	cfg.TimerRoom = "testroom"
	cfg.TimerUser = "testuser"
	cfg.TimerUrl = server.URL + "/"
	timer := webtimer.NewWebTimer(cfg)

	err := timer.StartBreakTimer(5)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var body map[string]interface{}
	json.Unmarshal(capturedBody, &body)
	if body["breaktimer"] != float64(5) {
		t.Errorf("expected breaktimer=5, got %v", body["breaktimer"])
	}
	if body["user"] != "testuser" {
		t.Errorf("expected user=testuser, got %v", body["user"])
	}
}
