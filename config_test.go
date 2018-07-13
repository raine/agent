package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewConfigSetsDefaults(t *testing.T) {
	config := NewConfig()

	if config.Endpoint != "https://logs.timber.io/frames" {
		t.Error("endpoint was not defaulted")
	}

	if config.BatchPeriodSeconds != 3 {
		t.Error("batch period was not defaulted")
	}
}

func TestNewConfigSpecificApiKey(t *testing.T) {
	configString := `
[[files]]
path = "/var/log/log.log"
api_key = "abc:1234"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expectedFileCount := 1
	fileCount := len(config.Files)

	if fileCount != expectedFileCount {
		t.Fatalf("Expected %d files from configuration but %d reported", expectedFileCount, fileCount)
	}

	expectedApiKey := "abc:1234"
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}

func TestNewConfigDefaultApiKey(t *testing.T) {
	configString := `
default_api_key = "zyx:0987"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expectedDefaultApiKey := "zyx:0987"
	defaultApiKey := config.DefaultApiKey

	if defaultApiKey != expectedDefaultApiKey {
		t.Errorf("Expected DefaultApiKey to be %s but got %s", expectedDefaultApiKey, defaultApiKey)
	}
}

func TestNewConfigDefaultFileApiKey(t *testing.T) {
	configString := `
default_api_key = "zyx:0987"

[[files]]
path = "/var/log/log.log"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expectedApiKey := "zyx:0987"
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}

func TestNewConfigNoApiKey(t *testing.T) {
	configString := `
[[files]]
path = "/var/log/log.log"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expectedApiKey := ""
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}

func TestNewConfigMultipleFiles(t *testing.T) {
	configString := `
default_api_key = "default_api_key"

[[files]]
path = "/var/log/log1.log"
api_key = "file1_api_key"

[[files]]
path = "/var/log/log2.log"
`

	config := NewConfig()
	configFile := strings.NewReader(configString)
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	// Check the file count
	expectedFileCount := 2
	fileCount := len(config.Files)

	if fileCount != expectedFileCount {
		t.Fatalf("Expected %d files from configuration but %d reported", expectedFileCount, fileCount)
	}

	// Check the first file path
	expectedPath := "/var/log/log1.log"
	path := config.Files[0].Path

	if path != expectedPath {
		t.Errorf("Expected Path to be %s but got %s", expectedPath, path)
	}

	// Check the first file api key
	expectedApiKey := "file1_api_key"
	apiKey := config.Files[0].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}

	// Check the second file path
	expectedPath = "/var/log/log2.log"
	path = config.Files[1].Path

	if path != expectedPath {
		t.Errorf("Expected Path to be %s but got %s", expectedPath, path)
	}

	// Check the second file api key
	expectedApiKey = "default_api_key"
	apiKey = config.Files[1].ApiKey

	if apiKey != expectedApiKey {
		t.Errorf("Expected ApiKey to be %s but got %s", expectedApiKey, apiKey)
	}
}

func TestNewKubernetesConfigSetsDefaults(t *testing.T) {
	kubernetesConfig := NewKubernetesConfig()

	if kubernetesConfig.Exclude == nil {
		t.Error("exclude should be initialized to map with default values")
	}

	if len(kubernetesConfig.Exclude) == 0 {
		t.Error("exclude should be initialized to map with default values")
	}
}

func TestKubernetesConfigReadExcludeFilter(t *testing.T) {
	configString := `
[kubernetes.exclude]
namespaces = "dev,prod"
`

	configFile := strings.NewReader(configString)
	config := NewConfig()
	config.KubernetesConfig = NewKubernetesConfig()
	err := config.UpdateFromReader(configFile)
	if err != nil {
		panic(err)
	}

	expected := "dev,prod"
	exclude := config.KubernetesConfig.Exclude["namespaces"]

	if !cmp.Equal(expected, exclude) {
		t.Errorf("Expected exclude config %s, but got %s", expected, exclude)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingPodName(t *testing.T) {
	exclude := map[string]string{
		"pods": "match",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		PodName: "match",
	}

	expectedFilter := "pods:match"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}
func TestKubernetesConfigApplyFilterWithNonMatchingPodName(t *testing.T) {
	exclude := map[string]string{
		"pods": "not-a-match",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match",
	}

	expectedFilter := ""
	expectedOk := false
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}
func TestKubernetesConfigApplyFilterWithMatchingNamespace(t *testing.T) {
	exclude := map[string]string{
		"namespaces": "match",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match",
	}

	expectedFilter := "namespaces:match"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithNonMatchingNamespace(t *testing.T) {
	exclude := map[string]string{
		"namespaces": "not-a-match",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match",
	}

	expectedFilter := ""
	expectedOk := false
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingDeploymentKind(t *testing.T) {
	exclude := map[string]string{
		"deployments": "not-a-match",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		RootOwner: map[string]string{
			"kind": "Deployment",
			"name": "match",
		},
	}

	expectedFilter := ""
	expectedOk := false
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingDeploymentName(t *testing.T) {
	exclude := map[string]string{
		"deployments": "match",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		RootOwner: map[string]string{
			"kind": "NotADeployment",
			"name": "match",
		},
	}

	expectedFilter := ""
	expectedOk := false
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingDeploymentKindAndName(t *testing.T) {
	exclude := map[string]string{
		"deployments": "match",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		RootOwner: map[string]string{
			"kind": "Deployment",
			"name": "match",
		},
	}

	expectedFilter := "deployments:match"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithUnsupportedKind(t *testing.T) {
	exclude := map[string]string{
		"unsupported": "kind",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{}

	expectedFilter := ""
	expectedOk := false
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingDeploymentAndNonMatchingNamespace(t *testing.T) {
	exclude := map[string]string{
		"deployments": "match-deployment",
		"namespaces":  "not-match-namespace",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match-namespace",
		RootOwner: map[string]string{
			"kind": "Deployment",
			"name": "match-deployment",
		},
	}

	expectedFilter := "deployments:match-deployment"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithNonMatchingDeploymentAndMatchingNamespace(t *testing.T) {
	exclude := map[string]string{
		"deployments": "not-match-deployment",
		"namespaces":  "match-namespace",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match-namespace",
		RootOwner: map[string]string{
			"kind": "Deployment",
			"name": "match-deployment",
		},
	}

	expectedFilter := "namespaces:match-namespace"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingDeploymentAndMatchingNamespace(t *testing.T) {
	exclude := map[string]string{
		"deployments": "match-deployment",
		"namespaces":  "match-namespace",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match-namespace",
		RootOwner: map[string]string{
			"kind": "Deployment",
			"name": "match-deployment",
		},
	}

	// Expectation is to always return the least specific match
	expectedFilter := "namespaces:match-namespace"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithNoMatch(t *testing.T) {
	kubernetesConfig := NewKubernetesConfig()
	kubernetesContext := &KubernetesContext{}

	expectedFilter := ""
	expectedOk := false
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingNamespaceCommaSeparated(t *testing.T) {
	exclude := map[string]string{
		"namespaces": "not-match-namespace,match-namespace",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match-namespace",
	}

	// Expectation is to return the least specific match
	expectedFilter := "namespaces:match-namespace"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}

func TestKubernetesConfigApplyFilterWithMatchingNamespaceRegExp(t *testing.T) {
	exclude := map[string]string{
		"namespaces": "^match.*$",
	}
	kubernetesConfig := &KubernetesConfig{
		Exclude: exclude,
	}
	kubernetesContext := &KubernetesContext{
		Namespace: "match-namespace",
	}

	// Expectation is to return the least specific match
	expectedFilter := "namespaces:^match.*$"
	expectedOk := true
	filter, ok := kubernetesConfig.ApplyFilter(kubernetesContext)

	if filter != expectedFilter {
		t.Errorf("Expected filter to be %s, got %s", expectedFilter, filter)
	}

	if ok != expectedOk {
		t.Errorf("Expected ok to be %t, got %t", expectedOk, ok)
	}
}
