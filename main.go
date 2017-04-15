package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hpcloud/tail"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "timber-agent"
	app.Usage = "forwards logs to timber.io"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file",
			Usage: "log file to forward",
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "read logs from stdin instead of a file",
		},
		cli.DurationFlag{
			Name:  "batch-period",
			Usage: "how often to flush logs to the server",
			Value: 5 * time.Second,
		},
		cli.BoolFlag{
			Name:  "poll",
			Usage: "poll files instead of using inotify",
		},
	}
	app.Action = run

	app.Run(os.Args)
}

func run(ctx *cli.Context) error {
	lines, err := buildTail(ctx)
	if err != nil {
		return err
	}

	// we send "finished" buffers over this channel to be sent as http requests
	// asynchonously. a slowdown in the sender goroutine will eventually (based on
	// channel buffering) provide backpressure on the log tailing goroutine, which
	// should shed load in response.
	//
	// this design relies on the gc to clean up old buffers, but an alternative
	// would be to have a second channel for sending back old buffers for reuse,
	// which could be a good option if we're seeing excess memory pressure
	bufChan := make(chan *bytes.Buffer)
	go sender(bufChan)

	buf := bytes.NewBuffer([]byte{})
	tick := time.Tick(ctx.Duration("batch-period"))
	for {
		select {
		case line := <-lines:
			io.WriteString(buf, line+"\n")
			// TODO: make this configurable
			if buf.Len() > 1000000 {
				bufChan <- buf
				buf = bytes.NewBuffer([]byte{})
			}
		case <-tick:
			if buf.Len() > 0 {
				// TODO: extract a shared version of this, maybe preallocate 2MB buffers
				bufChan <- buf
				buf = bytes.NewBuffer([]byte{})
			}
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

func sender(ch chan *bytes.Buffer) {
	transport := http.DefaultTransport
	for buf := range ch {
		req, err := http.NewRequest("POST", "http://localhost:8080/logs", buf)
		if err != nil {
			log.Fatal(err)
		}
		resp, err := transport.RoundTrip(req)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("flushed buffer, got status code %d", resp.StatusCode)
		}
	}
}
