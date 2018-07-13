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
		Name:   "endpoint",
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

	statefileFlag := cli.StringFlag{
		Name:  "statefile",
		Usage: "File path for storing global state, defaults to sane path based on OS",
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
			Name:   "capture-files",
			Usage:  "Captures log data from files declared in configuration and forwards to Timber's log collection endpoint",
			Action: runCaptureFiles,
			Flags: []cli.Flag{
				configFlag,
				daemonizeFlag,
				endpointFlag,
				logfileFlag,
				pidfileFlag,
				statefileFlag,
			},
		},
		{
			Name:   "capture-kube",
			Usage:  "Captures log data from Kubernetes according to configuration and forwards to configured log collection endpoint",
			Action: runCaptureKube,
			Flags: []cli.Flag{
				apiKeyFlag,
				configFlag,
				daemonizeFlag,
				logfileFlag,
				pidfileFlag,
				statefileFlag,
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

	config.Log()

	// Validate the configuration
	err = config.Validate()
	if err != nil {
		logger.Error(err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	// Once the configuration has been fetched, we build the base of the metadata that
	// will accompany every log frame sent to the collection endpoint. The metadata is
	// of the type *LogEvent.
	metadata := BuildBaseMetadata(config)

	// Start forwarding STDIN
	quit := handleSignals()
	err = ForwardStdin(config.Endpoint, config.DefaultApiKey, config.BatchPeriodSeconds, metadata, quit, config.DiscardLogsOnFatal)
	if err != nil {
		logger.Error(err)
	} else {
		logger.Info("STDIN forwarding goroutine quit")
	}

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

	// Update endpoint to one specified on command line if present
	endpoint := ctx.String("endpoint")
	if endpoint != "" {
		config.Endpoint = endpoint
	}

	// Update the configuration from a file. This *is* required for file mode.
	configFilePath := ctx.String("config")
	err := config.UpdateFromFile(configFilePath)
	if err != nil {
		logger.Errorf("Could not open config file at %s: %s", configFilePath, err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	config.Log()

	// Validate the configuration
	err = config.Validate()
	if err != nil {
		logger.Error(err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	// Initialize/Load global state
	// If we fail to initialize our global state, we exit the program. It is paramount that we are able to read
	// and record global state in order for our agent to work effectively and avoid duplicating data.
	stateFilePath := ctx.String("statefile")
	if stateFilePath == "" {
		stateFilePath = DefaultGlobalStateFilename()
	}

	err = globalState.Load(stateFilePath)
	if err != nil {
		logger.Errorf("Failed to initalize global state: %s", err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	// Once the configuration has been fetched, we build the base of the metadata that
	// will accompany every log frame sent to the collection endpoint. The metadata is
	// of the type *LogEvent.
	metadata := BuildBaseMetadata(config)

	// Each file in the configuration can contain a path with a glob pattern. So
	// we set a file path channel to receive these paths over time as they are discovered.
	fileConfigsChan := make(chan *FileConfig)
	for _, fileConfig := range config.Files {
		go func(fileConfig FileConfig) {
			err := GlobContinually(fileConfig.Path, fileConfig.ApiKey, fileConfigsChan)
			if err != nil {
				logger.Error(err)
			} else {
				logger.Infof("Globbing goroutine quit for %s", fileConfig.Path)
			}
		}(fileConfig)
	}

	// Listen for new files, tail them, and forward them
	quit := handleSignals()

	for fileConfig := range fileConfigsChan {
		logger.Infof("Received file %s, attempting to foward", fileConfig.Path)
		go func(fileConfig *FileConfig) {
			err := ForwardFile(fileConfig.Path, config.Endpoint, fileConfig.ApiKey, config.Poll, config.BatchPeriodSeconds, metadata, quit, nil, config.DiscardLogsOnFatal)
			if err != nil {
				logger.Error(err)
			} else {
				logger.Infof("Forwarding goroutine quit for %s", fileConfig.Path)
			}
		}(fileConfig)
	}

	return nil
}

// Entry point for running the agent on Kubernetes
func runCaptureKube(ctx *cli.Context) error {
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
	config.KubernetesConfig = NewKubernetesConfig()

	// Update the configuration from a file. This is not required for STDIN mode.
	configFilePath := ctx.String("config")
	err := config.UpdateFromFile(configFilePath)
	if err != nil {
		logger.Warnf("Could not open config file at %s: %s", configFilePath, err)
		logger.Info("Config file not required in Kubernetes mode")
	}

	// The API key flag take precedence if present.
	apiKey := ctx.String("api-key")
	if apiKey == "" && config.DefaultApiKey == "" {
		logger.Error("No API key. Please use --api-key, TIMBER_API_KEY, or set a default in a config file")
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	} else if apiKey == "" {
		apiKey = config.DefaultApiKey
	}

	// Set Kubernetes default configuration options
	// Overwrites any passed in file config
	if len(config.Files) > 0 {
		logger.Warn("File configurations are ignored in Kubernetes mode")
	}

	// Configure default glob path for Kubernetes application logs
	kubeFileConfig := FileConfig{ApiKey: apiKey, Path: "/var/log/containers/*"}
	config.Files = []FileConfig{kubeFileConfig}

	config.Log()

	// Validate the configuration
	err = config.Validate()
	if err != nil {
		logger.Error(err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	// Validate kubernetes configuration. This is used to diagnose misconfigurations and inform the user.
	config.KubernetesConfig.Validate()

	// Initialize/Load global state
	// If we fail to initialize our global state, we exit the program. It is paramount that we are able to read
	// and record global state in order for our agent to work effectively and avoid duplicating data.
	stateFilePath := ctx.String("statefile")
	if stateFilePath == "" {
		stateFilePath = DefaultGlobalStateFilename()
	}

	err = globalState.Load(stateFilePath)
	if err != nil {
		logger.Errorf("Failed to initalize global state: %s", err)
		// Exit with 65, EX_DATAERR, to indicate input data was incorrect
		os.Exit(65)
	}

	// Once the configuration has been fetched, we build the base of the metadata that
	// will accompany every log frame sent to the collection endpoint. The metadata is
	// of the type *LogEvent.
	metadata := BuildBaseMetadata(config)

	// Each file in the configuration can contain a path with a glob pattern. So
	// we set a file path channel to receive these paths over time as they are discovered.
	fileConfigsChan := make(chan *FileConfig)
	for _, fileConfig := range config.Files {
		go func(fileConfig FileConfig) {
			err := GlobContinually(fileConfig.Path, fileConfig.ApiKey, fileConfigsChan)
			if err != nil {
				logger.Error(err)
			} else {
				logger.Infof("Globbing goroutine quit for %s", fileConfig.Path)
			}
		}(fileConfig)
	}

	// Listen for new files, tail them, and forward them
	quit := handleSignals()

	// Initialize KubernetesClient for gathering additional Kubernetes metadata
	kubernetesClient, err := GetKubernetesClient()
	if err != nil {
		logger.Error("Unable to initialize Kubernetes client: " + err.Error())
		logger.Warn("Metadata dependent on the Kubernetes API will not be collected.")
	}

	for fileConfig := range fileConfigsChan {
		logger.Infof("Received file %s, attempting to forward", fileConfig.Path)

		go func(fileConfig *FileConfig) {
			forwardFile, stop, currentMetadata := CollectAndProcessKubernetesMetadata(kubernetesClient, config.KubernetesConfig, fileConfig.Path, metadata)
			if !forwardFile {
				return
			}

			err = ForwardFile(fileConfig.Path, config.Endpoint, fileConfig.ApiKey, config.Poll, config.BatchPeriodSeconds, currentMetadata, quit, stop, config.DiscardLogsOnFatal)
			if err != nil {
				logger.Error(err)
			} else {
				logger.Infof("Forwarding goroutine quit for %s", fileConfig.Path)
			}
		}(fileConfig)
	}

	return nil
}
