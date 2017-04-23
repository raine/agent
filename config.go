package main

import "github.com/BurntSushi/toml"

type fileConfig struct {
	Path   string
	ApiKey string
}

type Config struct {
	Files              []fileConfig
	Endpoint           string
	BatchPeriodSeconds int64
	Poll               bool
}

func readConfig(file string) (*Config, error) {
	var config Config

	if _, err := toml.DecodeFile(file, &config); err != nil {
		return nil, err
	}

	if config.BatchPeriodSeconds == 0 {
		config.BatchPeriodSeconds = 10
	}

	if config.Endpoint == "" {
		config.Endpoint = "https://logs.timber.io/frames"
	}

	return &config, nil
}
