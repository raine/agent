package main

var schema string = "https://raw.githubusercontent.com/timberio/log-event-json-schema/v2.4.2/schema.json"

type LogEvent struct {
	Schema  string  `json:"$schema"`
	Context Context `json:"context,omitempty"`
}

type Context struct {
	System SystemContext `json:"system,omitempty"`
}

type SystemContext struct {
	Hostname string `json:"hostname,omitempty"`
}

func NewLogEvent() *LogEvent {
	return &LogEvent{Schema: schema}
}
