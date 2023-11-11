package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"

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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(page))
	})
}

//go:embed all:static
var staticAssets embed.FS

func main() {
	flag.Parse()

	logger := log.New(os.Stdout, "hvor: ", log.LstdFlags)

	k := web.NewKraWeb(
		*hostname,
		*tailscaleKeyPath,
		*localAddr,
		web.WithVerbose(*verbose),
		web.WithControlURL(*controlURL),
		web.WithLogger(logger),
		web.WithTailscale(!*dev),
	)

	staticFS := http.FS(staticAssets)
	fs := http.FileServer(staticFS)
	k.Handle("/static/", fs)

	k.Handle("/", handler(html.Home().Render()))
	k.Handle("/about", handler(html.About().Render()))

	log.Fatalf("Failed to serve %s", k.ListenAndServe())
}
