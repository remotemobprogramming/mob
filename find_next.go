package main

func findNextTypist(lastCommitters []string, gitUserName string) (nextTypist string, previousCommitters []string) {
	for i := 0; i < len(lastCommitters); i++ {
		if lastCommitters[i] == gitUserName && i > 0 {
			nextTypist = lastCommitters[i-1]
			return
		}
		previousCommitters = prepend(previousCommitters, lastCommitters[i])
	}
	return
}

func prepend(list []string, element string) []string {
	list = append(list, element)
	copy(list[1:], list)
	list[0] = element
	return list
}
