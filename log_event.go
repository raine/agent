package main

import (
	"encoding/json"

	"github.com/mitchellh/copystructure"
)

var schema string = "https://raw.githubusercontent.com/timberio/log-event-json-schema/v4.1.0/schema.json"

type LogEvent struct {
	Schema  string   `json:"$schema"`
	Context *Context `json:"context,omitempty"`
}

type Context struct {
	System   *SystemContext   `json:"system,omitempty"`
	Platform *PlatformContext `json:"platform,omitempty"`
	Source   *SourceContext   `json:"source,omitempty"`
}

type SystemContext struct {
	Hostname string `json:"hostname,omitempty"`
}

type PlatformContext struct {
	AWSEC2     *AWSEC2Context     `json:"aws_ec2,omitempty"`
	Kubernetes *KubernetesContext `json:"kubernetes,omitempty"`
}

type SourceContext struct {
	FileName string `json:"file_name,omitempty"`
}

type AWSEC2Context struct {
	AmiID          string `json:"ami_id,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	InstanceID     string `json:"instance_id,omitempty"`
	InstanceType   string `json:"instance_type,omitempty"`
	PublicHostname string `json:"public_hostname,omitempty"`
}

type KubernetesContext struct {
	ContainerName string            `json:"container_name,omitempty"`
	PodName       string            `json:"pod_name,omitempty"`
	Namespace     string            `json:"namespace,omitempty"`
	RootOwner     map[string]string `json:"root_owner,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

func NewLogEvent() *LogEvent {
	return &LogEvent{Schema: schema}
}

func (logEvent *LogEvent) EncodeJSON() ([]byte, error) {
	return json.Marshal(logEvent)
}

// DeepCopy Returns a deep copy of caller *LogEvent or nil if copy fails.
func (logEvent *LogEvent) DeepCopy() *LogEvent {
	value, err := copystructure.Copy(logEvent)
	if err != nil {
		return nil
	}

	logEventCopy, ok := value.(*LogEvent)
	if !ok {
		return nil
	}

	return logEventCopy
}

func (logEvent *LogEvent) AddEC2Context(context *AWSEC2Context) {
	logEvent.ensurePlatformContext()
	logEvent.Context.Platform.AWSEC2 = context
}

func (logEvent *LogEvent) AddKubernetesContext(context *KubernetesContext) {
	logEvent.ensurePlatformContext()
	logEvent.Context.Platform.Kubernetes = context
}

func (logEvent *LogEvent) ensureContext() {
	if logEvent.Context == nil {
		logEvent.Context = &Context{}
	}
}

func (logEvent *LogEvent) ensurePlatformContext() {
	logEvent.ensureContext()
	if logEvent.Context.Platform == nil {
		logEvent.Context.Platform = &PlatformContext{}
	}
}

func (logEvent *LogEvent) ensureSystemContext() {
	logEvent.ensureContext()
	if logEvent.Context.System == nil {
		logEvent.Context.System = &SystemContext{}
	}
}

func (logEvent *LogEvent) ensureSourceContext() {
	logEvent.ensureContext()
	if logEvent.Context.Source == nil {
		logEvent.Context.Source = &SourceContext{}
	}
}
