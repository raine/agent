package main

import (
	"errors"
	"fmt"
	"io"
	"os"

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
	CollectEC2MetadataDisabled bool `toml:"disable_ec2_metadata"`
}

func (c *Config) Log() {
	logger.Infof("Log Collection Endpoint: %s", c.Endpoint)
	logger.Infof("Using filesystem polling: %s", c.Poll)
	logger.Infof("Maximum time between sends: %d seconds", c.BatchPeriodSeconds)
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
		BatchPeriodSeconds: 10,
		Endpoint:           "https://logs.timber.io/frames",
	}
}
