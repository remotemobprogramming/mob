package exit

import "os"

// Exit kann in Tests ueberschrieben werden, um os.Exit abzufangen.
var Exit = os.Exit
