package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
)

func Forward(bufChan chan *bytes.Buffer, transport http.RoundTripper, endpoint, apiKey string) {
	token := base64.StdEncoding.EncodeToString([]byte(apiKey))
	for buf := range bufChan {
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
}
