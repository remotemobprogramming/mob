//go:build !windows && !darwin
// +build !windows,!darwin

package open

import "os/exec"

func open(url string) error {
	return exec.Command("xdg-open", url).Run()
}
