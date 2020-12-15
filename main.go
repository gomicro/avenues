package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"

	"github.com/gomicro/avenues/config"
	log "github.com/gomicro/ledger"
)

var (
	conf    *config.File
	version string
)

func configure() {
	if version == "" {
		version = "dev-local"
	}
	log.Infof("Avenues %v", version)

	c, err := config.ParseFromFile()
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err.Error())
		os.Exit(1)
	}

	conf = c
	log.Debug("Config file parsed")

	log.Debug("Configuration complete")
}

func main() {
	configure()

	log.Infof("Listening on %v:%v", "0.0.0.0", "4567")

	http.Handle("/", conf)

	if conf.Key != "" && conf.Cert != "" {
		log.Info("Serving with SSL")

		cert, err := tls.X509KeyPair([]byte(conf.Cert), []byte(conf.Key))
		if err != nil {
			log.Fatalf("failed to create ssl cert/key pair: %v", err.Error())
			os.Exit(1)
		}

		cfg := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			Certificates: []tls.Certificate{cert},
		}

		srv := &http.Server{
			Addr:      net.JoinHostPort("0.0.0.0", "4567"),
			TLSConfig: cfg,
		}

		err = srv.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatalf("something went horribly wrong: %v", err.Error())
			os.Exit(1)
		}
	} else {
		log.Info("Serving without SSL")
		err := http.ListenAndServe("0.0.0.0:4567", nil)
		if err != nil {
			log.Fatalf("something went horribly wrong: %v", err.Error())
			os.Exit(1)
		}
	}
}
