package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Available()
// When the endpoint is available, Available() should return `true`
func TestEC2ClientAvailableTrue(test *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	ec2Client := GetEC2Client()
	ec2Client.BaseEndpoint = ts.URL

	available := ec2Client.Available()

	if available != true {
		test.Fatal("Expected connection to metadata provider to succeed")
	}
}

// Available()
// When the endpoint is not available, the connection should timeout
// and Available() should return `false`
func TestEC2ClientAvailableFalse(test *testing.T) {
	ec2Client := GetEC2Client()

	available := ec2Client.Available()

	if available != false {
		test.Fatal("Expected connection to metadata provider to fail")
	}
}

// GetMetadata()
// When the service is available, the metadata should be fetched and returned
// Tests that the client hits the appropriate endpoint
func TestEC2ClientGetMetadata(test *testing.T) {
	expected := "i1934195190"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURI := "/latest/meta-data/instance_id"
		if r.RequestURI != expectedURI {
			test.Fatalf("Expected request URI to be %s, but got %s", expectedURI, r.RequestURI)
		}

		w.WriteHeader(200)
		w.Write([]byte(expected))
	}))

	ec2Client := GetEC2Client()
	ec2Client.BaseEndpoint = ts.URL

	instanceId, err := ec2Client.GetMetadata("instance_id")

	if err != nil {
		test.Fatalf("Expected to get metadata, encountered error instead: %s", err)
	}

	if instanceId != expected {
		test.Fatalf("Expected instance ID of %s, got %s instead", expected, instanceId)
	}
}

// AddEC2Metadata()
// When the service is not available, the LogEvent should not be modified
func TestAddEC2MetadataNoService(test *testing.T) {
	ec2Client := GetEC2Client()
	logEvent := &LogEvent{}

	AddEC2Metadata(ec2Client, logEvent)

	amiID := logEvent.Context.Platform.AWSEC2.AmiID

	if amiID != "" {
		test.Fatalf("Expected AmiID to be an empty string, instead got %s", amiID)
	}
}

// AddEC2Metadata()
// When the service is available, modifies the LogEvent with the appropriate data
func TestAddEC2Metadata(test *testing.T) {
	expected := "i1934195190"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(expected))
	}))

	ec2Client := GetEC2Client()
	ec2Client.BaseEndpoint = ts.URL
	logEvent := &LogEvent{}

	AddEC2Metadata(ec2Client, logEvent)

	instanceID := logEvent.Context.Platform.AWSEC2.InstanceID

	if instanceID != expected {
		test.Fatalf("Expected InstanceID to be %s, instead got %s", expected, instanceID)
	}
}
