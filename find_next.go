package main

func findNextTypist(lastCommitters []string, gitUserName string) (nextTypist string, previousCommitters []string) {
	numberOfLastCommitters := len(lastCommitters)
	for i := 0; i < numberOfLastCommitters; i++ {
		if lastCommitters[i] == gitUserName && i > 0 {
			nextTypist = lastCommitters[i-1]
			if nextTypist != gitUserName {
				// '2*i+1' defines how far we look ahead. It is the number of already processed elements.
				lookaheadThreshold := min(2*i+1, len(lastCommitters))
				previousMobber := lookahead(lastCommitters[:i], lastCommitters[i:lookaheadThreshold])
				if previousMobber != "" {
					nextTypist = previousMobber
				}
				return
			}
		}
		// Do not add the last committer multiple times.
		if i == 0 || previousCommitters[0] != lastCommitters[i] {
			previousCommitters = prepend(previousCommitters, lastCommitters[i])
		}
	}
	return
}

func lookahead(processedCommitters []string, previousCommitters []string) (nextTypist string) {
	for i := 0; i < len(previousCommitters); i++ {
		if !contains(processedCommitters, previousCommitters[i]) {
			nextTypist = previousCommitters[i]
		}
	}
	return
}

func contains(list []string, element string) bool {
	for i := 0; i < len(list); i++ {
		if list[i] == element {
			return true
		}
	}
	return false
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func prepend(list []string, element string) []string {
	list = append(list, element)
	copy(list[1:], list)
	list[0] = element
	return list
}
