package web

import "fmt"

// ServerConfig captures the common configuration needed to start a KraWeb server.
type ServerConfig struct {
	Hostname        string
	LocalAddr       string
	AuthKey         string
	AuthKeyPath     string
	EnableTailscale bool
}

// NewServer creates a KraWeb server using the provided configuration.
// It wires common options (local listen address, Tailscale hostname/auth)
// so that applications can stay simple.
func NewServer(cfg ServerConfig, opts ...Option) (*KraWeb, error) {
	if cfg.LocalAddr == "" {
		return nil, fmt.Errorf("local listen address is required")
	}

	if cfg.EnableTailscale && cfg.Hostname == "" {
		return nil, fmt.Errorf("tailscale hostname is required when enabling tailscale")
	}

	baseOpts := []Option{WithTailscale(cfg.EnableTailscale)}
	if cfg.AuthKey != "" {
		baseOpts = append(baseOpts, WithAuthKey(cfg.AuthKey))
	}

	opts = append(baseOpts, opts...)

	return NewKraWeb(cfg.Hostname, cfg.AuthKeyPath, cfg.LocalAddr, opts...), nil
}
