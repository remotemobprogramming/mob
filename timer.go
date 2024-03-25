package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/say"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"
)

func StartTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startTimer(configuration.Timer, configuration); err != nil {
		exit(1)
	}
}

func startTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		say.Error("No timer configured, not starting timer")
		exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)
		if err != nil {
			say.Error("remote timer couldn't be started")
			say.Error(err.Error())
			exit(1)
		}
	}

	if startLocalTimer {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand(configuration.VoiceMessage, configuration.VoiceCommand), getNotifyCommand(configuration.NotifyMessage, configuration.NotifyCommand), "echo \"mobTimer\"")

		if err != nil {
			say.Error(fmt.Sprintf("timer couldn't be started on your system (%s)", runtime.GOOS))
			say.Error(err.Error())
			exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
	return nil
}

func getMobTimerRoom(configuration config.Configuration) string {
	if !isGit() {
		say.Debug("timer not in git repository, using MOB_TIMER_ROOM for room name")
		return configuration.TimerRoom
	}

	currentWipBranchQualifier := configuration.WipBranchQualifier
	if currentWipBranchQualifier == "" {
		currentBranch := gitCurrentBranch()
		currentBaseBranch, _ := determineBranches(currentBranch, gitBranches(), configuration)

		if currentBranch.IsWipBranch(configuration) {
			wipBranchWithoutWipPrefix := currentBranch.removeWipPrefix(configuration).Name
			currentWipBranchQualifier = removePrefix(removePrefix(wipBranchWithoutWipPrefix, currentBaseBranch.Name), configuration.WipBranchQualifierSeparator)
		}
	}

	if configuration.TimerRoomUseWipBranchQualifier && currentWipBranchQualifier != "" {
		say.Info("Using wip branch qualifier for room name")
		return currentWipBranchQualifier
	}

	return configuration.TimerRoom
}

func StartBreakTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startBreakTimer(configuration.Timer, configuration); err != nil {
		exit(1)
	}
}

func startBreakTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting break timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		say.Error("No break timer configured, not starting break timer")
		exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutBreakTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)

		if err != nil {
			say.Error("remote break timer couldn't be started")
			say.Error(err.Error())
			exit(1)
		}
	}

	if startLocalTimer {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand("mob start", configuration.VoiceCommand), getNotifyCommand("mob start", configuration.NotifyCommand), "echo \"mobTimer\"")

		if err != nil {
			say.Error(fmt.Sprintf("break timer couldn't be started on your system (%s)", runtime.GOOS))
			say.Error(err.Error())
			exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min break timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". So take a break now! :)")
	return nil
}

func getUserForMobTimer(userOverride string) string {
	if userOverride == "" {
		return gitUserName()
	}
	return userOverride
}

func toMinutes(timerInMinutes string) (error, int) {
	timeoutInMinutes, err := strconv.Atoi(timerInMinutes)
	if err != nil || timeoutInMinutes < 1 {
		say.Error(fmt.Sprintf("The parameter must be an integer number greater then zero"))
		return errors.New("The parameter must be an integer number greater then zero"), 0
	}
	return nil, timeoutInMinutes
}

func httpPutTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"timer": timeoutInMinutes,
		"user":  user,
	})
	return sendRequest(putBody, "PUT", timerService+room, disableSSLVerification)
}

func httpPutBreakTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"breaktimer": timeoutInMinutes,
		"user":       user,
	})
	return sendRequest(putBody, "PUT", timerService+room, disableSSLVerification)
}

func sendRequest(requestBody []byte, requestMethod string, requestUrl string, disableSSLVerification bool) error {
	say.Info(requestMethod + " " + requestUrl + " " + string(requestBody))

	responseBody := bytes.NewBuffer(requestBody)
	request, requestCreationError := http.NewRequest(requestMethod, requestUrl, responseBody)

	httpClient := http.DefaultClient
	if disableSSLVerification {
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: transCfg}
	}

	if requestCreationError != nil {
		return fmt.Errorf("failed to create the http request object: %w", requestCreationError)
	}

	request.Header.Set("Content-Type", "application/json")
	response, responseErr := httpClient.Do(request)
	if e, ok := responseErr.(*url.Error); ok {
		switch e.Err.(type) {
		case x509.UnknownAuthorityError:
			say.Error("The timer.mob.sh SSL certificate is signed by an unknown authority!")
			say.Fix("HINT: You can ignore that by adding MOB_TIMER_INSECURE=true to your configuration or environment.",
				"echo MOB_TIMER_INSECURE=true >> ~/.mob")
			return fmt.Errorf("failed, to amke the http request: %w", responseErr)

		default:
			return fmt.Errorf("failed to make the http request: %w", responseErr)

		}
	}

	if responseErr != nil {
		return fmt.Errorf("failed to make the http request: %w", responseErr)
	}
	defer response.Body.Close()
	body, responseReadingErr := io.ReadAll(response.Body)
	if responseReadingErr != nil {
		return fmt.Errorf("failed to read the http response: %w", responseReadingErr)
	}
	if string(body) != "" {
		say.Info(string(body))
	}
	return nil
}

func getSleepCommand(timeoutInSeconds int) string {
	return fmt.Sprintf("sleep %d", timeoutInSeconds)
}

func getVoiceCommand(message string, voiceCommand string) string {
	if len(voiceCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(voiceCommand, message)
}

func getNotifyCommand(message string, notifyCommand string) string {
	if len(notifyCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(notifyCommand, message)
}
