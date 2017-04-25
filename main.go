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
	log.SetOutput(os.Stdout)

	config, err := readConfig(ctx.String("config"))
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

		tailer := NewTailer(tailReader(os.Stdin), ctx.String("api-key"))

		// TODO: maybe have a GlobalConfig subset or interface to pass here
		return tailer.Run(config, quit)

	} else {
		var wg sync.WaitGroup
		log.Println("  Files:")
		for _, file := range config.Files {
			log.Printf("    %s", file.Path)

			tailer := NewTailer(tailFile(file.Path, config.Poll), file.ApiKey)
			tailer.After = func() {
				wg.Done()
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

func tailReader(r io.Reader) chan string {
	ch := make(chan string)
	scanner := bufio.NewScanner(r)

	go func() {
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Println("error reading stdin: ", err)
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
				log.Println("error reading from file: ", err)
			} else {
				ch <- line.Text
			}
		}
	}()

	return ch
}
