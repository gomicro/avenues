package main

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/gomicro/avenues/config"
	log "github.com/gomicro/ledger"
)

var (
	transport *http.Transport
	conf      *config.File
	proxies   map[string]*httputil.ReverseProxy
)

func configure() {
	c, err := config.ParseFromFile()
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err.Error())
		os.Exit(1)
	}

	conf = c
	log.Debug("Config file parsed")

	pool := x509.NewCertPool()
	if conf.CA != "" {
		ok := pool.AppendCertsFromPEM([]byte(conf.CA))
		if !ok {
			log.Fatal("Failed to append CA(s) to cert pool")
			os.Exit(1)
		}
		log.Debug("Custom CA configured")
	}
	log.Debug("CA configured")

	transport = &http.Transport{
		MaxIdleConnsPerHost: 50,
		MaxIdleConns:        50,
		TLSClientConfig:     &tls.Config{RootCAs: pool},
	}

	log.Debug("HTTP transport configured")

	proxies = make(map[string]*httputil.ReverseProxy)

	log.Debug("Configuration complete")
}

func main() {
	configure()

	log.Infof("Listening on %v:%v", "0.0.0.0", "4567")
	http.ListenAndServe("0.0.0.0:4567", newProxy())
}

type proxy struct {
}

func newProxy() *proxy {
	return &proxy{}
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "OPTIONS" {
		log.Info("responding with cors headers for options request")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "60")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusNoContent)

		return
	}

	if req.URL.Path == conf.Status {
		handleStatus(w, req)
		return
	}

	u, err := conf.ServiceURL(req.URL)
	if err != nil {
		log.Warnf("failed to proxy url: %v", err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rp := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", u.Host)

			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			req.URL.Path = u.Path
		},
		Transport: transport,
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Set("Access-Control-Allow-Origin", "*")
			resp.Header.Set("Access-Control-Allow-Methods", "*")
			resp.Header.Set("Access-Control-Allow-Headers", "*")
			resp.Header.Set("Access-Control-Max-Age", "60")
			resp.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
			resp.Header.Set("Vary", "Accept-Encoding")

			return nil
		},
	}

	rp.ServeHTTP(w, req)
	log.Infof("proxyed '%v' to '%v'", req.URL, u.String())
}

func handleStatus(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("avenues is functioning"))
}
