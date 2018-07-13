package main

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"time"
)

func TestForwardForwarding(test *testing.T) {
	bufChan := make(chan *bytes.Buffer, 1)
	bufChan <- bytes.NewBufferString("test log line\n")
	close(bufChan)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		output, err := ioutil.ReadAll(r.Body)
		if err != nil {
			test.Fatal(err)
		}
		actual := strings.TrimSpace(bytes.NewBuffer(output).String())
		expected := "test log line"

		if actual != expected {
			test.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
		}
	}))
	defer ts.Close()

	Forward(bufChan, retryablehttp.NewClient(), ts.URL, "api key", []byte{}, false)
}

func TestForwardRetries(test *testing.T) {
	bufChan := make(chan *bytes.Buffer, 1)
	bufChan <- bytes.NewBufferString("test log line\n")
	close(bufChan)

	retries := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retries < 1 {
			retries += 1
			w.WriteHeader(500)
		} else {
			w.WriteHeader(200)
		}
	}))
	defer ts.Close()

	client := retryablehttp.NewClient()
	client.RetryWaitMin = 0

	Forward(bufChan, client, ts.URL, "api key", []byte{}, false)

	if retries != 1 {
		test.Fatalf("expected 1 retry, got %d", retries)
	}
}

func TestForwardMetadata(test *testing.T) {
	bufChan := make(chan *bytes.Buffer, 1)
	bufChan <- bytes.NewBufferString("test log line\n")
	close(bufChan)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := base64.StdEncoding.EncodeToString([]byte("Metadata test"))
		actual := r.Header.Get("Timber-Metadata-Override")

		if actual != expected {
			test.Fatalf("expected \"%+v\", got \"%+v\"", expected, actual)
		}

		w.WriteHeader(200)
	}))

	defer ts.Close()

	Forward(bufChan, retryablehttp.NewClient(), ts.URL, "api key", []byte("Metadata test"), false)
}

func TestForwardForwardingTimeoutDoesNotFatal(test *testing.T) {
	bufChan := make(chan *bytes.Buffer, 1)
	bufChan <- bytes.NewBufferString("test log line\n")
	close(bufChan)

	retries := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retries += 1
		time.Sleep(2 * time.Millisecond)

		w.WriteHeader(200)
	}))

	defer ts.Close()

	client := retryablehttp.NewClient()
	client.HTTPClient.Timeout = 1 * time.Millisecond
	client.RetryWaitMin = 0

	Forward(bufChan, client, ts.URL, "api key", []byte{}, true)

	if retries != 5 {
		test.Fatalf("expected 1 retry, got %d", retries)
	}
}
