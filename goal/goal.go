package goal

import (
	"encoding/json"
	"errors"
	"fmt"
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/httpclient"
	"github.com/remotemobprogramming/mob/v4/say"
	"io"
	"os"
	"strings"
)

type GoalResponse struct {
	Goal string `json:"goal"`
}

type DeleteGoalRequest struct {
	User string `json:"user"`
}

type PutGoalRequest struct {
	Goal string `json:"goal"`
	User string `json:"user"`
}

func Goal(configuration config.Configuration, parameter []string) {
	if err := goal(configuration, parameter); err != nil {
		say.Error(err.Error())
		exit(1)
	}
}

func goal(configuration config.Configuration, parameter []string) error {
	if configuration.TimerRoom == "" {
		return errors.New("No room sepcified. Set MOB_TIMER_ROOM to your timer.mob.sh room in .mob file.")
	}
	if len(parameter) > 0 {
		if parameter[0] == "--delete" {
			err := deleteCurrentGoal(configuration)
			if err != nil {
				return err
			}
		} else {
			err := setNewGoal(configuration, strings.Join(parameter, " "))
			if err != nil {
				return err
			}
		}
	} else {
		err := showGoal(configuration)
		if err != nil {
			return err
		}
	}
	return nil
}

func setNewGoal(configuration config.Configuration, goal string) error {
	if err := putGoalHttp(goal, configuration); err != nil {
		say.Debug(err.Error())
		return errors.New("Could not set new goal. An error occurred while sending the request.")
	}
	say.Info(fmt.Sprintf("Set new goal to \"%s\"", goal))
	return nil
}

func putGoalHttp(goal string, configuration config.Configuration) error {
	requestBody, err := json.Marshal(PutGoalRequest{Goal: goal, User: configuration.TimerUser})
	if err != nil {
		return err
	}
	_, err = httpclient.SendRequest(requestBody, "PUT", getGoalUrl(configuration), configuration.TimerInsecure)
	return err
}

func getGoalUrl(configuration config.Configuration) string {
	return configuration.TimerUrl + configuration.TimerRoom + "/goal"
}

func deleteCurrentGoal(configuration config.Configuration) error {
	err := deleteGoalHttp(configuration.TimerRoom, configuration.TimerUser, configuration.TimerUrl, configuration.TimerInsecure)
	if err != nil {
		say.Debug(err.Error())
		return errors.New("Could not delete goal. An error occurred while sending the request.")
	}
	say.Info("Current goal has been deleted!")
	return nil
}

func deleteGoalHttp(room string, user string, timerService string, disableSslVerification bool) error {
	requestBody, err := json.Marshal(DeleteGoalRequest{User: user})
	if err != nil {
		return err
	}
	_, err = httpclient.SendRequest(requestBody, "DELETE", timerService+room+"/goal", disableSslVerification)
	return err
}

func showGoal(configuration config.Configuration) error {
	goal, err := getGoalHttp(configuration.TimerRoom, configuration.TimerUrl, configuration.TimerInsecure)
	if err != nil {
		say.Debug(err.Error())
		return errors.New("Could not get goal. An error occurred while sending the request.")
	}
	if goal == "" {
		say.Fix("No goal set. To set a goal, use", configuration.Mob("goal <your awesome goal>"))
		return nil
	}
	say.Info(goal)
	return nil
}
func getGoalHttp(room string, timerService string, disableSslVerification bool) (string, error) {
	url := timerService + room + "/goal"
	response, err := httpclient.GetHttpClient(disableSslVerification).Get(url)
	if err != nil {
		say.Debug(err.Error())
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", errors.New("got an error while requesting it: " + url + " " + response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		say.Debug(err.Error())
		return "", err
	}
	var goalResponse GoalResponse
	if err := json.Unmarshal(body, &goalResponse); err != nil {
		say.Debug(err.Error())
		return "", err
	}
	return goalResponse.Goal, nil
}

var exit = func(code int) {
	os.Exit(code)
}
