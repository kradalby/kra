package web

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/arl/statsviz"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
	"tailscale.com/tsweb"
)

type KraWeb struct {
	// hostname is the name that will be used when joining Tailscale
	hostname  string
	tsKeyPath string
	authKey   string
	localAddr string

	controlURL string
	verbose    bool
	logger     *slog.Logger
	stdLogger  *log.Logger
	enableTS   bool

	mux   *http.ServeMux
	tsmux *http.ServeMux
	tsSrv *tsnet.Server

	debugHandler *tsweb.DebugHandler
}

type Option = func(c *KraWeb)

func WithControlURL(url string) Option {
	return func(kw *KraWeb) {
		kw.controlURL = url
	}
}

func WithVerbose(b bool) Option {
	return func(kw *KraWeb) {
		kw.verbose = b
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(kw *KraWeb) {
		kw.logger = l
	}
}

func WithStdLogger(l *log.Logger) Option {
	return func(kw *KraWeb) {
		kw.stdLogger = l
	}
}

func WithTailscale(b bool) Option {
	return func(kw *KraWeb) {
		kw.enableTS = b
	}
}

func WithAuthKey(key string) Option {
	return func(kw *KraWeb) {
		kw.authKey = key
	}
}

func NewKraWeb(
	hostname string,
	tsKeyPath string,
	localAddr string,
	opts ...Option,
) *KraWeb {
	k := &KraWeb{
		hostname:   hostname,
		tsKeyPath:  tsKeyPath,
		localAddr:  localAddr,
		controlURL: "",
		verbose:    false,
		logger:     slog.Default(),
		stdLogger:  log.Default(),
		enableTS:   false,
	}

	for _, opt := range opts {
		opt(k)
	}

	k.mux = http.NewServeMux()
	k.tsmux = http.NewServeMux()

	debugHandler := tsweb.Debugger(k.tsmux)
	k.debugHandler = debugHandler

	if err := statsviz.Register(k.tsmux); err == nil {
		k.debugHandler.URL("/debug/statsviz", "Statsviz (visualise go metrics)")
	} else {
		k.logger.Warn("failed to register statsviz", slog.Any("error", err))
	}

	tsSrv := &tsnet.Server{
		Hostname:   k.hostname,
		Logf:       func(format string, args ...any) {},
		ControlURL: k.controlURL,
	}

	k.tsSrv = tsSrv

	return k
}

// DebugHandler returns the handler for the debug server.
// It can be used to add additional, application specific
// debug handlers.
// It is only available over Tailscale, and when tsnet is enabled.
func (k *KraWeb) DebugHandler() *tsweb.DebugHandler {
	return k.debugHandler
}

func (k *KraWeb) Handle(pattern string, handler http.Handler) {
	k.mux.Handle(pattern, handler)
	k.tsmux.Handle(pattern, handler)
}

func (k *KraWeb) HandleTSOnly(pattern string, handler http.Handler) {
	k.tsmux.Handle(pattern, handler)
}

func (k *KraWeb) TailscaleLocalClient() *tailscale.LocalClient {
	if k.tsSrv == nil {
		return nil
	}

	localClient, err := k.tsSrv.LocalClient()
	if err != nil {
		return nil
	}

	return localClient
}

func (k *KraWeb) ListenAndServe(ctx context.Context) error {
	logger := k.logger

	switch {
	case k.authKey != "":
		k.tsSrv.AuthKey = strings.TrimSpace(k.authKey)
	case k.tsKeyPath != "":
		key, err := os.ReadFile(k.tsKeyPath)
		if err != nil {
			return err
		}

		k.tsSrv.AuthKey = strings.TrimSpace(string(key))
	}

	if k.verbose && logger != nil {
		k.tsSrv.Logf = func(format string, args ...any) {
			logger.Info(fmt.Sprintf(format, args...))
		}
	}

	if k.enableTS {
		if err := k.tsSrv.Start(); err != nil {
			return err
		}

		localClient, err := k.tsSrv.LocalClient()
		if err != nil {
			return err
		}

		k.tsmux.Handle("/metrics", promhttp.Handler())
		k.tsmux.Handle("/who", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			who, err := localClient.WhoIs(r.Context(), r.RemoteAddr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}

			fmt.Fprintf(w, "<html><body><h1>Hello, world!</h1>\n")
			fmt.Fprintf(w, "<p>You are <b>%s</b> from <b>%s</b> (%s)</p>",
				html.EscapeString(who.UserProfile.LoginName),
				html.EscapeString(firstLabel(who.Node.ComputedName)),
				r.RemoteAddr)
		}))

		k.tsmux.Handle(
			"/quitquitquit",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				os.Exit(0)
			}),
		)

		tshttpSrv := &http.Server{
			Handler:     k.tsmux,
			ErrorLog:    k.stdLogger,
			ReadTimeout: 5 * time.Minute,
			Addr:        ":80",
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tshttpSrv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Warn("tailscale http shutdown error", slog.Any("error", err))
			}
			if err := k.tsSrv.Close(); err != nil {
				logger.Warn("tailscale server shutdown error", slog.Any("error", err))
			}
		}()

		// Starting HTTPS server
		go func() {
			ts443, err := k.tsSrv.Listen("tcp", ":443")
			if err != nil {
				logger.Error("failed to start https server", slog.Any("error", err))
				return
			}

			ts443 = tls.NewListener(ts443, &tls.Config{
				GetCertificate: localClient.GetCertificate,
			})

			logger.Info("Serving https via Tailscale", slog.String("hostname", k.hostname))

			if err := tshttpSrv.Serve(ts443); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("failed to start https server in Tailscale", slog.Any("error", err))
			}
		}()

		go func() {
			ts80, err := k.tsSrv.Listen("tcp", ":80")
			if err != nil {
				logger.Error("failed to start http server", slog.Any("error", err))
				return
			}

			logger.Info("Serving http via Tailscale", slog.String("hostname", k.hostname))

			if err := tshttpSrv.Serve(ts80); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("failed to start http server in Tailscale", slog.Any("error", err))
			}
		}()
	}

	httpSrv := &http.Server{
		Handler:     k.mux,
		ErrorLog:    k.stdLogger,
		ReadTimeout: 5 * time.Minute,
	}

	localListen, err := net.Listen("tcp", k.localAddr)
	if err != nil {
		return err
	}

	logger.Info("Serving local HTTP", slog.String("addr", k.localAddr))

	go func() {
		<-ctx.Done()
		_ = httpSrv.Shutdown(context.Background())
	}()

	if err := httpSrv.Serve(localListen); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func firstLabel(s string) string {
	s, _, _ = strings.Cut(s, ".")

	return s
}
