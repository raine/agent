package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

var defaultHTTPClient = retryablehttp.NewClient()
var UserAgent = fmt.Sprintf("timber-agent/%s", version)

func init() {
	defaultHTTPClient.HTTPClient.Timeout = 10 * time.Second
}

func Forward(bufChan chan *bytes.Buffer, httpClient *retryablehttp.Client, endpoint, apiKey string, metadata []byte, discard bool) {
	// Set the logger when the function is called to ensure we pickup any logger changes.
	httpClient.Logger = standardLoggerAlternative
	token := base64.StdEncoding.EncodeToString([]byte(apiKey))
	authorization := fmt.Sprintf("Basic %s", token)

	for buf := range bufChan {
		req, err := retryablehttp.NewRequest("POST", endpoint, bytes.NewReader(buf.Bytes()))
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

		resp, err := httpClient.Do(req)
		if err != nil {
			// retries have already happened at this point, so give up
			if !discard {
				logger.Fatal(err)
			} else {
				logger.Error("Error forwarding logs. Logs discarded.")
				logger.Error(err)
				continue
			}
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Infof("flushed buffer (status code %d)", resp.StatusCode)
		} else {
			logger.Fatalf("unexpected response (status code %d)", resp.StatusCode)
		}
	}
}

func ForwardStdin(endpoint string, apiKey string, batchPeriodSeconds int64, metadata *LogEvent, quit chan bool, discard bool) error {
	logger.Info("Starting forward for STDIN")

	encodedMetadata, err := metadata.EncodeJSON()
	if err != nil {
		// If there was an error encoding to JSON, we do not add it to the sources
		// list and therefore do not tail it
		logger.Error("Failed to encode additional metadata as JSON while preparing to tail STDIN")
		return err
	}

	bufChan := make(chan *bytes.Buffer)
	tailer := NewReaderTailer(os.Stdin, quit)
	go Batch(tailer.Lines(), bufChan, batchPeriodSeconds)
	Forward(bufChan, defaultHTTPClient, endpoint, apiKey, encodedMetadata, discard)

	return nil
}

func ForwardFile(filePath string, endpoint string, apiKey string, poll bool, batchPeriodSeconds int64, metadata *LogEvent, quit chan bool, stop chan bool, discard bool) error {
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

	bufChan := make(chan *bytes.Buffer)
	tailer := NewFileTailer(filePath, poll, quit, stop)
	go Batch(tailer.Lines(), bufChan, batchPeriodSeconds)
	Forward(bufChan, defaultHTTPClient, endpoint, apiKey, encodedMetadata, discard)

	return nil
}
