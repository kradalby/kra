package web

// ServerConfig captures the common configuration needed to start a KraWeb server.
type ServerConfig struct {
	Hostname        string
	LocalAddr       string
	AuthKey         string
	AuthKeyPath     string
	EnableTailscale bool
}
