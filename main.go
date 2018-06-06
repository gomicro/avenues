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
	CA       string            `yaml:"ca"`
}

var (
	client *http.Client
	config configuration
)

func init() {
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
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		u, err := proxyURL(req.URL)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
			return
		}

		if req.Body != nil {
			defer req.Body.Close()
		}

		//TODO: stream instead of read all?
		reqBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		proxyReq, err := http.NewRequest(req.Method, u.String(), bytes.NewBuffer(reqBody))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		proxyReq.Header = req.Header

		resp, err := client.Do(proxyReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp.Body != nil {
			defer resp.Body.Close()
		}

		//TODO: stream instead of read all?
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
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
