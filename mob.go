package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const branch = "mob-session"
func isDebug() bool {
	_, isSet:= os.LookupEnv("MOB_DEBUG")
	return isSet
}

// -d to enable insights into what happened in git

func main() {
	argument := getCommand()
	if argument == "s" || argument == "start" {
		start()
	} else if argument == "n" || argument == "next" {
		next()
	} else if argument == "d" || argument == "done" {
		done()
	} else if argument == "r" || argument == "reset" {
		reset()
	} else if argument == "h" || argument == "help" {
		help()
	} else {
		status()
	}
}

func reset() {
	git("checkout", "master")
	git("branch", "-D", branch)
	git("push", "origin", "--delete", branch)
}

func start() {
	git("checkout", "-b", branch)
	git("pull", "--force")
	say("start hacking")
}

func next() {
	git("add", ".", "--all")
	git("commit", "--message", "\"WIP in Mob Session [ci-skip]\"")
	git("push")
	say("join the 'rest of the mob'")
}

func done() {
	git("checkout", "master")
	git("merge", "--squash", branch)
	git("branch", "-D", branch)
	git("push", "origin", "--delete", branch)
	say("lean back, you survived your mob session :-)")
	say("execute 'git commit' to describe what the mob achieved")
}

func status() {
	if isMobbing() {
		say("mobbing in progress")
	} else {
		say("you aren't mobbing right now")
	}
}

func isMobbing() bool {
	output := git("branch")
	isMobbing := strings.Contains(output, "* "+branch)
	return isMobbing
}

func help() {
	say("usage")
	say("\tmob [s]tart \t# start mobbing as typist")
	say("\tmob [n]ext \t# hand over to next typist")
	say("\tmob [d]one \t# finish mob session")
	say("\tmob [r]eset \t# resets any unfinished mob session")
	say("\tmob [h]elp \t# prints this help")
}

func git(args ...string) string {
	command := exec.Command("git", args...)
	if isDebug() {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if isDebug() {
		fmt.Println(output)
	}
	if err != nil {
		fmt.Println(err)
	}
	return output
}

func say(s string) {
	fmt.Println(s)
}

func getCommand() string {
	args := os.Args
	if len(args) <= 1 {
		return ""
	}
	return args[1]
}
