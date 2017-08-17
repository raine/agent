package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
)

type fileConfig struct {
	Path   string
	ApiKey string `toml:"api_key"`
}

type Config struct {
	DefaultApiKey              string `toml:"default_api_key"`
	Files                      []fileConfig
	Endpoint                   string
	BatchPeriodSeconds         int64
	Poll                       bool
	Hostname                   string
	CollectEC2MetadataDisabled bool `toml:"disable_ec2_metadata"`
}

// parseConfig takes an io.Reader which should contain TOML formatted data.
// An error will be returned if the data is not valid TOML
func parseConfig(in io.Reader) (*Config, error) {
	var config Config

	if _, err := toml.DecodeReader(in, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// normalizeConfig takes a Config pointer and normalizes it for use by setting
// zero values to sensible defaults
//
// normalizeConfig should be called after parseConfig to produce a usable
// Config struct. Even with a zero-value Config struct, the normalization
// will make it usable.
func normalizeConfig(config *Config) {
	if config.BatchPeriodSeconds == 0 {
		config.BatchPeriodSeconds = 10
	}

	if config.Endpoint == "" {
		config.Endpoint = "https://logs.timber.io/frames"
	}

	// If a file does not define its own API key, the default API key
	// is used
	for i := range config.Files {
		if config.Files[i].ApiKey == "" {
			config.Files[i].ApiKey = config.DefaultApiKey
		}
	}

	return
}

func validateConfigFiles(config *Config) error {
	for _, f := range config.Files {
		if f.ApiKey == "" {
			errText := fmt.Sprintf("File %s has no API key", f.Path)
			return errors.New(errText)
		}
	}
	return nil
}

func validateConfigStdin(config *Config) error {
	if config.DefaultApiKey == "" {
		errText := "No API key. Please use --api-key, TIMBER_API_KEY, or set a default in a config file"
		return errors.New(errText)
	}

	return nil
}
