package web

import (
	"crypto/tls"
	"fmt"
	"html"
	"log"
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
	localAddr string

	controlURL string
	verbose    bool
	logger     *log.Logger
	noTS       bool

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

func WithLogger(l *log.Logger) Option {
	return func(kw *KraWeb) {
		kw.logger = l
	}
}

func WithTailscale(b bool) Option {
	return func(kw *KraWeb) {
		kw.noTS = b
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
		logger:     nil,
		noTS:       false,
	}

	for _, opt := range opts {
		opt(k)
	}

	k.mux = http.NewServeMux()
	k.tsmux = http.NewServeMux()

	debugHandler := tsweb.Debugger(k.tsmux)
	k.debugHandler = debugHandler

	err := statsviz.Register(k.tsmux)
	if err == nil {
		k.debugHandler.URL("/debug/statsviz", "Statsviz (visualise go metrics)")
	} else {
		log.Printf("failed to register statsviz: %s", err)
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

func (k *KraWeb) ListenAndServe() error {
	log := k.logger

	if k.tsKeyPath != "" {
		key, err := os.ReadFile(k.tsKeyPath)
		if err != nil {
			return err
		}

		k.tsSrv.AuthKey = strings.TrimSuffix(string(key), "\n")
	}

	if k.verbose {
		k.tsSrv.Logf = log.Printf
	}

	if k.noTS {
		if err := k.tsSrv.Start(); err != nil {
			return err
		}

		localClient, _ := k.tsSrv.LocalClient()

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
			ErrorLog:    k.logger,
			ReadTimeout: 5 * time.Minute,
		}

		// Starting HTTPS server
		go func() {
			ts443, err := k.tsSrv.Listen("tcp", ":443")
			if err != nil {
				log.Printf("failed to start https server: %s", err)
			}

			ts443 = tls.NewListener(ts443, &tls.Config{
				GetCertificate: localClient.GetCertificate,
			})

			log.Printf("Serving https://%s/ ...", k.hostname)

			if err := tshttpSrv.Serve(ts443); err != nil {
				log.Fatalf("failed to start https server in Tailscale: %s", err)
			}
		}()

		go func() {
			ts80, err := k.tsSrv.Listen("tcp", ":80")
			if err != nil {
				log.Printf("failed to start http server: %s", err)
			}

			log.Printf("Serving http://%s/ ...", k.hostname)

			if err := tshttpSrv.Serve(ts80); err != nil {
				log.Fatalf("failed to start http server in Tailscale: %s", err)
			}
		}()
	}

	httpSrv := &http.Server{
		Handler:     k.mux,
		ErrorLog:    k.logger,
		ReadTimeout: 5 * time.Minute,
	}

	localListen, err := net.Listen("tcp", k.localAddr)
	if err != nil {
		return err
	}

	log.Printf("Serving http://%s/ ...", k.localAddr)

	if err := httpSrv.Serve(localListen); err != nil {
		return err
	}

	return nil
}

func firstLabel(s string) string {
	s, _, _ = strings.Cut(s, ".")

	return s
}
