package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	log "github.com/gomicro/ledger"
	"gopkg.in/yaml.v2"
)

const (
	defaultStatusEndpoint = "/avenues/status"
	defaultResetEndpoint  = "/avenues/reset"
	defaultConfigFile     = "./routes.yaml"

	configFileEnv = "AVENUES_CONFIG_FILE"
)

// File represents all the configurable options of Avenues
type File struct {
	Routes    map[string]*Route                 `yaml:"routes"`
	Reset     string                            `yaml:"reset"`
	Status    string                            `yaml:"status"`
	Cert      string                            `yaml:"cert"`
	CertPath  string                            `yaml:"cert_path"`
	Key       string                            `yaml:"key"`
	KeyPath   string                            `yaml:"key_path"`
	CA        string                            `yaml:"ca"`
	CAPath    string                            `yaml:"ca_path"`
	proxies   map[string]*httputil.ReverseProxy `yaml:"-"`
	transport *http.Transport                   `yaml:"-"`
}

// ParseFromFile reads an Avenues config file from the file specified in the
// environment or from the default file location if no environment is specified.
// A File with the populated values is returned and any errors encountered while
// trying to read the file.
func ParseFromFile() (*File, error) {
	configFile := os.Getenv(configFileEnv)

	if configFile == "" {
		configFile = defaultConfigFile
	}

	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %v", err.Error())
	}

	var conf File
	err = yaml.Unmarshal(b, &conf)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal config file: %v", err.Error())
	}

	if conf.Status == "" {
		conf.Status = defaultStatusEndpoint
	}

	if conf.Reset == "" {
		conf.Reset = defaultResetEndpoint
	}

	conf.proxies = make(map[string]*httputil.ReverseProxy)

	if conf.CAPath != "" {
		ca, err := ioutil.ReadFile(conf.CAPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read CA(s) from file")
		}
		conf.CA = string(ca)
	}

	pool := x509.NewCertPool()
	if conf.CA != "" {
		ok := pool.AppendCertsFromPEM([]byte(conf.CA))
		if !ok {
			return nil, fmt.Errorf("Failed to append CA(s) to cert pool")
		}
	}

	conf.transport = &http.Transport{
		MaxIdleConnsPerHost: 50,
		MaxIdleConns:        50,
		TLSClientConfig:     &tls.Config{RootCAs: pool},
	}

	return &conf, nil
}

func (f *File) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == "OPTIONS" {
		log.Info("responding with cors headers for options request")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*, Authorization")
		w.Header().Set("Access-Control-Max-Age", "60")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusNoContent)

		return
	}

	switch req.URL.Path {
	case f.Status:
		handleStatus(w, req)
		return
	case f.Reset:
		f.handleReset(w, req)
		return
	}

	u, err := f.backingURL(req.URL)
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
		Transport: f.transport,
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Set("Access-Control-Allow-Origin", "*")
			resp.Header.Set("Access-Control-Allow-Methods", "*")
			resp.Header.Set("Access-Control-Allow-Headers", "*, Authorization")
			resp.Header.Set("Access-Control-Max-Age", "60")
			resp.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
			resp.Header.Set("Vary", "Accept-Encoding")

			return nil
		},
	}

	rp.ServeHTTP(w, req)
	log.Infof("proxyed '%v' to '%v'", req.URL, u.String())
}

func (f *File) handleReset(w http.ResponseWriter, req *http.Request) {
	for _, r := range f.Routes {
		r.reset()
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("routes have been reset"))
}

func handleStatus(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("avenues is functioning"))
}

func (f *File) backingURL(reqURL *url.URL) (*url.URL, error) {
	route, ok := f.pathToRoute(reqURL.Path)
	if !ok {
		return nil, fmt.Errorf("route not found for url: %v", reqURL.Path)
	}

	var u *url.URL
	var err error

	switch strings.ToLower(route.Type) {
	case ordinalRouteType:
		i := route.index

		if route.Backends == nil {
			return nil, fmt.Errorf("ordinal route requires backends directive")
		}

		u, err = url.Parse(route.Backends[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse service address: %v", err.Error())
		}

		if i < len(route.Backends)-1 {
			route.index++
		}
	case staticRouteType, "":
		u, err = url.Parse(route.Backend)
		if err != nil {
			return nil, fmt.Errorf("failed to parse service address: %v", err.Error())
		}
	}

	u.Path = reqURL.Path
	u.RawQuery = reqURL.Query().Encode()

	return u, nil
}

func (f *File) pathToRoute(path string) (*Route, bool) {
	for prefix, route := range f.Routes {
		if strings.HasPrefix(trailingSlash(path), prefix) {
			return route, true
		}
	}

	return nil, false
}

func trailingSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		return fmt.Sprintf("%v/", path)
	}

	return path
}

const (
	ordinalRouteType = "ordinal"
	staticRouteType  = "static"
)

// Route represents a backing route to direct a request to
type Route struct {
	Type     string   `yaml:"type"`
	Backend  string   `yaml:"backend,omitempty"`
	index    int      `yaml:"-"`
	Backends []string `yaml:"backends,omitempty"`
}

func (r *Route) reset() {
	r.index = 0
}
