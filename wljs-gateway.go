package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/koding/websocketproxy"
)

var transport = &http.Transport{
	ResponseHeaderTimeout: 360 * time.Second,
	IdleConnTimeout:       90 * time.Second,
}

func httpReverseProxy(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(u)
			r.SetXForwarded()

			r.Out.Host = u.Host

			// must set keep-alive otherwise wljs home page won't load
			r.Out.Header.Set("Connection", "keep-alive, upgrade")
		},

		ModifyResponse: func(resp *http.Response) error {
			return nil
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Println("proxy error:", err)
			http.Error(w, "bad gateway", http.StatusBadGateway)
		},
	}

	return proxy
}

type Config struct {
	Host       string
	HTTPPort   int
	WSPort     int
	WS2Port    int
	ListenPort int
	WSPrefix   string
	WS2Prefix  string
}

func (c *Config) HTTPUpstream() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.HTTPPort)
}

func (c *Config) wsURL(port int) *url.URL {
	u, err := url.Parse(fmt.Sprintf("ws://%s:%d", c.Host, port))
	if err != nil {
		log.Fatal(err)
	}
	return u
}

func (c *Config) WSUpstream() *url.URL  { return c.wsURL(c.WSPort) }
func (c *Config) WS2Upstream() *url.URL { return c.wsURL(c.WS2Port) }

func (c *Config) ListenAddr() string {
	return fmt.Sprintf(":%d", c.ListenPort)
}

func wsPath(prefix string) string {
	if len(prefix) > 0 && prefix[0] == '/' {
		return prefix
	}
	return "/" + prefix
}

func (c *Config) WSPath() string  { return wsPath(c.WSPrefix) }
func (c *Config) WS2Path() string { return wsPath(c.WS2Prefix) }

func ParseConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Host, "host", "127.0.0.1", "upstream host")
	flag.IntVar(&cfg.HTTPPort, "http", 4000, "http upstream port")
	flag.IntVar(&cfg.WSPort, "ws", 4001, "websocket upstream port")
	flag.IntVar(&cfg.WS2Port, "ws2", 4002, "websocket2 upstream port")
	flag.IntVar(&cfg.ListenPort, "port", 3000, "listen port")
	flag.StringVar(&cfg.WSPrefix, "wsprefix", "ws", "websocket path prefix (without leading slash)")
	flag.StringVar(&cfg.WS2Prefix, "ws2prefix", "ws2", "websocket2 path prefix (without leading slash)")

	flag.Parse()
	return cfg
}

func main() {
	cfg := ParseConfig()

	websocketproxy.DefaultUpgrader.CheckOrigin = func(*http.Request) bool {
		return true
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.WSPath(), websocketproxy.NewProxy(cfg.WSUpstream()))
	mux.Handle(cfg.WS2Path(), websocketproxy.NewProxy(cfg.WS2Upstream()))
	mux.Handle("/", httpReverseProxy(cfg.HTTPUpstream()))

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	log.Println("listening on", cfg.ListenAddr())
	log.Fatal(server.ListenAndServe())
}
