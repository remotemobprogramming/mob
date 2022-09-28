package test

import (
	"fmt"
	"github.com/remotemobprogramming/mob/v4/say"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

var (
	workingDir string
)

func Equals(t *testing.T, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		t.Log(string(debug.Stack()))
		failWithFailure(t, exp, act)
	}
}

func failWithFailure(t *testing.T, exp interface{}, act interface{}) {
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
	t.FailNow()
}

func CreateFile(t *testing.T, filename string, content string) (pathToFile string) {
	contentAsBytes := []byte(content)
	pathToFile = workingDir + "/" + filename
	err := ioutil.WriteFile(pathToFile, contentAsBytes, 0644)
	if err != nil {
		failWithFailure(t, "creating file "+filename+" with content "+content, "error")
	}
	return
}

func SetWorkingDir(dir string) {
	workingDir = dir
	say.Say("\n===== cd " + dir)
}

func CaptureOutput(t *testing.T) *string {
	messages := ""
	say.PrintToConsole = func(text string) {
		t.Log(strings.TrimRight(text, "\n"))
		messages += text
	}
	return &messages
}

func AssertOutputContains(t *testing.T, output *string, contains string) {
	currentOutput := *output
	if !strings.Contains(currentOutput, contains) {
		failWithFailure(t, "output contains '"+contains+"'", currentOutput)
	}
}

func AssertOutputNotContains(t *testing.T, output *string, notContains string) {
	if strings.Contains(*output, notContains) {
		failWithFailure(t, "output not contains "+notContains, output)
	}
}
