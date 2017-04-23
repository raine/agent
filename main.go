package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/influxdata/tail"
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
	config, err := readConfig(ctx.String("config"))
	if err != nil {
		return err
	}

	if err := validateConfig(config, ctx); err != nil {
		return err
	}

	fmt.Println("Timber agent starting up with config:")
	fmt.Printf("  Endpoint: %s\n", config.Endpoint)
	fmt.Printf("  BatchPeriodSeconds: %d\n", config.BatchPeriodSeconds)
	fmt.Printf("  Poll: %t\n", config.Poll)

	// this channel will close when we receive SIGINT or SIGTERM, hopefully giving
	// us enough of a chance to shut down gracefully
	quit := handleSignals()

	if ctx.IsSet("stdin") {
		fmt.Println("tailing stdin...")

		tailer := Tailer{
			Lines:  tailReader(os.Stdin),
			ApiKey: ctx.String("api-key"),
			After:  func() {},
		}

		// TODO: maybe have a GlobalConfig subset or interface to pass here
		return tailer.Run(config, quit)

	} else {
		var wg sync.WaitGroup
		for _, file := range config.Files {
			fmt.Printf("tailing %s...\n", file.Path)

			tailer := Tailer{
				Lines:  tailFile(file.Path, config.Poll),
				ApiKey: file.ApiKey,
				After: func() {
					wg.Done()
				},
			}

			wg.Add(1)
			go tailer.Run(config, quit)
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
		fmt.Println(fmt.Sprintf("Got %s, shutting down...", signal))
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

func tailReader(r io.Reader) chan string {
	ch := make(chan string)
	scanner := bufio.NewScanner(r)

	go func() {
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading stdin:", err)
		}
		close(ch)
	}()

	return ch
}

func tailFile(filename string, poll bool) chan string {
	ch := make(chan string)
	tailer, err := tail.TailFile(filename, tail.Config{
		Follow: true,
		ReOpen: true,
		Poll:   poll,
	})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for line := range tailer.Lines {
			if err := line.Err; err != nil {
				fmt.Fprintln(os.Stderr, "error reading from file:", err)
			} else {
				ch <- line.Text
			}
		}
	}()

	return ch
}
