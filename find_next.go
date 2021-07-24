package main

func findNextTypist(lastCommitters []string, gitUserName string) (string, string) {
	var history = ""
	for i := 0; i < len(lastCommitters); i++ {
		if lastCommitters[i] == gitUserName && i > 0 {
			return lastCommitters[i-1], history
		}
		if history != "" {
			history = ", " + history
		}
		history = lastCommitters[i] + history
	}
	return "", history
}
