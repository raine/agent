package main

import (
	"io"

	"github.com/BurntSushi/toml"
	"gopkg.in/urfave/cli.v1"
)

type fileConfig struct {
	Path   string
	ApiKey string `toml:"api_key"`
}

type Config struct {
	Files                      []fileConfig
	Endpoint                   string
	BatchPeriodSeconds         int64
	Poll                       bool
	Hostname                   string
	CollectEC2MetadataDisabled bool `toml:"disable_ec2_metadata"`
}

func readConfig(in io.Reader) (*Config, error) {
	var config Config

	if _, err := toml.DecodeReader(in, &config); err != nil {
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

func validateConfig(config *Config, ctx *cli.Context) error {
	if ctx.IsSet("stdin") {
		if !ctx.IsSet("api-key") {
			return cli.NewExitError("--stdin requires --api-key or TIMBER_API_KEY set", 1)
		}
	} else {
		if ctx.IsSet("api-key") {
			return cli.NewExitError("--api-key is only for use with --stdin", 1)
		}
	}
	return nil
}
