package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kradalby/kra/html"
	"github.com/kradalby/kra/util"
	"github.com/kradalby/kra/web"
)

const (
	defaultHostname = "krapage"
)

var (
	tailscaleKeyPath = flag.String(
		"ts-key-path",
		util.GetEnvString("KRAPAGE_TS_KEY_PATH", ""),
		"Path to tailscale auth key",
	)

	hostname = flag.String(
		"ts-hostname",
		util.GetEnvString("KRAPAGE_TS_HOSTNAME", defaultHostname),
		"",
	)

	controlURL = flag.String(
		"ts-controlurl",
		util.GetEnvString("KRAPAGE_TS_CONTROL_SERVER", ""),
		"Tailscale Control server, if empty, upstream",
	)

	verbose = flag.Bool("verbose", util.GetEnvBool("KRAPAGE_VERBOSE", false), "be verbose")

	localAddr = flag.String(
		"listen-addr",
		util.GetEnvString("KRAPAGE_LISTEN_ADDR", "localhost:56661"),
		"Local address to listen to",
	)

	dev = flag.Bool(
		"dev",
		util.GetEnvBool("KRAPAGE_DEV", false),
		"disable tailscale",
	)
)

func handler(page string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(page))
	})
}

func main() {
	flag.Parse()

	slogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	stdLogger := slog.NewLogLogger(slogger.Handler(), slog.LevelError)

	k, err := web.NewServer(
		web.ServerConfig{
			Hostname:        *hostname,
			LocalAddr:       *localAddr,
			AuthKeyPath:     *tailscaleKeyPath,
			EnableTailscale: !*dev,
		},
		web.WithVerbose(*verbose),
		web.WithControlURL(*controlURL),
		web.WithStdLogger(stdLogger),
		web.WithLogger(slogger),
	)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	k.Handle("/", handler(html.Home().Render()))
	k.Handle("/about", handler(html.About().Render()))

	salaryPage, err := html.Salary()
	if err != nil {
		log.Fatalf("failed to render salary page: %v", err)
	}
	k.Handle("/salary", handler(salaryPage.Render()))

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := k.ListenAndServe(ctx); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
