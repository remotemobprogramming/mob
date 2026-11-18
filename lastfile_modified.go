package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/remotemobprogramming/mob/v5/say"
	"github.com/remotemobprogramming/mob/v5/workdir"
)

// lastModifiedFileSource is the original behavior: the most recently modified file according to git status.
type lastModifiedFileSource struct{}

func (lastModifiedFileSource) name() string { return "last modified file" }

func (lastModifiedFileSource) find() (lastFile, bool) {
	path := getPathOfLastModifiedFile()
	return lastFile{Path: path}, path != ""
}

// uses git status --porcelain. To work properly files have to be staged.
func getPathOfLastModifiedFile() string {
	rootDir := gitRootDir()
	files := getModifiedFiles(rootDir)
	lastModifiedFilePath := ""
	lastModifiedTime := time.Time{}

	say.Debug("Find last modified file")
	if len(files) == 1 {
		lastModifiedFilePath = files[0]
		say.Debug("Just one modified file: " + lastModifiedFilePath)
		return lastModifiedFilePath
	}

	for _, file := range files {
		unquotedFile := file
		if strings.HasPrefix(file, "\"") {
			var err error
			unquotedFile, err = strconv.Unquote(file)
			if err != nil {
				say.Warning("Could not unquote filename from git: " + file)
				say.Warning(err.Error())
				continue
			}
		}
		absoluteFilepath := rootDir + "/" + unquotedFile
		say.Debug(absoluteFilepath)
		info, err := os.Stat(absoluteFilepath)
		if err != nil {
			say.Warning("Could not get statistics of file: " + absoluteFilepath)
			say.Warning(err.Error())
			continue
		}
		modTime := info.ModTime()
		if modTime.After(lastModifiedTime) {
			lastModifiedTime = modTime
			lastModifiedFilePath = file
		}
		say.Debug(modTime.String())
	}
	return lastModifiedFilePath
}

// uses git status --porcelain. To work properly files have to be staged.
func getModifiedFiles(rootDir string) []string {
	say.Debug("Find modified files")
	oldWorkingDir := workdir.Path
	workdir.Path = rootDir
	gitstatus := silentgit("status", "--porcelain")
	workdir.Path = oldWorkingDir
	lines := strings.Split(gitstatus, "\n")
	files := []string{}
	for _, line := range lines {
		relativeFilepath := ""
		if strings.HasPrefix(line, "M") {
			relativeFilepath = strings.TrimPrefix(line, "M")
		} else if strings.HasPrefix(line, "A") {
			relativeFilepath = strings.TrimPrefix(line, "A")
		} else {
			continue
		}
		relativeFilepath = strings.TrimSpace(relativeFilepath)
		say.Debug(relativeFilepath)
		files = append(files, relativeFilepath)
	}
	return files
}
