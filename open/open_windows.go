//go:build windows
// +build windows

package open

import (
	"os"
	"os/exec"
	"path/filepath"
)

var (
	cmd      = "url.dll,FileProtocolHandler"
	runDll32 = filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe")
)

func open(input string) error {
	return exec.Command(runDll32, cmd, input).Run()
}
