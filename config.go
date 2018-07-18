package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type FileConfig struct {
	Path   string
	ApiKey string `toml:"api_key"`
}

type Config struct {
	DefaultApiKey              string `toml:"default_api_key"`
	Files                      []FileConfig
	Endpoint                   string
	BatchPeriodSeconds         int64
	Poll                       bool
	Hostname                   string
	CollectEC2MetadataDisabled bool              `toml:"disable_ec2_metadata"`
	KubernetesConfig           *KubernetesConfig `toml:"kubernetes"`
	ReadNewFileFromStart       bool              `toml:"read_from_start"`
}

type KubernetesConfig struct {
	Exclude map[string]string
}

func (c *Config) Log() {
	logger.Infof("Log collection endpoint: %s", c.Endpoint)
	logger.Infof("Using filesystem polling: %s", c.Poll)
	logger.Infof("Maximum time between sends: %d seconds", c.BatchPeriodSeconds)
	logger.Infof("File count: %d", len(c.Files))

	for i, file := range c.Files {
		apiKeySample := file.ApiKey[len(file.ApiKey)-4:]
		logger.Infof("File %d: %s (api key: ...%s)", i+1, file.Path, apiKeySample)
	}
}

func (c *Config) UpdateFromFile(filePath string) error {
	configFile, err := os.Open(filePath)
	if err != nil {
		logger.Errorf("Could not open config file at %s: %s", filePath, err)
		return err
	}

	logger.Infof("Opened configuration file at %s", filePath)

	return c.UpdateFromReader(configFile)
}

func (c *Config) UpdateFromReader(in io.Reader) error {
	_, err := toml.DecodeReader(in, c)
	if err != nil {
		return err
	}

	// If a file does not define its own API key, the default API key
	// is used
	for i := range c.Files {
		if c.Files[i].ApiKey == "" {
			c.Files[i].ApiKey = c.DefaultApiKey
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if len(c.Files) > 0 {
		for _, f := range c.Files {
			if f.ApiKey == "" {
				errText := fmt.Sprintf("File %s has no API key", f.Path)
				return errors.New(errText)
			}
		}
	} else {
		if c.DefaultApiKey == "" {
			errText := "No API key. Please use --api-key, TIMBER_API_KEY, or set a default in a config file"
			return errors.New(errText)
		}
	}

	return nil
}

func NewConfig() *Config {
	return &Config{
		BatchPeriodSeconds: 3,
		Endpoint:           "https://logs.timber.io/frames",
	}
}

//NewKubernetesConfig Return a new KubernetesConfig initialized with opinionated defaults.
// These include filtering kube-system namespace and timber-agent pods, our expected Pod name.
func NewKubernetesConfig() *KubernetesConfig {
	return &KubernetesConfig{
		Exclude: map[string]string{
			"namespaces": "kube-system",
			"pods":       "timber-agent",
		},
	}
}

var supportedFilterKinds = []string{"namespaces", "deployments", "pods"}

//Validate Serves as a source of diagnostic information for the end user
func (kc *KubernetesConfig) Validate() {
	// Validate Exclude configuration
	for kind := range kc.Exclude {
		var match bool

		for _, filterKind := range supportedFilterKinds {
			if kind == filterKind {
				match = true
				break
			}
		}

		if !match {
			logger.Warnf("Exclusion kind %s is not supported and will not be applied as a filter.", kind)
		}
	}
}

func (kc *KubernetesConfig) ApplyFilter(context *KubernetesContext) (string, bool) {
	if len(kc.Exclude) == 0 {
		return "", false
	}

	// Namespaces
	if filterString, ok := kc.Exclude["namespaces"]; ok {
		match, ok := compareNamespace(filterString, context.Namespace)
		if ok {
			match = fmt.Sprintf("%s:%s", "namespaces", match)
			return match, ok
		}
	}

	// Deployments
	if filterString, ok := kc.Exclude["deployments"]; ok {
		match, ok := compareRootOwner("deployment", filterString, context.RootOwner)
		if ok {
			match = fmt.Sprintf("%s:%s", "deployments", match)
			return match, ok
		}
	}

	// Pods
	if filterString, ok := kc.Exclude["pods"]; ok {
		match, ok := comparePod(filterString, context.PodName)
		if ok {
			match = fmt.Sprintf("%s:%s", "pods", match)
			return match, ok
		}
	}

	return "", false
}

func compareNamespace(filterString, value string) (string, bool) {
	patterns := strings.Split(filterString, ",")

	for _, pattern := range patterns {
		r, err := regexp.Compile(pattern)
		if err != nil {
			logger.Errorf("Unable to parse invalid regular expression: %s", pattern)
			continue // try again with next pattern
		}

		if r.MatchString(value) {
			return pattern, true
		}
	}

	return "", false
}
func comparePod(filterString, value string) (string, bool) {
	patterns := strings.Split(filterString, ",")

	for _, pattern := range patterns {
		r, err := regexp.Compile(pattern)
		if err != nil {
			logger.Errorf("Unable to parse invalid regular expression: %s", pattern)
			continue // try again with next pattern
		}

		if r.MatchString(value) {
			return pattern, true
		}
	}

	return "", false
}

// rootOwner has fields kind and name
func compareRootOwner(kind, filterString string, rootOwner map[string]string) (string, bool) {
	if strings.ToLower(rootOwner["kind"]) != kind {
		return "", false
	}

	patterns := strings.Split(filterString, ",")

	for _, pattern := range patterns {
		r, err := regexp.Compile(pattern)
		if err != nil {
			logger.Errorf("Unable to parse invalid regular expression: %s", pattern)
			continue // try again with next pattern
		}

		if r.MatchString(rootOwner["name"]) {
			return pattern, true
		}
	}

	return "", false
}
