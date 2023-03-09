package open

var OpenInBrowser = func(url string) error {
	return open(url)
}
