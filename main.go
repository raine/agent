package main

import (
	"os"

	"gopkg.in/urfave/cli.v1"
)

var version string

func main() {
	apiKeyFlag := cli.StringFlag{
		Name:   "api-key",
		Usage:  "timber API key to use when capturing stdin",
		EnvVar: "TIMBER_API_KEY",
	}

	configFlag := cli.StringFlag{
		Name:  "config, c",
		Usage: "config file to use, for available options see https://timber.io/docs/platforms/other/agent/configuration-file",
		Value: "/etc/timber.toml",
	}

	daemonizeFlag := cli.BoolFlag{
		Name:  "daemonize",
		Usage: "starts an instance of agent as a daemon (only available on Linux; see documentation)",
	}

	endpointFlag := cli.StringFlag{
		Name:   "Endpoint",
		Usage:  "Configures the log collection endpoint logs are sent to",
		Hidden: true,
	}

	logfileFlag := cli.StringFlag{
		Name:  "output-log-file",
		Usage: "the agent will write its own logs to `FILE` (will use STDOUT if not provided)",
	}

	pidfileFlag := cli.StringFlag{
		Name:  "pidfile",
		Usage: "will store the pid in `FILE` when set",
	}

	app := cli.NewApp()
	app.Name = "timber-agent"
	app.Usage = "forwards logs to timber.io"
	app.Version = version
	app.Commands = []cli.Command{
		{
			Name:   "capture-stdin",
			Usage:  "Captures log data sent over STDIN and forwards to Timber's log collection endpoint",
			Action: runCaptureStdin,
			Flags: []cli.Flag{
				apiKeyFlag,
				configFlag,
				endpointFlag,
				logfileFlag,
				pidfileFlag,
			},
		},
		{
			Name:        "capture-files",
			Description: "Captures log data from files declared in configuration and forwards to Timber's log collection endpoint",
			Action:      runCaptureFiles,
			Flags: []cli.Flag{
				configFlag,
				daemonizeFlag,
				logfileFlag,
				pidfileFlag,
			},
		},
	}

	app.Run(os.Args)
}

// Entry point for running the agent over STDIN
func runCaptureStdin(ctx *cli.Context) error {
	// Setup the logger first so that any debug output can be made to the user.
	logfilePath := ctx.String("output-log-file")
	if logfilePath != "" {
		logFile, err := setLoggerOutputFile(logfilePath)
		if err != nil {
			// Exit with 65, EX_DATAERR, to indicate input data was incorrect
			os.Exit(65)
		}
		defer logFile.Close()
	}

	logger.Info("Timber agent starting")

	// Handle the PID file. If it exists, exit. If it does not, write it.
	pidfilePath := ctx.String("pidfile")
	if pidfilePath != "" {
		if err := writePIDFile(pidfilePath); err != nil {
			// Exit with 65, EX_DATAERR, to indicate input data was incorrect
			os.Exit(65)
		}
		defer removePIDFile(pidfilePath)
	}

	// Load the config with defaults.
	config := NewConfig()

	// Update the configuration from a file. This is not required for STDIN mode.
	configFilePath := ctx.String("config")
	err := config.UpdateFromFile(configFilePath)
	if err != nil {
		logger.Warnf("Could not open config file at %s: %s", configFilePath, err)
		logger.Infof("Config file not required in STDIN mode")
	}

	// The API key flag take precedence if present.
	apiKey := ctx.String("api-key")
	if apiKey != "" {
		config.DefaultApiKey = apiKey
	}

	// Validate the configuration
	err = config.Validate()
	if err != nil {
		logger.Error(err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	config.Log()

	// Once the configuration has been fetched, we build the base of the metadata that
	// will accompany every log frame sent to the collection endpoint. The metadata is
	// of the type *LogEvent.
	metadata := BuildBaseMetadata(config)

	// Start forwarding STDIN
	quit := handleSignals()
	ForwardStdin(config.Endpoint, config.DefaultApiKey, config.BatchPeriodSeconds, metadata, quit)

	return nil
}

// Entry point for running the agent to tail files
func runCaptureFiles(ctx *cli.Context) error {
	// Setup the logger first so that any debug output can be made to the user.
	logfilePath := ctx.String("output-log-file")
	if logfilePath != "" {
		logFile, err := setLoggerOutputFile(logfilePath)
		if err != nil {
			// Exit with 65, EX_DATAERR, to indicate input data was incorrect
			os.Exit(65)
		}
		defer logFile.Close()
	}

	logger.Info("Timber agent starting")

	// If the user has set the `--daemonize` flag, then we call the
	// Daemonize() function. The function is defined by either
	// daemon.go or daemon_linux.go depending on the build platform.
	// Daemonization is only possible on Linux, see daemon.go for
	// a full discussion on this
	if ctx.Bool("daemonize") {
		if err := Daemonize(); err != nil {
			return err
		}
	}

	// Handle the PID file. If it exists, exit. If it does not, write it.
	pidfilePath := ctx.String("pidfile")
	if pidfilePath != "" {
		if err := writePIDFile(pidfilePath); err != nil {
			// Exit with 65, EX_DATAERR, to indicate input data was incorrect
			os.Exit(65)
		}
		defer removePIDFile(pidfilePath)
	}

	// Load the config with defaults.
	config := NewConfig()

	// Update the configuration from a file. This *is* required for file mode.
	configFilePath := ctx.String("config")
	err := config.UpdateFromFile(configFilePath)
	if err != nil {
		logger.Errorf("Could not open config file at %s: %s", configFilePath, err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	// Validate the configuration
	err = config.Validate()
	if err != nil {
		logger.Error(err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	config.Log()

	// Once the configuration has been fetched, we build the base of the metadata that
	// will accompany every log frame sent to the collection endpoint. The metadata is
	// of the type *LogEvent.
	metadata := BuildBaseMetadata(config)

	// Each file in the configuration can contain a path with a glob pattern. So
	// we set a file path channel to receive these paths over time as they are discovered.
	fileConfigsChan := make(chan *FileConfig)
	for _, fileConfig := range config.Files {
		go Glob(fileConfigsChan, &fileConfig)
	}

	// Listen for new files, tail them, and forward them
	quit := handleSignals()

	for fileConfig := range fileConfigsChan {
		go ForwardFile(fileConfig.Path, config.Endpoint, fileConfig.ApiKey, config.Poll, config.BatchPeriodSeconds, metadata, quit)
	}

	return nil
}
