package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "timber-agent"
	app.Usage = "forwards logs to timber.io"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "config file to use",
			Value: "/etc/timber.toml",
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
	log.SetOutput(os.Stdout)

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

	log.Println("Timber agent starting up with config:")
	log.Printf("  Endpoint: %s", config.Endpoint)
	log.Printf("  BatchPeriodSeconds: %d", config.BatchPeriodSeconds)
	log.Printf("  Poll: %t", config.Poll)

	// this channel will close when we receive SIGINT or SIGTERM, hopefully giving
	// us enough of a chance to shut down gracefully
	quit := handleSignals()

	if ctx.IsSet("stdin") {
		log.Println("  Stdin: true")

		bufChan := make(chan *bytes.Buffer)
		tailer := NewReaderTailer(os.Stdin, quit)

		go Batch(tailer.Lines(), bufChan, config.BatchPeriodSeconds)

		Forward(bufChan, http.DefaultTransport, config.Endpoint, ctx.String("api-key"))

	} else {
		var wg sync.WaitGroup
		log.Println("  Files:")
		for _, file := range config.Files {
			log.Printf("    %s", file.Path)

			go func() {
				bufChan := make(chan *bytes.Buffer)
				tailer := NewFileTailer(file.Path, config.Poll, quit)

				go Batch(tailer.Lines(), bufChan, config.BatchPeriodSeconds)

				Forward(bufChan, http.DefaultTransport, config.Endpoint, file.ApiKey)

				wg.Done()
			}()
			wg.Add(1)

			wg.Wait()
		}
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
