package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type KubernetesClient struct {
	BaseEndpoint string
	HTTPClient   *http.Client
}

type KubernetesLogFileParseError struct {
	message string
}

func NewKubernetesLogFileParseError(message string) *KubernetesLogFileParseError {
	return &KubernetesLogFileParseError{
		message: message,
	}
}

func (e *KubernetesLogFileParseError) Error() string {
	return e.message
}

type KubernetesClientUnavailableError struct {
	message string
}

func NewKubernetesClientUnavailableError(message string) *KubernetesClientUnavailableError {
	return &KubernetesClientUnavailableError{
		message: message,
	}
}

func (e *KubernetesClientUnavailableError) Error() string {
	return e.message
}

//CollectAndProcessKubernetesMetadata Abstracts all logic involved in collected and acting upon Kubernetes metadata,
// with the goal of simplifying our main function and only exposing the necessary details.
func CollectAndProcessKubernetesMetadata(client *KubernetesClient, config *KubernetesConfig, filepath string, metadata *LogEvent) (bool, chan bool, *LogEvent) {
	// A *KubernetesContext is always returned
	context, err := GetKubernetesMetadata(client, filepath)

	// If we have received a KubernetesLogFileParseError, we were and will be unable to retrieve any new metadata and
	// should ship the logs with our existing metadata
	_, ok := err.(*KubernetesLogFileParseError)
	if ok {
		logger.Warnf("Failed to parse log file %s. Logs will be sent without Kubernetes fields.", filepath)
		return true, nil, metadata
	}

	// Attempt to filter based on metadata we have collected so far
	if filter, ok := config.ApplyFilter(context); ok {
		logger.Infof("File logs will not be forwarded due to matching an exclusion filter: %s %s",
			filepath, filter)

		return false, nil, nil
	}

	// By default, in Kubernetes each container has its own log file and as such, each file has its
	// own unique set of metadata. In order to send per file metadata, we create a copy of the already collected
	// host metadata, and append to it Kubernetes metadata. Since the metadata is of the type *LogEvent, which
	// itself contains pointers to structs, a deep copy is required.

	// Attempt to create a deep copy of metadata. If it fails, continue to send logs with existing
	// known-to-be-good metadata.
	metadataCopy := metadata.DeepCopy()
	if metadataCopy == nil {
		logger.Warnf("Failed to add Kubernetes metadata. Logs will be sent without Kubernetes fields.", filepath)
		return true, nil, metadata
	}

	// Add Kubernetes metadata we have collected to our overall metadata context. Saving a reference to this pointer
	// will allows us to add additional Kubernetes metadata if retries are required.
	metadataCopy.AddKubernetesContext(context)

	// If the client was unavailable at the time of our request, we need to retry
	_, ok = err.(*KubernetesClientUnavailableError)
	if ok {
		// Create a channel to be passed to listers. The only channel event will be a close, that will indicate the
		// log source has matched an exclusion filter, and should no longer be forwarded.
		stop := make(chan bool)

		// Start a backgroud task to continually query the Kubernetes API with exponential backoff
		go func() {
			// We are using exponential backoff with jitter to avoid a stampede
			// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
			rand := rand.New(rand.NewSource(time.Now().UnixNano()))

			// Blocks until error or success
			client.GetMetadataFromAPIWithBackoff(context, rand)

			// We attempt to filter again as we may have retrieved new Kubernetes metadata that matches a configured
			// exclusion filter
			if filter, ok := config.ApplyFilter(context); ok {
				logger.Infof("File logs will not be forwarded due to matching an exclusion filter: %s %s",
					filepath, filter)

				// Inform listeners that this file should no longer be forwarded
				close(stop)
			}
		}()

		// Return references to be used by listeners
		return true, stop, metadataCopy
	}

	// If we have reached this point, we have either encountered no errors or an error that we do not have specific
	// logic for handing. In either case we return what we have.
	return true, nil, metadataCopy
}

//GetKubernetesMetadata Retrieves as much Kubernetes metadata as possible from both the filepath and the API. Always
// returns a *KubernetesContext which may or may not be empty.
func GetKubernetesMetadata(client *KubernetesClient, filepath string) (*KubernetesContext, error) {
	context := &KubernetesContext{}

	err := GetKubernetesMetadataFromFile(filepath, context)
	if err != nil {
		return context, err
	}

	// If the *KubernetesClient fails to be allocated, the returned value is nil. We check here since our client is
	// initialized in the global scope, and we can collect some metadata without it.
	if client == nil {
		return context, errors.New("KubernetesClient is nil")
	}

	err = client.GetMetadataFromAPI(context)
	if err != nil {
		return context, err
	}

	return context, nil
}

//GetKubernetesMetadataFromFile Given a filePath, the given *KubernetesContext will be updated with discovered metadata.
// The expected format of the file is /path/to/file/PODNAME_NAMESPACE_CONTAINERNAME.ext
func GetKubernetesMetadataFromFile(filePath string, context *KubernetesContext) error {
	fileName := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))
	parts := strings.Split(fileName, "_")

	if len(parts) != 3 {
		logger.Warn("Kubernetes log file is not in the expected format. No Kubernetes metadata will be collected for %s",
			filePath)
		return NewKubernetesLogFileParseError(fmt.Sprintf("Unable to parse log file: %s", filePath))
	}

	context.PodName = parts[0]
	context.Namespace = parts[1]
	context.ContainerName = parts[2]

	return nil
}

//GetKubernetesClient Return a *KubernetesClient, or nil if an error is encountered
func GetKubernetesClient() (*KubernetesClient, error) {
	host := os.Getenv("TIMBER_AGENT_PROXY_SERVICE_HOST")
	if host == "" {
		return nil, errors.New("Could not read TIMBER_AGENT_PROXY_SERVICE_HOST from environment")
	}

	port := os.Getenv("TIMBER_AGENT_PROXY_SERVICE_PORT")
	if port == "" {
		return nil, errors.New("Could not read TIMBER_AGENT_PROXY_SERVICE_PORT from environment")
	}

	kubernetesClient := &KubernetesClient{
		BaseEndpoint: fmt.Sprintf("http://%s:%s", host, port),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Second,
		},
	}

	logger.Warn(kubernetesClient.BaseEndpoint)

	return kubernetesClient, nil
}

//GetMetadataFromAPI The given *KubernetesContext will be updated with metadata discovered from the API.
func (client *KubernetesClient) GetMetadataFromAPI(context *KubernetesContext) error {
	if !client.Available() {
		return NewKubernetesClientUnavailableError("Kubernetes API is unavailable")
	}

	// Query api to collect addtional metadata
	labels, err := client.getPodLabels(context.Namespace, context.PodName)
	if err != nil {
		logger.Warnf("Failed to retrieve labels for Kubernetes Pod %s in namespace %s", context.PodName,
			context.Namespace)
	} else {
		logger.Infof("Retrieved labels for Kubernetes Pod %s in namespace %s", context.PodName, context.Namespace)
		context.Labels = labels
	}

	owner, err := client.getPodRootOwner(context.Namespace, context.PodName)
	if err != nil {
		logger.Warnf("Failed to retrieve root owner for Kubernetes Pod %s in namespace %s", context.PodName,
			context.Namespace)
	} else {
		logger.Infof("Retrieved root owner for Kubernetes Pod %s in namespace %s", context.PodName,
			context.Namespace)
		context.RootOwner = owner
	}

	return nil
}

//Available Returns true if the configured Kubernetes API base URL returns a success, false otherwise
func (client *KubernetesClient) Available() bool {
	resp, err := client.HTTPClient.Get(client.BaseEndpoint + "/healthz")
	if err != nil {
		return false
	}
	resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}

	return true
}

//GetMetadataFromAPIWithBackoff Queries the Kubernetes API for metadata and if available updates the *KubernetesContext.
// If the API is unavailable, retries are performed with exponential backoff.
func (client *KubernetesClient) GetMetadataFromAPIWithBackoff(context *KubernetesContext, rand *rand.Rand) {
	// sleepLimit tracks the maximum amount of time to sleep in seconds
	sleepLimit := 0.0

	// Initialize number of attempts we have made against the API
	attempts := 0

	for {
		err := client.GetMetadataFromAPI(context)
		switch err.(type) {
		case *KubernetesClientUnavailableError:
			// If client is still unavailable, backoff and try again
			// rand is used to provide jitter in order to avoid an API stampede.
			// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
			attempts++

			// We want to cap request retries at ~10 minutes. Since we are
			// exponentially backing off with steps of 2^x, our values around
			// 10 minutes are 512 seconds and 1024 seconds. In this case we
			// opt to use the lesser value of 512 as our limit, since 1024
			// seconds (~17 minutes) is well past our desired maximum. This
			// also means that at most, retries will be between ~4 and ~8
			// minutes.
			// 2^8 seconds == 512 seconds == 8.53 minutes
			// 2^9 seconds == 1024 seconds == 17.07 minutes
			if sleepLimit < 512 {
				sleepLimit = math.Pow(2, float64(attempts))
			}

			// Calculate time to sleep with jitter
			sleepDuration := sleepLimit/2 + (rand.Float64()*sleepLimit)/2
			logger.Warnf("Failed to get metadata for container %s in pod %s in namespace %s. Retrying in %.f seconds.",
				context.ContainerName, context.PodName, context.Namespace, sleepDuration)

			// time.Duration defaults to nanoseconds, so we multiply by seconds factor
			time.Sleep(time.Duration(sleepDuration) * time.Second)
		default:
			// We have succeeded or received a different error and no longer retry
			return
		}
	}
}

type KubernetesResponse struct {
	Metadata *KubernetesMetadata `json:"metadata"`
}

type KubernetesOwnerReference struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion"`
}

type KubernetesMetadata struct {
	Labels          map[string]string           `json:"labels"`
	OwnerReferences []*KubernetesOwnerReference `json:"ownerReferences,omitempty"`
}

//GetPodMetadata Returns *KubernetesResponse for given pod
func (client *KubernetesClient) GetPodMetadata(namespace, name string) (*KubernetesResponse, error) {
	return client.GetMetadataWithAPIVersion(namespace, "pod", name, "v1")
}

//GetMetadataWithAPIVersion Returns *KubernetesResponse for given resource.
func (client *KubernetesClient) GetMetadataWithAPIVersion(namespace, kind, name, apiVersion string) (*KubernetesResponse, error) {
	var url string
	var apiPrefix = "/apis/"

	// Needed for pods and replication controllers
	if apiVersion == "v1" {
		apiPrefix = "/api/"
	}

	pluralKind := strings.ToLower(kind)
	if string(pluralKind[len(pluralKind)-1]) != "s" {
		pluralKind += "s"
	}

	url = fmt.Sprintf("%s%s/namespaces/%s/%s/%s", apiPrefix, apiVersion, namespace, pluralKind, name)

	resp, err := client.HTTPClient.Get(client.BaseEndpoint + url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("Did not receive a valid response from Kubernetes API")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var kr KubernetesResponse
	err = json.Unmarshal(body, &kr)
	if err != nil {
		return nil, err
	}

	return &kr, nil
}

func (client *KubernetesClient) getPodLabels(namespace, name string) (map[string]string, error) {
	kr, err := client.GetPodMetadata(namespace, name)
	if err != nil {
		return nil, err
	}

	return kr.Metadata.Labels, nil
}

// getPodRootOwner attempts to retrieve the top level or root owner from all directly reachable owners of the given pod
// by following ownerReferences found in metadata returned by the Kubernetes API.
func (client *KubernetesClient) getPodRootOwner(namespace, name string) (map[string]string, error) {
	var apiVersion = "v1"
	var kind = "pod"

	for {
		kr, err := client.GetMetadataWithAPIVersion(namespace, kind, name, apiVersion)
		if err != nil {
			return nil, err
		}

		ownerReferences := kr.Metadata.OwnerReferences
		if len(ownerReferences) == 0 {
			return map[string]string{
				"kind": kind,
				"name": name,
			}, nil
		}

		// We are assuming ownerReferences contains only one reference, or the
		// first reference is the one we are interested in.
		ownerReference := ownerReferences[0]
		kind = ownerReference.Kind
		name = ownerReference.Name
		apiVersion = ownerReference.APIVersion
	}
}
