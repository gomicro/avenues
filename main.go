package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	log "github.com/gomicro/ledger"

	"gopkg.in/yaml.v2"
)

const (
	defaultConfigFile = "./routes.yaml"
	configFileEnv     = "AVENUES_CONFIG_FILE"
)

type configuration struct {
	Services map[string]string `yaml:"services"`
	Routes   map[string]string `yaml:"routes"`
	Status   string            `yaml:"status"`
	CA       string            `yaml:"ca"`
}

var (
	client *http.Client
	config configuration
)

func configure() {
	readConfigFile()

	pool := x509.NewCertPool()

	if config.CA != "" {
		ok := pool.AppendCertsFromPEM([]byte(config.CA))
		if !ok {
			log.Fatal("Failed to append CA(s) to cert pool")
			os.Exit(1)
		}
	}

	client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 50,
			MaxIdleConns:        50,
			TLSClientConfig:     &tls.Config{RootCAs: pool},
		},
	}
}

func readConfigFile() {
	configFile := os.Getenv(configFileEnv)

	if configFile == "" {
		configFile = defaultConfigFile
	}

	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Errorf("Failed to read config file: %v", err.Error())
		os.Exit(1)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Errorf("Failed to unmarshal config file: %v", err.Error())
		os.Exit(1)
	}

	if config.Status == "" {
		config.Status = "/avenues/status"
	}
}

func main() {
	configure()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.Header().Set("Access-Control-Max-Age", "60")

			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")

			w.Header().Set("Vary", "Accept-Encoding")

			return
		}

		u, err := proxyURL(req.URL)
		if err != nil {
			log.Warnf("failed to proxy url: %v", err.Error())
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if req.Body != nil {
			defer req.Body.Close()
		}

		reqBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Errorf("failed to read request body: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		proxyReq, err := http.NewRequest(req.Method, u.String(), bytes.NewBuffer(reqBody))
		if err != nil {
			log.Errorf("failed to create proxy request: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		proxyReq.Header = req.Header

		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Errorf("failed to do proxy request: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp.Body != nil {
			defer resp.Body.Close()
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to read response body: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for k, v := range resp.Header {
			w.Header().Set(k, strings.Join(v, ""))
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
		w.Header().Set("Vary", "Accept-Encoding")

		w.WriteHeader(resp.StatusCode)
		w.Write(b)

		log.Infof("proxyed '%v' to '%v'", req.URL, u.String())
	})

	mux.HandleFunc(config.Status, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("avenues is functioning"))
	})

	log.Infof("Listening on %v:%v", "0.0.0.0", "4567")
	http.ListenAndServe("0.0.0.0:4567", mux)
}

func proxyURL(reqURL *url.URL) (*url.URL, error) {
	serviceName, ok := pathToServiceName(reqURL.Path)
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	serviceAddress, ok := config.Services[serviceName]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	u, err := url.Parse(serviceAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service address")
	}

	u.Path = reqURL.Path
	u.RawQuery = reqURL.Query().Encode()

	return u, nil
}

func pathToServiceName(path string) (string, bool) {
	for route, serviceName := range config.Routes {
		if strings.HasPrefix(trailingSlash(path), route) {
			return serviceName, true
		}
	}

	return "", false
}

func trailingSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		return fmt.Sprintf("%v/", path)
	}

	return path
}
