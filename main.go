package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/influxdata/tail"
	"github.com/urfave/cli/altsrc"
	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "timber-agent"
	app.Usage = "forwards logs to timber.io"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "location of the config file to read",
			Value: "/etc/timber.toml",
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "read logs from stdin instead of a file",
		},
		altsrc.NewStringFlag(cli.StringFlag{
			Name:  "file",
			Usage: "log file to forward",
		}),
		altsrc.NewDurationFlag(cli.DurationFlag{
			Name:  "batch-period",
			Usage: "how often to flush logs to the server",
			Value: 5 * time.Second,
		}),
		altsrc.NewBoolFlag(cli.BoolFlag{
			Name:  "poll",
			Usage: "poll files instead of using inotify",
		}),
		altsrc.NewStringFlag(cli.StringFlag{
			Name:  "api-key",
			Usage: "your timber API key",
		}),
		altsrc.NewStringFlag(cli.StringFlag{
			Name:  "endpoint",
			Usage: "the endpoint to which to forward logs",
			Value: "https://ingestion-staging.timber.io/frames",
		}),
	}
	app.Before = altsrc.InitInputSourceWithContext(app.Flags, altsrc.NewTomlSourceFromFlagFunc("config"))
	app.Action = run

	app.Run(os.Args)
}

func run(ctx *cli.Context) error {
	lines, err := buildTail(ctx)
	if err != nil {
		return err
	}

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

	token := base64.StdEncoding.EncodeToString([]byte(ctx.String("api-key")))

	// we send "finished" buffers over this channel to be sent as http requests
	// asynchonously. a slowdown in the sender goroutine will eventually (based on
	// channel buffering) provide backpressure on the log tailing goroutine, which
	// should shed load in response.
	//
	// this design relies on the gc to clean up old buffers, but an alternative
	// would be to have a second channel for sending back old buffers for reuse,
	// which could be a good option if we're seeing excess memory pressure
	done := make(chan bool)
	bufChan := make(chan *bytes.Buffer)
	go sender(ctx.String("endpoint"), token, bufChan, done)

	buf := bytes.NewBuffer([]byte{})
	tick := time.Tick(ctx.Duration("batch-period"))
	for {
		select {
		case line, ok := <-lines:
			io.WriteString(buf, line+"\n")
			// TODO: make this configurable
			if buf.Len() > 1000000 {
				bufChan <- buf
				buf = bytes.NewBuffer([]byte{})
			}
			if !ok { // channel is closed
				bufChan <- buf
				close(bufChan)
				// wait for sender to finish
				<-done
				return nil
			}
		case <-tick:
			if buf.Len() > 0 {
				// TODO: extract a shared version of this, maybe preallocate 2MB buffers
				bufChan <- buf
				buf = bytes.NewBuffer([]byte{})
			}
		case <-quit:
			bufChan <- buf
			close(bufChan)
			// wait for sender to finish
			<-done
			return nil
		}
	}
}

func buildTail(ctx *cli.Context) (chan string, error) {
	if ctx.IsSet("stdin") && ctx.IsSet("file") {
		return nil, cli.NewExitError("can't set both --stdin and --file", 1)
	} else if ctx.IsSet("stdin") {
		return tailStdin(), nil
	} else if ctx.IsSet("file") {
		return tailFile(ctx.String("file"), ctx.IsSet("poll")), nil
	} else {
		return nil, cli.NewExitError("must set one of --stdin or --file", 1)
	}
}

// TODO: pass a reader so this is easier to test
func tailStdin() chan string {
	ch := make(chan string)
	scanner := bufio.NewScanner(os.Stdin)

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

func sender(endpoint, token string, ch chan *bytes.Buffer, done chan bool) {
	transport := http.DefaultTransport
	for buf := range ch {
		req, err := http.NewRequest("POST", endpoint, buf)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Content-Type", "text/plain")
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
		req.Header.Add("Accept", "application/json")
		resp, err := transport.RoundTrip(req)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("flushed buffer, got status code %d", resp.StatusCode)
			resp.Body.Close()
		}
	}
	done <- true
}
