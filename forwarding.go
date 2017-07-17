package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/hashicorp/go-retryablehttp"
)

func Forward(bufChan chan *bytes.Buffer, client *retryablehttp.Client, endpoint, apiKey string, metadata []byte) {
	token := base64.StdEncoding.EncodeToString([]byte(apiKey))
	for buf := range bufChan {
		req, err := retryablehttp.NewRequest("POST", endpoint, bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Add("Content-Type", "text/plain")
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
		req.Header.Add("Timber-Metadata-Override", string(metadata))

		resp, err := client.Do(req)
		if err != nil {
			// retries have already happened at this point, so give up
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("flushed buffer (status code %d)", resp.StatusCode)
		} else {
			log.Fatalf("unexpected response (status code %d)", resp.StatusCode)
		}
	}
}
