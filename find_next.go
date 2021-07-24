package main

func findNextTypist(lastCommitters []string, gitUserName string) (nextTypist string, previousCommitters []string) {
	for i := 0; i < len(lastCommitters); i++ {
		if lastCommitters[i] == gitUserName && i > 0 {
			nextTypist = lastCommitters[i-1]
			return
		}
		// Do not add the last committer multiple times.
		if i == 0 || previousCommitters[0] != lastCommitters[i] {
			previousCommitters = prepend(previousCommitters, lastCommitters[i])
		}
	}
	return
}

func prepend(list []string, element string) []string {
	list = append(list, element)
	copy(list[1:], list)
	list[0] = element
	return list
}
