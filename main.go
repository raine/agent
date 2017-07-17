package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/VividCortex/godaemon"
	"github.com/hashicorp/go-retryablehttp"

	"gopkg.in/urfave/cli.v1"
)

var version string

func main() {
	app := cli.NewApp()
	app.Name = "timber-agent"
	app.Usage = "forwards logs to timber.io"
	app.Version = version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "config file to use",
			Value: "/etc/timber.toml",
		},
		cli.StringFlag{
			Name:  "pidfile",
			Usage: "will store the pid in `PIDFILE` when set",
		},
		cli.StringFlag{
			Name:  "agent-log-file",
			Usage: "file path to store logs (will use STDOUT if blank)",
		},
		cli.BoolFlag{
			Name:  "daemonize",
			Usage: "starts an instance of agent as a daemon (see documentation)",
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "read logs from stdin instead of tailing files",
		},
		cli.StringFlag{
			Name:   "api-key",
			Usage:  "timber API key to use when forwarding stdin",
			EnvVar: "TIMBER_API_KEY",
		},
	}
	app.Action = run

	app.Run(os.Args)
}

func run(ctx *cli.Context) error {
	// If the user has set the `--daemonize` flag, then we use
	if ctx.Bool("daemonize") {
		if err := daemonize(ctx); err != nil {
			os.Exit(1)
		}
	}

	// If we have reached this point, daemonization is either complete, or
	// is not necessary
	//
	// File descriptors can now be opened without issue

	// Prepare logging
	//
	// The user is allowed to configure logging to a file using the configuration
	// flag --agent-log-file. If the user does not set this configuration value,
	// logs are directed to STDOUT instead.
	//
	// Internally, we assume that STDOUT will be used, then we attempt to open
	// an io Writer for the file path the user provided (if any). If that is
	// successful, we then redirect logs there. This lets us fallback gracefully
	// on STDOUT
	//
	// If the user has set the --agent-log-file configuration flag but the file
	// cannot be opened for writing, **the agent will exit**. It will attempt to notify
	// the user by printing a notice to STDOUT; since the user is already expecting
	// the output to be sent to the specified file, they may not pay attention to
	// STDOUT, or the STDOUT the output is directed to may not be in their TTY
	// session (for example, if the user also specified the --daemonize flag).
	// This could cause significant confusion for the user, but unfortunately there
	// is not much we can do at this point.

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
	if destination := ctx.String("agent-log-file"); destination != "" {
		logfile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0664)

		if err != nil {
			// Encountered some error while opening the agent log file for writing; we will _try_ to
			// inform the user by writing to STDOUT, but depending on the context the agent is running in,
			// the user might not see this
			log.Printf("Attempted to open \"%s\" for agent logging, but failed: %v\n", destination, err)
			// Exiting; the user specified the --agent-log-file but we cannot honor it.
			os.Exit(1)
		} else {
			// Under many circumstances, this defer call will never be called. The majority of the exit
			// sequences the agent goes through call os.Exit() which skips defer calls. Nonetheless,
			// the OS should clean up file descriptors on exit.
			defer logfile.Close()
			// Set the destination of the default logger
			log.SetOutput(logfile)
			// Set the destination of the pointer logger
			logger.SetOutput(logfile)
		}
	}

	if pidfile := ctx.String("pidfile"); pidfile != "" {
		writePIDFile(pidfile)
	}

	configFile, err := os.Open(ctx.String("config"))
	if err != nil {
		return err
	}

	config, err := readConfig(configFile)
	if err != nil {
		return err
	}

	if err := validateConfig(config, ctx); err != nil {
		return err
	}

	// Once the configuration has been fetched, we build the metadata string that will
	// accompany each request. This metadata string is a JSON object derived from a
	// LogEvent struct
	metadata, err := BuildMetadata(config)

	if err != nil {
		return err
	}

	log.Println("Timber agent starting up with config:")
	log.Printf("  Endpoint: %s", config.Endpoint)
	log.Printf("  BatchPeriodSeconds: %d", config.BatchPeriodSeconds)
	log.Printf("  Poll: %t", config.Poll)

	// this channel will close when we receive SIGINT or SIGTERM, hopefully giving
	// us enough of a chance to shut down gracefully
	quit := handleSignals()

	client := retryablehttp.NewClient()
	client.Logger = logger
	client.HTTPClient.Timeout = 10 * time.Second

	if ctx.IsSet("stdin") {
		log.Println("  Stdin: true")

		bufChan := make(chan *bytes.Buffer)
		tailer := NewReaderTailer(os.Stdin, quit)

		go Batch(tailer.Lines(), bufChan, config.BatchPeriodSeconds)

		Forward(bufChan, client, config.Endpoint, ctx.String("api-key"), metadata)

	} else {
		var wg sync.WaitGroup
		log.Println("  Files:")
		for _, file := range config.Files {
			file := file
			log.Printf("    %s", file.Path)

			go func() {
				bufChan := make(chan *bytes.Buffer)
				tailer := NewFileTailer(file.Path, config.Poll, quit, logger)

				go Batch(tailer.Lines(), bufChan, config.BatchPeriodSeconds)

				Forward(bufChan, client, config.Endpoint, file.ApiKey, metadata)

				wg.Done()
			}()
			wg.Add(1)
		}

		wg.Wait()
	}

	if pidfile := ctx.String("pidfile"); pidfile != "" {
		// We don't care if removing the PID file errors; there isn't
		// much we can do, and the function itself reports it to the log
		_ = removePIDFile(pidfile)
	}

	return nil
}

func handleSignals() chan bool {
	quit := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		signal := <-signals
		log.Println(fmt.Sprintf("got %s, shutting down...", signal))
		close(quit)
		timeout := time.After(5 * time.Second)
		select {
		case <-signals:
			os.Exit(1)
		case <-timeout:
			os.Exit(1)
		}
	}()

	return quit
}

// daemonize will attempt to daemonize the executable by calling
// the godaemon package to perform a non-native forking operation; traditionally
// this would be accomplished using the appropriate OS level functionality,
// however this is not possible in Go. The call to `godaemon.MakeDaemon` will
// cause the currently running agent to exit and launch a new agent outside
// the current context.
//
// If daemonization fails, the function will CLI context to print out a message
// to print the error out to the agent log (or STDOUT if no log path has been
// set). The error is returned to the caller and the caller is expected to
// exit. In addition to yielding control back to the caller, this also allows
// any defer statements to complete.
func daemonize(ctx *cli.Context) error {
	_, _, err := godaemon.MakeDaemon(&godaemon.DaemonAttr{})

	if err != nil {
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

	return err
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
