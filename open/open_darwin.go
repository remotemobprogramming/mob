//go:build darwin
// +build darwin

package open

import "os/exec"

func open(url string) error {
	return exec.Command("open", url).Run()
}
