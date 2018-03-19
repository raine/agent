package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// Used to generate random host
const chars = "abcdefghijklmnopqrstuvwxyz"

// getKubernetesClientWithEnvironment helper function to ensure environment changes do not persist between tests
func getKubernetesClientWithEnvironment(host, port string) (*KubernetesClient, error) {
	os.Setenv("TIMBER_AGENT_PROXY_SERVICE_HOST", host)
	os.Setenv("TIMBER_AGENT_PROXY_SERVICE_PORT", port)
	kubernetesClient, err := GetKubernetesClient()
	os.Unsetenv("TIMBER_AGENT_PROXY_SERVICE_HOST")
	os.Unsetenv("TIMBER_AGENT_PROXY_SERVICE_PORT")

	return kubernetesClient, err
}

// getHostAndPortFromURL helper function to split httptest server url into host and port
func getHostAndPortFromURL(url string) (string, string) {
	// url string is of form http://ipaddr:port
	split := strings.Split(url, ":")
	host := strings.TrimLeft(split[1], "/")
	port := split[2]

	return host, port
}

// randomHost generates a random host name of length 10
func randomHost() string {
	var host string

	for i := 0; i < 10; i++ {
		index := rand.Intn(len(chars))
		host += string(chars[index])
	}

	return host
}

// randomPort generates a random port number between 1024 and 65535
func randomPort() string {
	max := 65535
	min := 1024

	return string(rand.Intn(max-min) + min)
}

// GetKubernetesClient()
// When neither TIMBER_AGENT_PROXY_SERVICE_HOST or TIMBER_AGENT_PROXY_SERVICE_PORT is not set, client creation should fail
func TestGetKubernetesClientWithoutEnvVars(test *testing.T) {
	kubernetesClient, err := GetKubernetesClient()

	if err == nil {
		test.Fatal("Expecting kubernetesClient creation to fail")
	}

	if kubernetesClient != nil {
		test.Fatal("Expecting kubernetesClient to be nil")
	}
}

// GetKubernetesClient()
// When TIMBER_AGENT_PROXY_SERVICE_PORT is not set, client creation should fail
func TestGetKubernetesClientWithHostEnvVar(test *testing.T) {
	os.Setenv("TIMBER_AGENT_PROXY_SERVICE_HOST", randomHost())

	kubernetesClient, err := GetKubernetesClient()
	os.Unsetenv("TIMBER_AGENT_PROXY_SERVICE_HOST")

	if err == nil {
		test.Fatal("Expecting kubernetesClient creation to fail")
	}

	if kubernetesClient != nil {
		test.Fatal("Expecting kubernetesClient to be nil")
	}
}

// GetKubernetesClient()
// When TIMBER_AGENT_PROXY_SERVICE_HOST is not set, client creation should fail
func TestGetKubernetesClientWithPortEnvVar(test *testing.T) {
	os.Setenv("TIMBER_AGENT_PROXY_SERVICE_PORT", randomPort())

	kubernetesClient, err := GetKubernetesClient()
	os.Unsetenv("TIMBER_AGENT_PROXY_SERVICE_PORT")

	if err == nil {
		test.Fatal("Expecting kubernetesClient creation to fail")
	}

	if kubernetesClient != nil {
		test.Fatal("Expecting kubernetesClient to be nil")
	}
}

// GetKubernetesClient()
// When TIMBER_AGENT_PROXY_SERVICE_HOST AND TIMBER_AGENT_PROXY_SERVICE_PORT are set, client creation should succeed and
// return a *KubernetesClient
func TestGetKubernetesClientWithHostAndPortEnvVars(test *testing.T) {
	kubernetesClient, err := getKubernetesClientWithEnvironment(randomHost(), randomPort())

	if err != nil {
		test.Fatal("Expecting kubernetesClient creation to succeed")
	}

	if reflect.TypeOf(kubernetesClient).Name() == "*KubernetesClient" {
		test.Fatal("Expecting kubernetesClient to be a *KubernetesClient")
	}
}

// Available()
// When the endpoint is available, Available() should return `true`
func TestKubernetesClientAvailableTrue(test *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	available := kubernetesClient.Available()

	if available == false {
		test.Fatal("Expected connection to mock Kubernetes API to succeed")
	}
}

// Available()
// When the endpoint is available but returns a non-200 response, Available() should return `false`
func TestKubernetesClientAvailable500(test *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))

	// URL is of the form http://ipaddr:port
	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	available := kubernetesClient.Available()

	if available != false {
		test.Fatal("Expected connection to mock Kubernetes API to fail")
	}
}

// Available()
// When the endpoint is available but unresponsive, the connection should timeout and Available() should return `false`
func TestKubernetesClientAvailableTimeout(test *testing.T) {
	kubernetesClient, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// default timeout is 1 second
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))

	kubernetesClient.BaseEndpoint = ts.URL
	available := kubernetesClient.Available()

	if available != false {
		test.Fatal("Expected connection to mock Kubernetes API to fail")
	}
}

// Available()
// When the endpoint is unavailable, the connection should timeout and Available() should return `false`
func TestKubernetesClientAvailableFalse(test *testing.T) {
	kubernetesClient, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())
	available := kubernetesClient.Available()

	if available != false {
		test.Fatal("Expected connection to mock Kubernetes API to fail")
	}
}

// AddKubernetesMetadata()
// When file path is not in the expected form, no metadata should be added.
func TestAddKubernetesMetadataBadFilePath(test *testing.T) {
	// filePath is bad because it does not match expected format of /path/PODNAME_NAMESPACE_CONTAINERNAME.ext
	filePath := "/known/to/be/bad-file-path.log"

	kubernetesClient, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())
	kubernetesContext, err := GetKubernetesMetadata(kubernetesClient, filePath)

	if _, ok := err.(*KubernetesLogFileParseError); !ok {
		test.Fatal("Expected err not to be nil and of type *KubernetesLogFileParseError")
	}

	if !cmp.Equal(kubernetesContext, &KubernetesContext{}) {
		test.Fatal("Expected *KubernetesContext to be empty")
	}
}

// AddKubernetesMetadata()
// When file path is in the expected form, some metadata should be collected.
func TestAddKubernetesMetadataGoodFilePathClientAvailable(test *testing.T) {
	filePath := "/known/to/be/good_file_path.log"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	kubernetesContext, err := GetKubernetesMetadata(kubernetesClient, filePath)

	expected := &KubernetesContext{
		ContainerName: "path",
		Namespace:     "file",
		PodName:       "good",
	}

	if err != nil {
		test.Fatal("Expected err to be nil")
	}

	if !cmp.Equal(kubernetesContext, expected) {
		test.Fatalf("Expected %s, got %s", expected, kubernetesContext)
	}
}

func TestAddKubernetesMetadataGoodFilePathClientUnavailable(test *testing.T) {
	filePath := "/known/to/be/good_file_path.log"

	kubernetesClient, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())
	kubernetesContext, err := GetKubernetesMetadata(kubernetesClient, filePath)

	expected := &KubernetesContext{
		ContainerName: "path",
		Namespace:     "file",
		PodName:       "good",
	}

	if _, ok := err.(*KubernetesClientUnavailableError); !ok {
		test.Fatal("Expected err not to be nil and of type *KubernetesLogFileParseError")
	}

	if !cmp.Equal(kubernetesContext, expected) {
		test.Fatalf("Expected %s, got %s", expected, kubernetesContext)
	}
}

// JSON fixtures for testing KubernetesClient.GetMetadata and AddKubernetesMetadata
var DaemonSetMetadataJSON = `{
	"name": "daemonset-name",
	"metadata": {
		"labels": {
			"name": "daemonset-name"
		}
	}
}`
var DeploymentMetadataJSON = `{
	"name": "deployment-name",
	"metadata": {
		"labels": {
			"name": "deployment-name"
		}
	}
}`
var PodMetadataWithoutOwnerJSON = `{
	"name": "pod-name",
	"metadata": {
		"labels": {
			"name": "pod-name"
		}
	}
}`
var PodMetadataWithOwnerJSON = `{
	"name": "pod-name",
	"metadata": {
		"labels": {
			"name": "pod-name"
		},
		"ownerReferences": [{
			"kind": "ReplicaSet",
			"name": "replicaset-name",
			"apiVersion": "extensions/v1beta1"
		}]
	}
}`
var ReplicaSetWithoutOwnerMetdataJSON = `{
	"name": "replicaset-name",
	"metadata": {
		"labels": {
			"name": "replicaset-name"
		}
	}
}`
var ReplicaSetWithOwnerMetdataJSON = `{
	"name": "replicaset-name",
	"metadata": {
		"labels": {
			"name": "replicaset-name"
		},
		"ownerReferences": [{
			"kind": "Deployment",
			"name": "deployment-name",
			"apiVersion": "extensions/v1beta1"
		}]
	}
}`
var UnknownMetadataJSON string

// KubernetesClientGetMetadataWithAPIVersion()
// When available, DaemonSet metadata should be parsed and returned
func TestKubernetesClientGetMetadataWithAPIVersionDaemonSet(test *testing.T) {
	expected := &KubernetesResponse{}
	err := json.Unmarshal([]byte(DaemonSetMetadataJSON), expected)
	if err != nil {
		test.Fatal("Failed to parse fixture JSON")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURI := "/apis/extensions/v1beta1/namespaces/test-namespace/daemonsets/daemonset-name"
		if r.RequestURI != expectedURI {
			test.Fatalf("Expected request URI to be %s, but got %s", expectedURI, r.RequestURI)
		}

		w.WriteHeader(200)
		w.Write([]byte(DaemonSetMetadataJSON))
	}))

	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	kubernetesResponse, _ := kubernetesClient.GetMetadataWithAPIVersion("test-namespace", "daemonset", "daemonset-name", "extensions/v1beta1")

	if !cmp.Equal(expected, kubernetesResponse) {
		test.Fatalf("Expected daemonset metadata to be %s, got %s", expected, kubernetesResponse)
	}
}

// KubernetesClientGetMetadataWithAPIVersion()
// When available, Deployment metadata should be parsed and returned
func TestKubernetesClientGetMetadataWithAPIVersionDeployment(test *testing.T) {
	expected := &KubernetesResponse{}
	err := json.Unmarshal([]byte(DeploymentMetadataJSON), expected)
	if err != nil {
		test.Fatal("Failed to parse fixture JSON")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURI := "/apis/extensions/v1beta1/namespaces/test-namespace/deployments/deployment-name"
		if r.RequestURI != expectedURI {
			test.Fatalf("Expected request URI to be %s, but got %s", expectedURI, r.RequestURI)
		}

		w.WriteHeader(200)
		w.Write([]byte(DeploymentMetadataJSON))
	}))

	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	kubernetesResponse, _ := kubernetesClient.GetMetadataWithAPIVersion("test-namespace", "deployment", "deployment-name", "extensions/v1beta1")

	if !cmp.Equal(expected, kubernetesResponse) {
		test.Fatalf("Expected deployment metadata to be %s, got %s", expected, kubernetesResponse)
	}
}

// KubernetesClientGetPodMetadata()
// When available, Pod metadata should be parsed and returned
func TestKubernetesClientGetPodMetadata(test *testing.T) {
	expected := &KubernetesResponse{}
	err := json.Unmarshal([]byte(PodMetadataWithoutOwnerJSON), expected)
	if err != nil {
		test.Fatal("Failed to parse fixture JSON")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURI := "/api/v1/namespaces/test-namespace/pods/pod-name"
		if r.RequestURI != expectedURI {
			test.Fatalf("Expected request URI to be %s, but got %s", expectedURI, r.RequestURI)
		}

		w.WriteHeader(200)
		w.Write([]byte(PodMetadataWithoutOwnerJSON))
	}))

	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	kubernetesResponse, _ := kubernetesClient.GetPodMetadata("test-namespace", "pod-name")

	if !cmp.Equal(expected, kubernetesResponse) {
		test.Fatalf("Expected pod metadata to be %s, got %s", expected, kubernetesResponse)
	}
}

// KubernetesClientGetMetadataWithAPIVersion()
// When available, ReplicaSet metadata should be parsed and returned
func TestKubernetesClientGetMetadataWithAPIVersionReplicaSet(test *testing.T) {
	expected := &KubernetesResponse{}
	err := json.Unmarshal([]byte(ReplicaSetWithoutOwnerMetdataJSON), expected)
	if err != nil {
		test.Fatal("Failed to parse fixture JSON")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURI := "/apis/extensions/v1beta1/namespaces/test-namespace/replicasets/replicaset-name"
		if r.RequestURI != expectedURI {
			test.Fatalf("Expected request URI to be %s, but got %s", expectedURI, r.RequestURI)
		}

		w.WriteHeader(200)
		w.Write([]byte(ReplicaSetWithoutOwnerMetdataJSON))
	}))

	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	kubernetesResponse, _ := kubernetesClient.GetMetadataWithAPIVersion("test-namespace", "replicaset", "replicaset-name", "extensions/v1beta1")

	if !cmp.Equal(expected, kubernetesResponse) {
		test.Fatalf("Expected metadata to be %s, got %s", expected, kubernetesResponse)
	}
}

// KubernetesClientGetMetadata()
// When a Kubernetes resource type is not supported, the metadata request should fail
func TestKubernetesClientGetMetadatWithAPIVersionaUnknown(test *testing.T) {
	kubernetesClient, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())
	_, err := kubernetesClient.GetMetadataWithAPIVersion("namespace", "unknown", "unknown-name", "v1")

	if err == nil {
		test.Fatalf("Expected request to fail, as Unknown is not a supported resource type")
	}
}

// AddKubernetesMetadata()
// When a Pod has a single ownerReference in its metadata, that owner should be added to the KubernetesContext
func TestAddKubernetesMetadataPodWithOwner(test *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		podURL := "/api/v1/namespaces/namespace/pods/pod-name"
		replicaSetURL := "/apis/extensions/v1beta1/namespaces/namespace/replicasets/replicaset-name"

		w.WriteHeader(200)
		switch r.RequestURI {
		case "/healthz":
		case podURL:
			w.Write([]byte(PodMetadataWithOwnerJSON))
		case replicaSetURL:
			w.Write([]byte(ReplicaSetWithoutOwnerMetdataJSON))
		default:
			test.Fatalf("Expected request URI to match %s or %s, but got %s", podURL, replicaSetURL, r.RequestURI)
		}
	}))

	filePath := "/var/log/containers/pod-name_namespace_container-name.log"
	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	kubernetesContext, _ := GetKubernetesMetadata(kubernetesClient, filePath)

	expected := &KubernetesContext{
		ContainerName: "container-name",
		Namespace:     "namespace",
		PodName:       "pod-name",
		RootOwner: map[string]string{
			"kind": "ReplicaSet",
			"name": "replicaset-name",
		},
		Labels: map[string]string{
			"name": "pod-name",
		},
	}

	if !cmp.Equal(expected, kubernetesContext) {
		test.Fatalf("Expected logEvent.Context.Platform.KubernetesContext to be %s, got %s",
			expected, kubernetesContext)
	}
}

// AddKubernetesMetadta()
// When a Pod has a multiple ownerReferences in its metadata, only the root owner should be added to the KubernetesContext
func TestAddKubernetesMetadataPodWithOwners(test *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		podURL := "/api/v1/namespaces/namespace/pods/pod-name"
		replicaSetURL := "/apis/extensions/v1beta1/namespaces/namespace/replicasets/replicaset-name"
		deploymentURL := "/apis/extensions/v1beta1/namespaces/namespace/deployments/deployment-name"

		w.WriteHeader(200)
		switch r.RequestURI {
		case "/healthz":
		case deploymentURL:
			w.Write([]byte(DeploymentMetadataJSON))
		case podURL:
			w.Write([]byte(PodMetadataWithOwnerJSON))
		case replicaSetURL:
			w.Write([]byte(ReplicaSetWithOwnerMetdataJSON))
		default:
			test.Fatalf("Expected request URI to match %s, %s, or %s, but got %s",
				deploymentURL, podURL, replicaSetURL, r.RequestURI)
		}
	}))

	filePath := "/var/log/containers/pod-name_namespace_container-name.log"
	host, port := getHostAndPortFromURL(ts.URL)
	kubernetesClient, _ := getKubernetesClientWithEnvironment(host, port)
	kubernetesContext, _ := GetKubernetesMetadata(kubernetesClient, filePath)

	expected := &KubernetesContext{
		ContainerName: "container-name",
		Namespace:     "namespace",
		PodName:       "pod-name",
		RootOwner: map[string]string{
			"kind": "Deployment",
			"name": "deployment-name",
		},
		Labels: map[string]string{
			"name": "pod-name",
		},
	}

	if !cmp.Equal(expected, kubernetesContext) {
		test.Fatalf("Expected logEvent.Context.Platform.KubernetesContext to be %s, got %s",
			expected, kubernetesContext)
	}
}

func TestCollectAndProcessKubernetesMetadataBadFilePath(test *testing.T) {
	client, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())
	config := NewKubernetesConfig()
	filePath := "/known/to/be/bad-file-path.log"

	metadata := NewLogEvent()

	forwardFile, stop, currentMetadata := CollectAndProcessKubernetesMetadata(client, config, filePath, metadata)

	if !forwardFile {
		test.Fatal("Expected forwardFile to be true")
	}

	if stop != nil {
		test.Fatal("Expected stop channel to be an initialized channel")
	}

	if metadata != currentMetadata {
		test.Fatalf("Expected to receive address or original metadata (%p), got %p", metadata, currentMetadata)
	}
}

func TestCollectAndProcessKubernetesMetadataMatchingFilter(test *testing.T) {
	client, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())
	filePath := "/known/to/be/good_file_path.log"

	config := NewKubernetesConfig()
	config.Exclude = map[string]string{
		"namespaces": "file",
	}

	metadata := NewLogEvent()

	forwardFile, stop, currentMetadata := CollectAndProcessKubernetesMetadata(client, config, filePath, metadata)

	if forwardFile {
		test.Fatal("Expected forwardFile to be false")
	}

	if stop != nil {
		test.Fatal("Expected stop channel to be nil")
	}

	if currentMetadata != nil {
		test.Fatal("Execpted metadata returned to be nil")
	}
}

func TestCollectAndProcessKubernetesMetadataClientUnavailable(test *testing.T) {
	client, _ := getKubernetesClientWithEnvironment(randomHost(), randomPort())
	config := NewKubernetesConfig()
	filePath := "/known/to/be/good_file_path.log"

	context, _ := GetKubernetesMetadata(client, filePath)
	metadata := NewLogEvent()
	metadata.AddKubernetesContext(context)

	forwardFile, stop, currentMetadata := CollectAndProcessKubernetesMetadata(client, config, filePath, metadata)

	if !forwardFile {
		test.Fatal("Expected forwardFile to be true")
	}

	if stop == nil {
		test.Fatal("Expected stop channel to be an initialized channel")
	}

	if metadata == currentMetadata {
		test.Fatal("Expected deep copy of metadata to be returned")
	}

	if !cmp.Equal(metadata, currentMetadata) {
		test.Fatalf("Execpted metadata to equal %s, got %s", metadata, currentMetadata)
	}

}

func TestCollectAndProcessKubernetesMetadataClientUninitialized(test *testing.T) {
	// kubernetesClient should be nil here
	client, _ := GetKubernetesClient()
	config := NewKubernetesConfig()
	filePath := "/known/to/be/good_file_path.log"

	context, _ := GetKubernetesMetadata(client, filePath)
	metadata := NewLogEvent()
	metadata.AddKubernetesContext(context)

	forwardFile, stop, currentMetadata := CollectAndProcessKubernetesMetadata(client, config, filePath, metadata)

	if !forwardFile {
		test.Fatal("Expected forwardFile to be true")
	}

	if stop != nil {
		test.Fatal("Expected stop channel to be nil")
	}

	if metadata == currentMetadata {
		test.Fatal("Expected deep copy of metadata to be returned")
	}

	if !cmp.Equal(metadata, currentMetadata) {
		test.Fatalf("Execpted metadata to equal %s, got %s", metadata, currentMetadata)
	}
}

func TestCollectAndProcessKubernetesMetadataClientAvailable(test *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		podURL := "/api/v1/namespaces/test-namespace/pods/pod-name"

		w.WriteHeader(200)
		switch r.RequestURI {
		case "/healthz":
		case podURL:
			w.Write([]byte(PodMetadataWithoutOwnerJSON))
		default:
			test.Fatalf("Expected request URI to match %s or %s, but got %s",
				podURL, "/", r.RequestURI)
		}
	}))

	host, port := getHostAndPortFromURL(ts.URL)
	client, _ := getKubernetesClientWithEnvironment(host, port)
	config := NewKubernetesConfig()
	filePath := "/var/log/containers/pod-name_test-namespace_container-name.log"

	context, _ := GetKubernetesMetadata(client, filePath)
	metadata := NewLogEvent()
	metadata.AddKubernetesContext(context)

	forwardFile, stop, currentMetadata := CollectAndProcessKubernetesMetadata(client, config, filePath, metadata)

	if !forwardFile {
		test.Fatal("Expected forwardFile to be true")
	}

	if stop != nil {
		test.Fatal("Expected stop channel to be nil")
	}

	if metadata == currentMetadata {
		test.Fatal("Expected deep copy of metadata to be returned")
	}

	if !cmp.Equal(metadata, currentMetadata) {
		test.Fatalf("Execpted metadata to equal %s, got %s", metadata, currentMetadata)
	}
}
