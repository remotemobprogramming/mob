package ask

import (
	"bufio"
	"github.com/remotemobprogramming/mob/v5/say"
	"io"
	"os"
	"strings"
)

func YesNo(q string) bool {
	say.Say(q)
	reader := ReadFromConsole(io.Reader(os.Stdin))
	for {
		text, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			say.Error("Error reading from console")
			return false
		}

		if strings.ToLower(text) == "y\n" || text == "\n" {
			return true
		} else {
			say.Say("Aborted")
			return false
		}
	}
}

var ReadFromConsole = func(reader io.Reader) *bufio.Reader {
	return bufio.NewReader(reader)
}
