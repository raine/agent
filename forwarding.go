package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

var defaultHTTPClient = retryablehttp.NewClient()
var UserAgent = fmt.Sprintf("timber-agent/%s", version)

func init() {
	defaultHTTPClient.HTTPClient.Timeout = 10 * time.Second
	// Retry "forever"
	defaultHTTPClient.RetryMax = math.MaxInt32
}

func Forward(messageChan chan *LogMessage, httpClient *retryablehttp.Client, endpoint, apiKey string, metadata []byte) error {
	// Set the logger when the function is called to ensure we pickup any logger changes.
	httpClient.Logger = standardLoggerAlternative
	token := base64.StdEncoding.EncodeToString([]byte(apiKey))
	authorization := fmt.Sprintf("Basic %s", token)

	for message := range messageChan {
		req, err := retryablehttp.NewRequest("POST", endpoint, bytes.NewReader(message.Lines))
		if err != nil {
			logger.Fatal(err)
		}

		req.Header.Add("Content-Type", "text/plain")
		req.Header.Add("Authorization", authorization)
		req.Header.Add("User-Agent", UserAgent)

		if len(metadata) > 0 {
			encodedMetadata := base64.StdEncoding.EncodeToString(metadata)
			req.Header.Add("Timber-Metadata-Override", encodedMetadata)
		}

		// We do not need to handle this error since we retry "forever"
		resp, _ := httpClient.Do(req)

		// We should not reach this if, but require it for testing
		if resp == nil {
			return errors.New("httpClient did not return a response")
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Warn("unable to read response body")
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Infof("flushed buffer (status code %d)", resp.StatusCode)

			// Store state in global state upon success
			if message.Position != 0 {
				// If position != 0, we have a LogMessage that supports recording state
				UpdateStateOffset(message.Filename, message.Position)
			}
		} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return errors.New(fmt.Sprintf("unexpected response (status code %d): %s", resp.StatusCode, string(body)))
		}
	}

	return nil
}

func ForwardStdin(endpoint string, apiKey string, batchPeriodSeconds int64, metadata *LogEvent, quit chan bool) error {
	logger.Info("Starting forward for STDIN")

	encodedMetadata, err := metadata.EncodeJSON()
	if err != nil {
		// If there was an error encoding to JSON, we do not add it to the sources
		// list and therefore do not tail it
		logger.Error("Failed to encode additional metadata as JSON while preparing to tail STDIN")
		return err
	}

	messageChan := make(chan *LogMessage)
	tailer := NewReaderTailer(os.Stdin, quit)

	// Here we run our batcher in the background and return from Forward
	// Forward will block until the tailer is closed
	go Batch(tailer.Lines(), messageChan, batchPeriodSeconds)
	return Forward(messageChan, defaultHTTPClient, endpoint, apiKey, encodedMetadata)
}

func ForwardFile(filePath string, readNewFileFromStart bool, endpoint string, apiKey string, poll bool, batchPeriodSeconds int64, metadata *LogEvent, quit chan bool, stop chan bool) error {
	logger.Infof("Starting forward for file %s", filePath)

	// Takes the base of the file's path so that "/var/log/apache2/access.log"
	// becomes "access.log"
	fileName := path.Base(filePath)

	// Makes a copy of the metadata; we only want set the filename on the
	// local copy of the metadata
	localMetadata := *metadata // localMetadata is of type LogEvent
	md := &localMetadata       // md is of type *LogEvent
	md.ensureSourceContext()
	md.Context.Source.FileName = fileName

	encodedMetadata, err := md.EncodeJSON()
	if err != nil {
		// If there was an error encoding to JSON, we do not add it to the sources
		// list and therefore do not tail it
		logger.Errorf("Failed to encode additional metadata as JSON while preparing to tail %s", filePath)
		return err
	}

	messageChan := make(chan *LogMessage)
	tailer := NewFileTailer(filePath, readNewFileFromStart, poll, quit, stop)

	// Here we run our batcher in the background and return from Forward
	// Forward will block until the tailer is closed
	go Batch(tailer.Lines(), messageChan, batchPeriodSeconds)
	return Forward(messageChan, defaultHTTPClient, endpoint, apiKey, encodedMetadata)
}
