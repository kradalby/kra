package web

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFirstLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo.bar.baz", "foo"},
		{"single", "single"},
		{"", ""},
		{"dot.", "dot"},
		{".leading", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := firstLabel(tt.input)
			if got != tt.want {
				t.Errorf("firstLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func mustNewServer(t *testing.T, cfg ServerConfig, opts ...Option) *KraWeb {
	t.Helper()

	k, err := NewServer(cfg, opts...)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	return k
}

func TestNewServerDefaults(t *testing.T) {
	k := mustNewServer(t, ServerConfig{Hostname: "testhost", AuthKeyPath: "/tmp/key", LocalAddr: "localhost:0"})

	if k.hostname != "testhost" {
		t.Errorf("hostname = %q, want %q", k.hostname, "testhost")
	}
	if k.tsKeyPath != "/tmp/key" {
		t.Errorf("tsKeyPath = %q, want %q", k.tsKeyPath, "/tmp/key")
	}
	if k.localAddr != "localhost:0" {
		t.Errorf("localAddr = %q, want %q", k.localAddr, "localhost:0")
	}
	if k.verbose {
		t.Error("verbose should be false by default")
	}
	if k.enableTS {
		t.Error("enableTS should be false by default")
	}
	if k.controlURL != "" {
		t.Errorf("controlURL = %q, want empty string", k.controlURL)
	}
	if k.mux == nil {
		t.Error("mux should not be nil")
	}
	if k.tsmux == nil {
		t.Error("tsmux should not be nil")
	}
}

func TestWithOptions(t *testing.T) {
	customLogger := slog.Default()
	customStdLogger := log.Default()

	k := mustNewServer(t,
		ServerConfig{Hostname: "host", LocalAddr: "localhost:0"},
		WithControlURL("https://control.example.com"),
		WithVerbose(true),
		WithLogger(customLogger),
		WithStdLogger(customStdLogger),
		WithTailscale(true),
		WithTailscaleStateDir("/tmp/state"),
		WithAuthKey("tskey-auth-test"),
	)

	if k.controlURL != "https://control.example.com" {
		t.Errorf("controlURL = %q, want %q", k.controlURL, "https://control.example.com")
	}
	if !k.verbose {
		t.Error("verbose should be true")
	}
	if k.logger != customLogger {
		t.Error("logger not set correctly")
	}
	if k.stdLogger != customStdLogger {
		t.Error("stdLogger not set correctly")
	}
	if !k.enableTS {
		t.Error("enableTS should be true")
	}
	if k.tsStateDir != "/tmp/state" {
		t.Errorf("tsStateDir = %q, want %q", k.tsStateDir, "/tmp/state")
	}
	if k.authKey != "tskey-auth-test" {
		t.Errorf("authKey = %q, want %q", k.authKey, "tskey-auth-test")
	}
}

func TestHandleRegistersOnBothMuxes(t *testing.T) {
	k := mustNewServer(t, ServerConfig{Hostname: "host", LocalAddr: "localhost:0"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	k.Handle("/test", handler)

	// Test that the route is registered on the local mux
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	k.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("local mux: got status %d, want %d", rec.Code, http.StatusOK)
	}

	// Test that the route is registered on the TS mux
	rec = httptest.NewRecorder()
	k.tsmux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("ts mux: got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleTSOnlyRegistersOnlyOnTSMux(t *testing.T) {
	k := mustNewServer(t, ServerConfig{Hostname: "host", LocalAddr: "localhost:0"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ts-only"))
	})

	k.HandleTSOnly("/ts-secret", handler)

	// Test that the route IS on the TS mux
	req := httptest.NewRequest(http.MethodGet, "/ts-secret", nil)
	rec := httptest.NewRecorder()
	k.tsmux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("ts mux: got status %d, want %d", rec.Code, http.StatusOK)
	}

	// Test that the route is NOT on the local mux (will 404 or redirect to /)
	rec = httptest.NewRecorder()
	k.mux.ServeHTTP(rec, req)
	if rec.Body.String() == "ts-only" {
		t.Error("local mux should not have the ts-only handler")
	}
}

func TestNewServerValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerConfig
		wantErr bool
	}{
		{
			name: "valid config without tailscale",
			cfg: ServerConfig{
				Hostname:        "test",
				LocalAddr:       "localhost:0",
				EnableTailscale: false,
			},
			wantErr: false,
		},
		{
			name: "missing local addr",
			cfg: ServerConfig{
				Hostname:        "test",
				LocalAddr:       "",
				EnableTailscale: false,
			},
			wantErr: true,
		},
		{
			name: "tailscale enabled without hostname",
			cfg: ServerConfig{
				Hostname:        "",
				LocalAddr:       "localhost:0",
				EnableTailscale: true,
			},
			wantErr: true,
		},
		{
			name: "valid config with tailscale",
			cfg: ServerConfig{
				Hostname:        "myhost",
				LocalAddr:       "localhost:0",
				EnableTailscale: true,
				AuthKey:         "tskey-auth-test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kw, err := NewServer(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && kw == nil {
				t.Error("NewServer() returned nil KraWeb for valid config")
			}
		})
	}
}

func TestDebugHandlerWithTS(t *testing.T) {
	k := mustNewServer(t, ServerConfig{Hostname: "host", LocalAddr: "localhost:0", EnableTailscale: true})
	dh := k.DebugHandler()
	if dh == nil {
		t.Error("DebugHandler() should not return nil when TS is enabled")
	}
}

func TestDebugHandlerWithoutTS(t *testing.T) {
	k := mustNewServer(t, ServerConfig{Hostname: "host", LocalAddr: "localhost:0"})
	dh := k.DebugHandler()
	if dh != nil {
		t.Error("DebugHandler() should return nil when TS is disabled")
	}
}

func TestNewServerConfiguresTailscale(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(keyPath, []byte("  tskey-auth-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		cfg     ServerConfig
		opts    []Option
		wantKey string
		wantErr bool
	}{
		{
			name:    "key from config",
			cfg:     ServerConfig{AuthKey: " tskey-auth-cfg "},
			wantKey: "tskey-auth-cfg",
		},
		{
			name:    "key from file",
			cfg:     ServerConfig{AuthKeyPath: keyPath},
			wantKey: "tskey-auth-file",
		},
		{
			name:    "key beats file",
			cfg:     ServerConfig{AuthKey: "tskey-auth-cfg", AuthKeyPath: keyPath},
			wantKey: "tskey-auth-cfg",
		},
		{
			name:    "option beats config",
			cfg:     ServerConfig{AuthKey: "tskey-auth-cfg"},
			opts:    []Option{WithAuthKey("tskey-auth-opt")},
			wantKey: "tskey-auth-opt",
		},
		{
			name:    "missing key file",
			cfg:     ServerConfig{AuthKeyPath: filepath.Join(t.TempDir(), "nope")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Hostname = "host"
			tt.cfg.LocalAddr = "localhost:0"
			tt.cfg.EnableTailscale = true

			k, err := NewServer(tt.cfg, tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewServer() error = nil, want error")
				}

				return
			}
			if err != nil {
				t.Fatalf("NewServer() error = %v", err)
			}

			// tsnet reads its config once, on first Start or LocalClient,
			// so it must be complete before NewServer returns.
			if k.tsSrv.AuthKey != tt.wantKey {
				t.Errorf("tsSrv.AuthKey = %q, want %q", k.tsSrv.AuthKey, tt.wantKey)
			}
			if k.tsSrv.Logf == nil {
				t.Error("tsSrv.Logf is nil")
			}

			for _, path := range []string{"/metrics", "/who", "/quitquitquit"} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				if _, pattern := k.tsmux.Handler(req); pattern != path {
					t.Errorf("tsmux pattern for %s = %q, want %q", path, pattern, path)
				}
			}
		})
	}
}

// Mirrors hvor, which grabs the LocalClient (starting tsnet) before serving.
func TestListenAndServeAfterLocalClient(t *testing.T) {
	k, err := NewServer(
		ServerConfig{
			Hostname:        "kra-test",
			LocalAddr:       "127.0.0.1:0",
			AuthKey:         "tskey-auth-test",
			EnableTailscale: true,
		},
		WithVerbose(true),
		WithTailscaleStateDir(t.TempDir()),
		WithControlURL("http://127.0.0.1:1"),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	// Runs before TempDir removal; waits out the async Close in ListenAndServe.
	t.Cleanup(func() { _ = k.tsSrv.Close() })

	if k.TailscaleLocalClient() == nil {
		t.Fatal("TailscaleLocalClient() = nil")
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	if err := k.ListenAndServe(ctx); err != nil {
		t.Fatalf("ListenAndServe() error = %v", err)
	}
}
