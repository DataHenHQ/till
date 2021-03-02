package proxy

// PageConfig is where the page configuration is set
type PageConfig struct {
	ForceUA bool   // if true, overrides the User-Agent header
	UaType  string // default to "desktop"
}
