package proxy

// PageConfig is where the page configuration is set
type PageConfig struct {
	ForceUA  bool   // if true, overrides the User-Agent header
	UaType   string // default to "desktop"
	UseProxy bool
}

func generatePageConfig() (conf *PageConfig) {
	useProxy := false
	if ProxyCount > 0 {
		useProxy = true
	}

	return &PageConfig{
		ForceUA:  ForceUA,
		UaType:   UAType,
		UseProxy: useProxy,
	}
}
