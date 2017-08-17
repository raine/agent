package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"gopkg.in/urfave/cli.v1"
)

var version string

// Amount of time other go routines have to exit after an
// OS exit signal is received from the OS before the agent
// forcibly exits
const signalHandlingTimeout = 5 * time.Second

// source reperesents a single source to pull log lines from, regardless of
// whether it is a file or STDIN. This is passed as input to commonCaptureRun
// which coordinates sending the lines to the server.
type source struct {
	ApiKey   string // The API key to use with this log source
	Metadata string // A LogEvent coerced to stringified JSON passed in the header of HTTP requests
	Tailer   Tailer // An implementation of the Tailer interface which provides the source's lines
}

// commonCaptureRunContext represents a configuration struct for details that are the
// same regardless of whether capturing data from files or STDIN.
type commonCaptureRunContext struct {
	Config   *Config     // Config struct (pointer) representing configuration set by user
	Logfile  string      // location of the agent's logfile, empty for stdout
	Logger   *log.Logger // log.Logger pointer passed to other code to homogenize output
	Sources  []source    // Array of sources to be captured
	Metadata *LogEvent   // LogEvent pointer representing metadata
	Pidfile  string      // Location of the Pidfile, empty for none
	Quit     chan bool   // Channel used to notify capture routines to exit based on OS signals
}

func main() {
	apiKeyFlag := cli.StringFlag{
		Name:   "api-key",
		Usage:  "timber API key to use when capturing stdin",
		EnvVar: "TIMBER_API_KEY",
	}

	configFlag := cli.StringFlag{
		Name:  "config, c",
		Usage: "config file to use",
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

func commonCaptureRunSetup(ctx *cli.Context, configFileRequired bool) (*commonCaptureRunContext, error) {
	logfile := ctx.String("output-log-file")

	logger := configureLogger(logfile)

	log.Printf("Timber Agent v%s is starting", version)

	// this channel will close when we receive SIGINT or SIGTERM, hopefully giving
	// us enough of a chance to shut down gracefully
	quit := handleSignals()

	pidfile := ctx.String("pidfile")

	// We use a pointer to a zero-value Config struct as the default for
	// config. In the lines below, it will be replaced with a pointer to a
	// struct based on a configuration file. However, if that fails and the
	// value set below is used, the proper defaults will still be set when
	// normalizeConfig is called.
	config := &Config{}
	configFilePath := ctx.String("config")
	configFile, err := os.Open(configFilePath)

	if configFileRequired {
		if err != nil {
			log.Printf("Could not open config file at %s: %s", configFilePath, err)
			return nil, err
		}

		log.Printf("Opened configuration file at %s", configFilePath)
		config, err = parseConfig(configFile)
		if err != nil {
			log.Printf("Could not parse contents of configuration file: %s", err)
			return nil, err
		}
	} else {
		if err != nil {
			log.Printf("Could not open config file at %s: %s", configFilePath, err)
			log.Printf("Config file is not required in this mode")
		} else {
			log.Printf("Opened configuration file at %s", configFilePath)
			config, err = parseConfig(configFile)

			if err != nil {
				log.Printf("Could not parse contents of configuration file: %s", err)
				return nil, err
			}
		}
	}

	normalizeConfig(config)

	log.Printf("Log Collection Endpoint: %s", config.Endpoint)
	log.Printf("Using filesystem polling: %s", config.Poll)
	log.Printf("Maximum time between sends: %d seconds", config.BatchPeriodSeconds)

	// Once the configuration has been fetched, we build the base of the metadata that
	// will accompany every log frame sent to the collection endpoint. The metadata is
	// of the type *LogEvent.
	metadata := BuildBaseMetadata(config)

	runContext := &commonCaptureRunContext{
		Config:   config,
		Logfile:  logfile,
		Logger:   logger,
		Metadata: metadata,
		Pidfile:  pidfile,
		Quit:     quit,
	}

	if runContext.Pidfile != "" {
		writePIDFile(runContext.Pidfile)
	}

	return runContext, nil
}

func commonCaptureRunTeardown(runContext *commonCaptureRunContext) error {
	if runContext.Pidfile != "" {
		// We don't care if removing the PID file errors; there isn't
		// much we can do, and the function itself reports it to the log
		_ = removePIDFile(runContext.Pidfile)
	}

	return nil
}

func runCaptureStdin(ctx *cli.Context) error {
	runContext, err := commonCaptureRunSetup(ctx, false)

	if err != nil {
		// Exit with 65, EX_DATAERR, to indicate that the configuration file data
		// was in an incorrect, unparseable format
		os.Exit(65)
	}

	apiKey := ctx.String("api-key")

	if apiKey != "" {
		runContext.Config.DefaultApiKey = apiKey
	}

	if err = validateConfigStdin(runContext.Config); err != nil {
		log.Printf("%s", err)
		os.Exit(65)
	}

	log.Println("Preparing to read data over STDIN")

	// The metadata doesn't not rquire further modification, so we just
	// transform is to JSON.
	mdJSON, err := runContext.Metadata.EncodeJSON()
	// The JSON is a byte array, convert it to a string
	mdString := string(mdJSON)

	tailer := NewReaderTailer(os.Stdin, runContext.Quit)

	sources := make([]source, 1)

	sources[0] = source{
		ApiKey:   runContext.Config.DefaultApiKey,
		Tailer:   tailer,
		Metadata: mdString,
	}

	runContext.Sources = sources
	commonCaptureRun(runContext)
	commonCaptureRunTeardown(runContext)

	return nil
}

func runCaptureFiles(ctx *cli.Context) error {
	// Configure an initial logger for any daemon error output to
	// use; the real logger will be configured in the call to commonCaptureRunSetup
	logDestination := ctx.String("output-log-file")
	_ = configureLogger(logDestination)

	// If the user has set the `--daemonize` flag, then we call the
	// Daemonize() function. The function is defined by either
	// daemon.go or daemon_linux.go depending on the build platform.
	// Daemonization is only possible on Linux, see daemon.go for
	// a full discussion on this
	if ctx.Bool("daemonize") {
		if err := Daemonize(); err != nil {
			logDaemonFailMessage(ctx, err)
			return err
		}
	}

	// If we have reached this point, daemonization is either complete, or
	// is not necessary

	runContext, err := commonCaptureRunSetup(ctx, true)

	if err = validateConfigFiles(runContext.Config); err != nil {
		log.Printf("%s", err)
		os.Exit(65)
	}

	log.Println("Preparing to read files based on configuration")

	addFileSources(runContext)
	commonCaptureRun(runContext)
	commonCaptureRunTeardown(runContext)

	return nil
}

func addFileSources(runContext *commonCaptureRunContext) {
	sources := make([]source, 0)

	for _, file := range runContext.Config.Files {
		// Takes the base of the file's path so that "/var/log/apache2/access.log"
		// becomes "access.log"
		fileName := path.Base(file.Path)

		// Makes a copy of the metadata; we only want set the filename on the
		// local copy of the metadata
		localMetadata := *runContext.Metadata // localMetadata is of type LogEvent
		md := &localMetadata                  // md is of type *LogEvent
		md.Context.Source.FileName = fileName
		mdJSON, err := md.EncodeJSON()

		if err != nil {
			// If there was an error encoding to JSON, we do not add it to the sources
			// list and therefore do not tail it
			log.Printf("Failed to encode additional metadata as JSON while preparing to tail %s", file.Path)
			log.Printf("%s will not be tailed", file.Path)
			continue
		}

		mdString := string(mdJSON)

		log.Printf("Preparing to tail %s", file.Path)
		tailer := NewFileTailer(file.Path, runContext.Config.Poll, runContext.Quit, runContext.Logger)

		newSource := source{
			ApiKey:   file.ApiKey,
			Metadata: mdString,
			Tailer:   tailer,
		}

		sources = append(sources, newSource)
	}

	runContext.Sources = sources

	return
}

func commonCaptureRun(runContext *commonCaptureRunContext) error {
	client := retryablehttp.NewClient()
	client.Logger = runContext.Logger
	client.HTTPClient.Timeout = 10 * time.Second

	var wg sync.WaitGroup
	for _, source := range runContext.Sources {
		go func() {
			bufChan := make(chan *bytes.Buffer)
			go Batch(source.Tailer.Lines(), bufChan, runContext.Config.BatchPeriodSeconds)
			Forward(bufChan, client, runContext.Config.Endpoint, source.ApiKey, source.Metadata)

			wg.Done()
		}()
		wg.Add(1)
	}

	wg.Wait()

	return nil
}

// handleSignals returns a chan bool that will be closed
// when an OS signal is sent requesting the agent to shut down.
//
// If the request to shut down is not handled in a certain amount of time
// or the OS sends another shut down signal, the agent will immediately
// exit
func handleSignals() chan bool {
	quit := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Starts a goroutine that will handle the inbound signal
	// from the OS. When a signal is received, the
	go func() {
		signal := <-signals
		log.Println(fmt.Sprintf("got %s, shutting down...", signal))
		close(quit)
		timeout := time.After(signalHandlingTimeout)
		select {
		case <-signals:
			os.Exit(1)
		case <-timeout:
			os.Exit(1)
		}
	}()

	return quit
}

// writePIDFile writes the agent's process ID to the given file location
func writePIDFile(pidfileLocation string) {
	pid := int64(os.Getpid())
	pidString := strconv.FormatInt(pid, 10)

	pidfile, err := os.OpenFile(pidfileLocation, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)

	if err != nil {
		log.Printf("Failed to open \"%s\" to write PID to: %v\n", pidfileLocation, err)
		os.Exit(1)
	}

	defer pidfile.Close()

	_, err = pidfile.WriteString(pidString)

	if err != nil {
		log.Printf("Failed to write PID to open file \"%s\": %v\n", pidfileLocation, err)
		os.Exit(1)
	}

	return
}

// removePIDFile deletes the file holding the process ID
func removePIDFile(pidfileLocation string) error {
	err := os.Remove(pidfileLocation)

	if err != nil {
		log.Printf("Unable to remove PID file from \"%s\": %v", pidfileLocation, err)
		return err
	}

	return nil
}

// configureLogger sets up the standard logger. The parameter `destination` can either
// be a file path or an empty string. If it is an empty string, logs will be
// printed to STDOUT. If a valid path is provided, we attempt to open that in
// append-only mode for writing out log messages.
//
// The end-user provides the destination using the --output-log-file flag.
//
// Internally, we assume that STDOUT will be used, then we attempt to open
// an io Writer for the file path the user provided (if any). If that is
// successful, we then redirect logs there. This lets us fallback gracefully
// on STDOUT
//
// If the user has set a destination path, but the file cannot be opened for writing,
// **the agent will exit**. It will attempt to notify
// the user by printing a notice to STDOUT; since the user is already expecting
// the output to be sent to the specified file, they may not pay attention to
// STDOUT, or the STDOUT the output is directed to may not be in their TTY
// session (for example, if agent is also directed to daemonize).
//
// This could cause significant confusion for the user, but unfortunately there
// is not much we can do at this point.
//
// Note that if a destination file is used, it will not be closed when the agent exits
// and the OS is expected to clean up any remaining file descriptor.
func configureLogger(destination string) *log.Logger {
	// Log messages are printed out with the Date and Time in UTC zone; because of the way
	// the Go log package is designed, we cannot define the format or ordering of this
	// metadata
	log_flags := log.Ldate | log.Ltime | log.LUTC
	log_prefix := ""

	// We configure the default logger, which is used whenever log.* is called
	// The default logger is used throughout the agent. For packages that hook
	// into a logger pointer, we provide the `logger` pointer below

	log.SetOutput(os.Stdout)
	log.SetPrefix(log_prefix)
	log.SetFlags(log_flags)

	// logger is a pointer to a Logger; it should be used when a package
	// needs to be passed a log.Logger pointer.
	logger := log.New(os.Stdout, "", log_flags)

	// Check if the user set an agent log path. If so, attempt to open it for writing
	// and redirect log output here
	if destination != "" {
		logfile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0664)

		if err != nil {
			// Encountered some error while opening the agent log file for writing; we will _try_ to
			// inform the user by writing to STDOUT, but depending on the context the agent is running in,
			// the user might not see this
			log.Printf("Attempted to open \"%s\" for agent logging, but failed: %v\n", destination, err)
			// Exiting; the user specified a destination file but we cannot honor it.
			os.Exit(1)
		} else {
			// Set the destination of the default logger
			log.SetOutput(logfile)
			// Set the destination of the pointer logger
			logger.SetOutput(logfile)
		}
	}

	return logger
}

// At this point, we have failed to daemonize. Unfortunately, we don't
// know what point in the daemonization process godaemon had reached.
//
// We will make a best-effort attempt to notify the user of the reason
// why and then exit with a non-zero code
//
// If we are still in stage 0, then the system context remains the same
// and we still have access to STDOUT and STDERR.
//
// If we are in stage 1, then we no longer have a logical STDOUT or STDERR
// that the user will be able to access
// function will CLI context to print out a message
// to print the error out to the agent log (or STDOUT if no log path has been
// set).
func logDaemonFailMessage(ctx *cli.Context, err error) {
	// Fallback to STDOUT even if it might not go to a TTY in case the log file
	// cannot be opened or the path just wasn't set
	log.SetOutput(os.Stdout)

	// Atempt to write the log information to the agent-log-file path given by the user
	if destination := ctx.String("agent-log-file"); destination != "" {
		logfile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0664)

		if err != nil {
			log.Printf("Attempted to open \"%s\" for agent logging, but failed: %v\n", destination, err)
		} else {
			defer logfile.Close()
			log.SetOutput(logfile)
		}
	}

	log.Printf("Failed to daemonize: %v\n", err)
}
