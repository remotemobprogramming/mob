package webtimer

import (
	"encoding/json"
	"fmt"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/git"
	"github.com/remotemobprogramming/mob/v5/httpclient"
)

// WebTimer is a Timer implementation that notifies a remote timer service via HTTP.
type WebTimer struct {
	room          string
	timerUser     string
	timerUrl      string
	timerInsecure bool
}

func NewWebTimer(configuration config.Configuration) WebTimer {
	room := configuration.TimerRoom
	if configuration.TimerRoomUseWipBranchQualifier && configuration.WipBranchQualifier != "" {
		room = configuration.WipBranchQualifier
	}
	return WebTimer{
		room:          room,
		timerUser:     getUserForMobTimer(configuration.TimerUser),
		timerUrl:      configuration.TimerUrl,
		timerInsecure: configuration.TimerInsecure,
	}
}

func getUserForMobTimer(userOverride string) string {
	if userOverride == "" {
		gitClient := &git.Client{}
		return gitClient.UserName()
	}
	return userOverride
}

func (t WebTimer) IsActive() bool {
	return t.room != ""
}

func (t WebTimer) StartTimer(minutes int) error {
	if err := httpPutTimer(minutes, t.room, t.timerUser, t.timerUrl, t.timerInsecure); err != nil {
		return fmt.Errorf("remote timer couldn't be started: %w", err)
	}
	return nil
}

func (t WebTimer) StartBreakTimer(minutes int) error {
	if err := httpPutBreakTimer(minutes, t.room, t.timerUser, t.timerUrl, t.timerInsecure); err != nil {
		return fmt.Errorf("remote break timer couldn't be started: %w", err)
	}
	return nil
}

func httpPutTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"timer": timeoutInMinutes,
		"user":  user,
	})
	client := httpclient.CreateHttpClient(disableSSLVerification)
	_, err := client.SendRequest(putBody, "PUT", timerService+room)
	return err
}

func httpPutBreakTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"breaktimer": timeoutInMinutes,
		"user":       user,
	})
	client := httpclient.CreateHttpClient(disableSSLVerification)
	_, err := client.SendRequest(putBody, "PUT", timerService+room)
	return err
}
