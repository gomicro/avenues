package config

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	defaultStatusEndpoint = "/avenues/status"
	defaultConfigFile     = "./routes.yaml"

	configFileEnv = "AVENUES_CONFIG_FILE"
)

// File represents all the configurable options of Avenues
type File struct {
	Services map[string]string `yaml:"services"`
	Routes   map[string]string `yaml:"routes"`
	Status   string            `yaml:"status"`
	CA       string            `yaml:"ca"`
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

	return &conf, nil
}

func (f *File) ServiceURL(reqURL *url.URL) (*url.URL, error) {
	serviceName, ok := f.pathToServiceName(reqURL.Path)
	if !ok {
		serviceName = "default"
		_, ok := f.Services["default"]
		if !ok {
			return nil, fmt.Errorf("service name not found for url: %v", reqURL.Path)
		}
	}

	serviceAddress, ok := f.Services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service address not found for name: %v", serviceName)
	}

	u, err := url.Parse(serviceAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service address")
	}

	u.Path = reqURL.Path
	u.RawQuery = reqURL.Query().Encode()

	return u, nil
}

func (f *File) pathToServiceName(path string) (string, bool) {
	for route, serviceName := range f.Routes {
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
